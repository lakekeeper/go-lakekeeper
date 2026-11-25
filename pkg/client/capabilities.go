package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Edition is which build a server runs: Lakekeeper or Lakekeeper+. The
// server reports it in its server info.
type Edition string

const (
	// EditionLakekeeper is the open-source Lakekeeper.
	EditionLakekeeper Edition = "Lakekeeper"
	// EditionLakekeeperPlus is the commercial Lakekeeper+, a superset of
	// Lakekeeper's API.
	EditionLakekeeperPlus Edition = "Lakekeeper+"
)

// AuthzBackend is the authorization backend a server reports as its
// authz-backend.
type AuthzBackend string

// Authorization backends.
const (
	AuthzBackendCedar   AuthzBackend = "cedar"
	AuthzBackendOpenFGA AuthzBackend = "openfga"
)

// Capabilities describes what one server can do: its edition, versions,
// authorization backend and the task queues it runs. Client.Capabilities
// builds it from the server info.
type Capabilities struct {
	Edition Edition
	// LakekeeperVersion is the Lakekeeper version the server is built on,
	// for either edition.
	LakekeeperVersion string
	// LakekeeperPlusVersion is the Lakekeeper+ version; empty on Lakekeeper.
	LakekeeperPlusVersion string
	// AuthzBackend is the authorization backend, for example
	// AuthzBackendCedar or AuthzBackendOpenFGA.
	AuthzBackend AuthzBackend
	// Queues are the task queues the server runs.
	Queues []Queue
}

// String names the server by edition and that edition's version, for
// example "Lakekeeper 0.15.1" or "Lakekeeper+ 0.15.1".
func (c *Capabilities) String() string {
	return string(c.Edition) + " " + c.version(c.Edition)
}

// version is the server's version for edition: the Lakekeeper+ version for
// Lakekeeper+, the Lakekeeper version it is built on otherwise.
func (c *Capabilities) version(edition Edition) string {
	if edition == EditionLakekeeperPlus {
		return c.LakekeeperPlusVersion
	}
	return c.LakekeeperVersion
}

// newCapabilities builds the capabilities from the server info's fields.
// lakekeeper-enterprise-version, null on Lakekeeper, decides the edition.
func newCapabilities(lakekeeperVersion, enterpriseVersion, authzBackend string, queues []string) *Capabilities {
	caps := &Capabilities{
		Edition:           EditionLakekeeper,
		LakekeeperVersion: lakekeeperVersion,
		AuthzBackend:      AuthzBackend(authzBackend),
	}
	for _, q := range queues {
		caps.Queues = append(caps.Queues, Queue(q))
	}
	if enterpriseVersion != "" {
		caps.Edition = EditionLakekeeperPlus
		caps.LakekeeperPlusVersion = enterpriseVersion
	}
	return caps
}

func capabilitiesFromServerInfo(info *managementv1.ServerInfo) *Capabilities {
	return newCapabilities(info.GetLakekeeperVersion(), info.GetLakekeeperEnterpriseVersion(),
		info.GetAuthzBackend(), info.GetQueues())
}

// capabilitiesFromBody reads the capabilities from a server info body the
// generated model could not decode. It returns nil when the body names no
// Lakekeeper version.
func capabilitiesFromBody(body []byte) *Capabilities {
	var info struct {
		LakekeeperVersion           string   `json:"lakekeeper-version"`
		LakekeeperEnterpriseVersion string   `json:"lakekeeper-enterprise-version"`
		AuthzBackend                string   `json:"authz-backend"`
		Queues                      []string `json:"queues"`
	}
	if json.Unmarshal(body, &info) != nil {
		return nil
	}
	if info.LakekeeperVersion == "" {
		return nil
	}
	return newCapabilities(info.LakekeeperVersion, info.LakekeeperEnterpriseVersion, info.AuthzBackend, info.Queues)
}

// Capabilities returns what the server can do. The server info behind it is
// fetched on the first call (or the first diagnosed failure of a gated
// façade) and cached for the life of the Client; a failed fetch is not
// cached. Each call returns a fresh copy.
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	return c.capabilities.get(ctx)
}

