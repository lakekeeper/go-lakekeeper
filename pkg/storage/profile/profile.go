// Package profile holds ergonomic builders for the StorageProfile* variants
// emitted by the OpenAPI generator under managementv1.
//
// Each provider has a constructor (NewS3Profile, NewGCSProfile, NewADLSProfile)
// that takes the required spec-mandated fields positionally and accepts a
// variadic list of With*-style options for the optional ones.
//
// Builders return the typed concrete struct (e.g. *managementv1.StorageProfileS3).
// Callers wrap into the union with the generator-emitted helpers when handing
// off to the API:
//
//	p := profile.NewS3Profile("my-bucket", "us-east-1",
//	    profile.WithS3Endpoint("http://minio:9000"),
//	    profile.WithS3STSEnabled(),
//	)
//	req.SetStorageProfile(managementv1.StorageProfileS3AsStorageProfile(p))
package profile

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// Discriminator string values for the StorageProfile oneOf, taken from the
// spec. Kept here so callers don't need to remember the magic strings, and so
// a regen that changes them surfaces as a build-time mismatch.
const (
	typeS3   = "s3"
	typeGCS  = "gcs"
	typeADLS = "adls"
)

// ptrTo is a small helper used internally to take the address of a literal.
// Exported types from managementv1 use *T for optional scalars.
func ptrTo[T any](v T) *T { return &v }

// Re-export the managementv1 enum types that callers commonly need to pass
// into options. Avoids forcing a managementv1 import for a single value.
type (
	S3Flavor                = managementv1.S3Flavor
	S3UrlStyleDetectionMode = managementv1.S3UrlStyleDetectionMode
	StorageLayout           = managementv1.StorageLayout
)
