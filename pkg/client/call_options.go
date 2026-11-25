package client

import (
	"errors"
	"fmt"
	"slices"
)

// ErrEmptyProjectID is returned by a project-scoped façade method called
// with WithProject(""), before any request is sent.
var ErrEmptyProjectID = errors.New("project id must not be empty")

// ErrProjectRequired is returned, before any request is sent, by a
// destructive project-level method (Projects.Delete, and Grant and Revoke on
// Permissions.Projects) called without WithProject. These never fall back to
// the server's default project.
var ErrProjectRequired = errors.New("project must be named with WithProject")

// ErrProjectIDConflict is returned, before any request is sent, when a
// request model's deprecated project-id body field names a different
// Project than WithProject.
var ErrProjectIDConflict = errors.New("project id in the request body does not match WithProject")

// ErrEmptyBreakGlassReason is returned by a façade method called with
// WithBreakGlass(""), before any request is sent.
var ErrEmptyBreakGlassReason = errors.New("break-glass reason must not be empty")

// ErrCallOptionNotAccepted is returned, before any request is sent, when a
// façade method is given a CallOption its operation cannot send: a
// WithBreakGlass to an operation without the x-break-glass header, or a
// WithProject to a warehouse-scoped operation. The option is never silently
// dropped.
var ErrCallOptionNotAccepted = errors.New("call option not accepted by this operation")

// CallOption configures a single façade call. Project-scoped façade methods
// take a trailing variadic ...CallOption. Omitting it lets the server apply
// its default project, except on the destructive project-level methods
// (Projects.Delete, Permissions.Projects.Grant and Revoke), which return
// ErrProjectRequired without it.
type CallOption func(*callOptions)

type callOptions struct {
	projectID  string
	projectSet bool

	breakGlassReason string
	breakGlassSet    bool
}

// WithProject names the Project a project-scoped call runs in. The façade
// sends it as the x-project-id header, never as the deprecated projectId
// query parameter or body field.
//
// Without WithProject, no project is sent and the server applies its default
// project (or answers 400 ProjectIdMissing when the operator disabled it).
// WithProject("") is not the same: the call fails with ErrEmptyProjectID
// before sending anything, so an unset variable can't silently target the
// default project. If several are given, the last one wins.
func WithProject(projectID string) CallOption {
	return func(o *callOptions) {
		o.projectID = projectID
		o.projectSet = true
	}
}

// WithBreakGlass asks for emergency access (break-glass) on a Lakekeeper+
// Cedar operation, with reason recorded by the server. The façade sends it as
// the x-break-glass header. Only the operations whose spec accepts that
// header take it (listing and applying Cedar scope policies); any other
// façade method fails with ErrCallOptionNotAccepted. WithBreakGlass("")
// fails with ErrEmptyBreakGlassReason. If several are given, the last one
// wins.
func WithBreakGlass(reason string) CallOption {
	return func(o *callOptions) {
		o.breakGlassReason = reason
		o.breakGlassSet = true
	}
}