// CachedCapabilities returns the capabilities an earlier server info fetch
// read (Server.Info, WithInitialBootstrap, Capabilities, or the diagnosis of
// a gated façade's failure), and false when none has. It sends no request,
// so tooling can, for example, warn about a server outside
// SupportedVersions without an extra round trip, as lkctl does. It returns
// a fresh copy.
func (c *Client) CachedCapabilities() (*Capabilities, bool) {
	return c.capabilities.cached()
}

// capabilityCache fetches the server's capabilities once per Client. The
// façades that gate operations on a capability share it, and Server.Info
// fills it on every fetch.
type capabilityCache struct {
	server *Server

	mu   sync.Mutex
	caps *Capabilities
}

func (cc *capabilityCache) get(ctx context.Context) (*Capabilities, error) {
	// Held across the fetch, so concurrent first calls fetch once.
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.caps == nil {
		_, caps, err := cc.server.fetchInfo(ctx)
		if caps == nil {
			return nil, fmt.Errorf("capabilities: %w", err)
		}
		cc.caps = caps
	}
	return cc.caps.clone(), nil
}

func (cc *capabilityCache) cached() (*Capabilities, bool) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.caps == nil {
		return nil, false
	}
	return cc.caps.clone(), true
}

// record stores caps read by a server info fetch outside the cache.
func (cc *capabilityCache) record(caps *Capabilities) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.caps = caps
}

func (c *Capabilities) clone() *Capabilities {
	out := *c
	out.Queues = slices.Clone(c.Queues)
	return &out
}

// requirement is what a capability-gated façade operation needs from the
// server. Each gated façade declares its requirement once, next to the
// façade type (cedarRequirement for Cedar), and passes it to diagnose.
type requirement struct {
	// feature names what the operation belongs to, for UnsupportedError.
	feature string
	// lakekeeperPlus is set when the operation exists on Lakekeeper+ only.
	lakekeeperPlus bool
	// authzBackend, when set, is the authorization backend it needs.
	authzBackend AuthzBackend
	// queue, when set, is the task queue it needs the server to run.
	queue Queue
}

// unmet returns why caps do not meet r, or "" when they do. apiErr is the
// server's failed answer, nil for a successful one; it decides whether an
// older server counts as too old.
func (r requirement) unmet(caps *Capabilities, apiErr *APIError) (UnsupportedReason, string) {
	if r.lakekeeperPlus && caps.Edition != EditionLakekeeperPlus {
		return ReasonNotLakekeeperPlus, "requires Lakekeeper+; server is " + caps.String()
	}
	if r.authzBackend != "" && caps.AuthzBackend != r.authzBackend {
		return ReasonAuthzBackend, fmt.Sprintf(
			"requires the %s authorization backend; server is %s with the %s authorization backend",
			r.authzBackend, caps, caps.AuthzBackend)
	}
	if r.queue != "" && !slices.Contains(caps.Queues, r.queue) {
		return ReasonQueueNotConfigured, fmt.Sprintf(
			"requires the %s task queue; server is %s without it", r.queue, caps)
	}
	if release, ok := r.newerThan(caps, apiErr); ok {
		return ReasonServerTooOld, fmt.Sprintf("requires %s %s or later; server is %s", r.specEdition(), release, caps)
	}
	return "", ""
}

// specEdition is the edition whose spec the operation came from: a
// Lakekeeper+-only operation came from the Lakekeeper+ spec.
func (r requirement) specEdition() Edition {
	if r.lakekeeperPlus {
		return EditionLakekeeperPlus
	}
	return EditionLakekeeper
}

// newerThan reports whether the release of the spec the operation came from
// is newer than the server's version for that edition, and returns the
// release. Only an answer without a Lakekeeper error body counts: that is
// how a server answers a route it does not know, while a 404 with an error
// body is a genuine not-found, which an older server sends too.
func (r requirement) newerThan(caps *Capabilities, apiErr *APIError) (string, bool) {
	if apiErr == nil || apiErr.Model != nil {
		return "", false
	}
	release := specRelease(r.specEdition())
	have, ok := parseVersion(caps.version(r.specEdition()))
	want, wantOK := parseVersion(release)
	if !ok || !wantOK || !have.less(want) {
		return "", false
	}
	return release, true
}

