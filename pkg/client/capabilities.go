package client

import (
	"context"
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

// Authorization backends a server reports as its authz-backend.
const (
	AuthzBackendCedar   = "cedar"
	AuthzBackendOpenFGA = "openfga"
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
	AuthzBackend string
	// Queues are the task queues the server runs.
	Queues []string
}

// String names the server by edition and that edition's version, for
// example "Lakekeeper 0.15.1" or "Lakekeeper+ 0.15.1".
func (c *Capabilities) String() string {
	if c.Edition == EditionLakekeeperPlus {
		return string(c.Edition) + " " + c.LakekeeperPlusVersion
	}
	return string(c.Edition) + " " + c.LakekeeperVersion
}

func capabilitiesFromServerInfo(info *managementv1.ServerInfo) *Capabilities {
	caps := &Capabilities{
		Edition:           EditionLakekeeper,
		LakekeeperVersion: info.GetLakekeeperVersion(),
		AuthzBackend:      info.GetAuthzBackend(),
		Queues:            slices.Clone(info.GetQueues()),
	}
	// lakekeeper-enterprise-version is null on Lakekeeper.
	if v := info.GetLakekeeperEnterpriseVersion(); v != "" {
		caps.Edition = EditionLakekeeperPlus
		caps.LakekeeperPlusVersion = v
	}
	return caps
}

// Capabilities returns what the server can do. The server info behind it is
// fetched on the first call (or the first diagnosed failure of a gated
// façade) and cached for the life of the Client; a failed fetch is not
// cached. Each call returns a fresh copy.
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	return c.capabilities.get(ctx)
}

// capabilityCache fetches the server's capabilities once per Client. The
// façades that gate operations on a capability share it.
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
		info, err := cc.server.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("capabilities: %w", err)
		}
		cc.caps = capabilitiesFromServerInfo(info)
	}
	caps := *cc.caps
	caps.Queues = slices.Clone(cc.caps.Queues)
	return &caps, nil
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
	authzBackend string
	// queue, when set, is the task queue it needs the server to run, as
	// the server info names it in its queues.
	queue string
}

// unmet returns why caps do not meet r, or "" when they do.
func (r requirement) unmet(caps *Capabilities) (UnsupportedReason, string) {
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
	return "", ""
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
	reason, sentence := req.unmet(caps)
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
)

// UnsupportedError is the error a capability-gated façade returns when the
// server rejects the operation as unknown (404 or 405) and the server's
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
	// APIError is the server's original answer.
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
