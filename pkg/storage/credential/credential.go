// Package credential holds ergonomic builders for storage credentials.
//
// Each provider has one constructor per credential variant (S3 access-key /
// AWS system-identity / Cloudflare R2; GCS service-account-key / GCP
// system-identity; Azure client-credentials / shared-access-key /
// managed-identity). Constructors take only the spec-required fields; rare
// optional fields like S3 external-id have their own _WithExternalID
// variants.
//
// Builders return a managementv1.StorageCredential umbrella value — the
// shape API methods accept directly:
//
//	c := credential.NewS3AccessKey("AKIA...", "secret")
//	req.SetStorageCredential(c)
//
// Internally each builder constructs the correct generated leaf type and
// wraps it via the generator-emitted *AsStorageCredential helper. The wire
// format contains only the variant's own fields plus the discriminators —
// no zero-valued fields from sibling variants leak through, and inbound
// payloads from the server round-trip cleanly.
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