func resolveCallOptions(opts []CallOption) callOptions {
	var o callOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// projectScoped is any generated request builder that can carry the
// x-project-id header.
type projectScoped[R any] interface {
	XProjectId(projectID string) R
}

// withProject applies the Project from opts to a generated request builder:
// the x-project-id header when a project is named, nothing when none is, and
// ErrEmptyProjectID (wrapped with op) when an empty one is. Every
// project-scoped façade method routes its builder through it (or, when it
// also accepts break-glass, through withProjectAndBreakGlass), so how a
// project is sent is decided in applyProject only. A WithBreakGlass fails
// with ErrCallOptionNotAccepted.
func withProject[R projectScoped[R]](op string, req R, opts []CallOption) (R, error) {
	o := resolveCallOptions(opts)
	if o.breakGlassSet {
		return req, notAccepted(op, "WithBreakGlass")
	}
	return applyProject(op, req, o)
}

func applyProject[R projectScoped[R]](op string, req R, o callOptions) (R, error) {
	switch {
	case !o.projectSet:
		return req, nil
	case o.projectID == "":
		return req, fmt.Errorf("%s: %w", op, ErrEmptyProjectID)
	default:
		return req.XProjectId(o.projectID), nil
	}
}

// withRequiredProject is withProject for the destructive project-level
// methods: without a project option it returns ErrProjectRequired (wrapped
// with op) instead of leaving the project to the server.
func withRequiredProject[R projectScoped[R]](op string, req R, opts []CallOption) (R, error) {
	if !resolveCallOptions(opts).projectSet {
		return req, fmt.Errorf("%s: %w", op, ErrProjectRequired)
	}
	return withProject(op, req, opts)
}

// bodyProjectModel is a generated request model that still carries the
// deprecated project-id body field (CreateWarehouseRequest,
// CreateRoleRequest, SearchRoleRequest).
type bodyProjectModel[T any] interface {
	*T
	GetProjectId() string
	UnsetProjectId()
}

// withBodyProject is withProject for a request whose body model still has the
// deprecated project-id field. The body project is sent as the header like
// everywhere else: alone it acts like WithProject, equal to it it is dropped,
// and a different one fails with ErrProjectIDConflict. It returns the
// configured builder and a copy of body with the field unset, so the field is
// never sent and the caller's body is not modified.
func withBodyProject[R projectScoped[R], T any, PT bodyProjectModel[T]](op string, req R, body *T, opts []CallOption) (R, T, error) {
	cp := *body
	if bodyProjectID := PT(&cp).GetProjectId(); bodyProjectID != "" {
		o := resolveCallOptions(opts)
		switch {
		case !o.projectSet:
			opts = append(slices.Clip(opts), WithProject(bodyProjectID))
		case o.projectID != "" && o.projectID != bodyProjectID:
			return req, cp, fmt.Errorf("%s: %w: body %q, WithProject %q", op, ErrProjectIDConflict, bodyProjectID, o.projectID)
		}
		// An empty WithProject is left to withProject (ErrEmptyProjectID).
	}
	PT(&cp).UnsetProjectId()
	req, err := withProject(op, req, opts)
	return req, cp, err
}

// breakGlassable is any generated request builder that can carry the
// x-break-glass header.
type breakGlassable[R any] interface {
	XBreakGlass(reason string) R
}

// applyBreakGlass sends the x-break-glass header when WithBreakGlass was
// given, and fails with ErrEmptyBreakGlassReason (wrapped with op) on an
// empty reason.
func applyBreakGlass[R breakGlassable[R]](op string, req R, o callOptions) (R, error) {
	switch {
	case !o.breakGlassSet:
		return req, nil
	case o.breakGlassReason == "":
		return req, fmt.Errorf("%s: %w", op, ErrEmptyBreakGlassReason)
	default:
		return req.XBreakGlass(o.breakGlassReason), nil
	}
}

// withProjectAndBreakGlass is withProject for a project-scoped operation that
// also accepts the x-break-glass header.
func withProjectAndBreakGlass[R interface {
	projectScoped[R]
	breakGlassable[R]
}](op string, req R, opts []CallOption,
) (R, error) {
	o := resolveCallOptions(opts)
	req, err := applyProject(op, req, o)
	if err != nil {
		return req, err
	}
	return applyBreakGlass(op, req, o)
}

// withBreakGlass applies the options of a warehouse-scoped operation that
// accepts the x-break-glass header. The warehouse names its project, so a
// WithProject fails with ErrCallOptionNotAccepted.
func withBreakGlass[R breakGlassable[R]](op string, req R, opts []CallOption) (R, error) {
	o := resolveCallOptions(opts)
	if o.projectSet {
		return req, notAccepted(op, "WithProject")
	}
	return applyBreakGlass(op, req, o)
}

func notAccepted(op, option string) error {
	return fmt.Errorf("%s: %w: %s", op, ErrCallOptionNotAccepted, option)
}
