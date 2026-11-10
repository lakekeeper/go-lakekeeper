package client

import (
	"errors"
	"fmt"
)

// ErrEmptyProjectID is returned by a project-scoped façade method called
// with WithProject(""), before any request is sent.
var ErrEmptyProjectID = errors.New("project id must not be empty")

// ErrProjectIDConflict is returned, before any request is sent, when a
// request model's deprecated project-id body field names a different
// Project than WithProject.
var ErrProjectIDConflict = errors.New("project id in the request body does not match WithProject")

// CallOption configures a single façade call. Project-scoped façade methods
// take a trailing variadic ...CallOption; omitting it is always valid.
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

// projectFromBody reconciles a request model's deprecated project-id body
// field with opts, so the project is sent as the header like everywhere
// else. A body project becomes WithProject; if opts already name a different
// one, it returns ErrProjectIDConflict. The caller must still unset the field
// on its copy of the body, so it is never sent.
func projectFromBody(op, bodyProjectID string, opts []CallOption) ([]CallOption, error) {
	if bodyProjectID == "" {
		return opts, nil
	}
	o := resolveCallOptions(opts)
	switch {
	case !o.projectSet:
		return []CallOption{WithProject(bodyProjectID)}, nil
	case o.projectID == "":
		return opts, nil // withProject reports ErrEmptyProjectID
	case o.projectID != bodyProjectID:
		return nil, fmt.Errorf("%s: %w: body %q, WithProject %q", op, ErrProjectIDConflict, bodyProjectID, o.projectID)
	default:
		return opts, nil
	}
}