// mayExplain reports whether the server's answer is one the capabilities
// may explain: 404 or 405, the answers to an unknown operation, and, for a
// task-queue operation, a 400 QueueNotFound, which Lakekeeper sends for a
// queue it does not know when setting a queue configuration.
func (r requirement) mayExplain(apiErr *APIError) bool {
	switch apiErr.StatusCode {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return true
	case http.StatusBadRequest:
		return r.queue != "" && apiErr.Model != nil && apiErr.Model.GetType() == "QueueNotFound"
	}
	return false
}

// diagnose finishes a gated façade call. Successful calls (err == nil) and
// failures the capabilities cannot explain (see mayExplain) pass through
// without a request. For the others (404, 405, and a task queue's 400
// QueueNotFound) it consults the cached capabilities: when they explain the failure
// it returns an *UnsupportedError, otherwise (a genuine not-found, or the
// capabilities cannot be fetched) the original error unchanged.
func (cc *capabilityCache) diagnose(ctx context.Context, req requirement, op string, err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !req.mayExplain(apiErr) {
		return err
	}
	caps, capsErr := cc.get(ctx)
	if capsErr != nil {
		return err
	}
	reason, sentence := req.unmet(caps, apiErr)
	if reason == "" {
		return err
	}
	return &UnsupportedError{
		Feature:  req.feature,
		Reason:   reason,
		Server:   *caps,
		APIError: apiErr,
		op:       op,
		sentence: sentence,
	}
}

// ErrUnsupported matches every *UnsupportedError with errors.Is: the server
// does not offer the operation, and its capabilities say why.
var ErrUnsupported = errors.New("unsupported by the server")

// UnsupportedReason says why a server does not offer an operation.
type UnsupportedReason string

const (
	// ReasonNotLakekeeperPlus: the operation needs Lakekeeper+ and the
	// server runs Lakekeeper.
	ReasonNotLakekeeperPlus UnsupportedReason = "not-lakekeeper-plus"
	// ReasonAuthzBackend: the operation needs another authorization
	// backend than the server runs.
	ReasonAuthzBackend UnsupportedReason = "authz-backend"
	// ReasonQueueNotConfigured: the operation needs a task queue the
	// server does not run.
	ReasonQueueNotConfigured UnsupportedReason = "queue-not-configured"
	// ReasonServerTooOld: the server answers the operation as an unknown
	// route and its version, for the edition whose spec the operation came
	// from, is older than that spec's release.
	ReasonServerTooOld UnsupportedReason = "server-too-old"
)

// UnsupportedError is the error a capability-gated façade returns when the
// server rejects the operation as unknown (404 or 405, or a task queue's
// 400 QueueNotFound or empty configuration) and the server's
// capabilities explain why. Match it with errors.Is(err, ErrUnsupported)
// or errors.As(err, &unsupported).
//
// It does not unwrap to the server's *APIError, so a caller that treats a
// 404 APIError as "resource gone" does not mistake an unsupported
// operation for a missing resource. The APIError is kept in the APIError
// field.
type UnsupportedError struct {
	// Feature names what the operation belongs to, for example "Cedar".
	Feature string
	// Reason says why the server does not offer it.
	Reason UnsupportedReason
	// Server is the server's capabilities.
	Server Capabilities
	// APIError is the server's original answer; nil when the server
	// answered with success, as Lakekeeper 0.15.0 does to a Lakekeeper+
	// queue configuration read (see TaskQueues).
	APIError *APIError

	op       string
	sentence string
}

// Error reports "<op>: <requirement>; server is <server>", for example
// "get cedar schema: requires Lakekeeper+; server is Lakekeeper 0.15.1".
func (e *UnsupportedError) Error() string {
	return e.op + ": " + e.sentence
}

// Is reports whether target is ErrUnsupported.
func (e *UnsupportedError) Is(target error) bool {
	return target == ErrUnsupported
}
