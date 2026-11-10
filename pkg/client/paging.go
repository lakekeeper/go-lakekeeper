package client

// paged is any generated list request builder that takes page size and
// page token query parameters.
type paged[R any] interface {
	PageSize(pageSize int64) R
	PageToken(pageToken string) R
}

// withPaging applies the PageSize and PageToken of a list method's options
// to its request builder. A zero value sends nothing for that parameter.
func withPaging[R paged[R]](req R, pageSize int64, pageToken string) R {
	if pageSize != 0 {
		req = req.PageSize(pageSize)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	return req
}
