package client

// CallOption configures a single façade call. Project-scoped façade methods
// take a trailing variadic ...CallOption; omitting it is always valid.
type CallOption func(*callOptions)

type callOptions struct {
	projectID string
}

// WithProject names the Project a project-scoped call runs in. The façade
// sends it as the x-project-id header, never as the deprecated projectId
// query parameter or body field.
//
// Without WithProject, or with an empty id, no project is sent and the server
// applies its default project (or answers 400 ProjectIdMissing when the
// operator disabled it). If several are given, the last one wins.
func WithProject(projectID string) CallOption {
	return func(o *callOptions) { o.projectID = projectID }
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
// the x-project-id header when a project is named, nothing otherwise. Every
// project-scoped façade method routes its builder through it, so how a
// project is sent is decided here only.
func withProject[R projectScoped[R]](req R, opts []CallOption) R {
	if o := resolveCallOptions(opts); o.projectID != "" {
		return req.XProjectId(o.projectID)
	}
	return req
}
