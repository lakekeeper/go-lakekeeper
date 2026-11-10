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

// CallOption configures a single façade call. Project-scoped façade methods
// take a trailing variadic ...CallOption. Omitting it lets the server apply
// its default project, except on the destructive project-level methods
// (Projects.Delete, Permissions.Projects.Grant and Revoke), which return
// ErrProjectRequired without it.
type CallOption func(*callOptions)

type callOptions struct {
	projectID  string
	projectSet bool
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
// project-scoped façade method routes its builder through it, so how a
// project is sent is decided here only.
func withProject[R projectScoped[R]](op string, req R, opts []CallOption) (R, error) {
	o := resolveCallOptions(opts)
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
