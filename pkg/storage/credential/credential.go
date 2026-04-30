// Package credential holds ergonomic builders for the StorageCredential*
// variants emitted by the OpenAPI generator under managementv1.
//
// Each provider has one constructor per credential variant (S3 access-key /
// AWS system-identity / Cloudflare R2; GCS service-account-key / GCP
// system-identity; Azure client-credentials / shared-access-key /
// managed-identity). Constructors take only the spec-required fields; rare
// optional fields like S3 ExternalId can be set on the returned struct.
//
// Builders return the typed concrete struct (e.g. *managementv1.StorageCredentialS3).
// Wrap into the union with the generator-emitted helpers when handing off:
//
//	c := credential.NewS3AccessKey("AKIA...", "secret")
//	req.SetStorageCredential(managementv1.StorageCredentialS3AsStorageCredential(c))
//
// Wire-format note: the generator collapses each provider's oneOf credential
// variants into a single flattened struct (e.g. StorageCredentialS3 holds the
// union of access-key, system-identity, and Cloudflare R2 fields). Builders
// populate only the fields relevant to the chosen variant — the rest serialize
// as zero values. If Lakekeeper rejects the resulting payload during
// integration testing, the workaround is to add a custom MarshalJSON on a
// thin wrapper type returned from these constructors.
package credential

const (
	// Outer Type discriminator on StorageCredential variants.
	typeS3  = "s3"
	typeGCS = "gcs"
	typeAZ  = "az"

	// CredentialType discriminator values, per the spec's per-provider enums.
	credS3AccessKey         = "access-key"
	credS3AwsSystemIdentity = "aws-system-identity"
	credS3CloudflareR2      = "cloudflare-r2"
	credGCSServiceAccount   = "service-account-key"
	credGCSSystemIdentity   = "gcp-system-identity"
	credAZClient            = "client-credentials"
	credAZSharedAccessKey   = "shared-access-key"
	credAZManagedIdentity   = "azure-system-identity"
)
