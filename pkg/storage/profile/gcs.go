package profile

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// GCSOption customises a GCS profile during construction.
type GCSOption func(*managementv1.StorageProfileGcs)

// NewGCSProfile constructs a StorageProfileGcs with the spec-required field
// (bucket) populated. Apply With* options for anything else.
//
// Returns the StorageProfile union directly so callers can pass it to
// request setters (e.g. CreateWarehouseRequest.StorageProfile) without going
// through the generated *AsStorageProfile wrapper.
func NewGCSProfile(bucket string, opts ...GCSOption) managementv1.StorageProfile {
	p := &managementv1.StorageProfileGcs{
		Bucket: bucket,
		Type:   typeGCS,
	}
	for _, opt := range opts {
		opt(p)
	}
	return managementv1.StorageProfileGcsAsStorageProfile(p)
}

func WithGCSKeyPrefix(prefix string) GCSOption {
	return func(p *managementv1.StorageProfileGcs) { p.KeyPrefix = &prefix }
}

// WithGCSSTSEnabled lets the caller explicitly set STS-vended-credentials on
// or off. The spec default is true, so passing false is the typical use here.
func WithGCSSTSEnabled(enabled bool) GCSOption {
	return func(p *managementv1.StorageProfileGcs) { p.StsEnabled = &enabled }
}

func WithGCSStorageLayout(layout StorageLayout) GCSOption {
	return func(p *managementv1.StorageProfileGcs) { p.StorageLayout = &layout }
}
