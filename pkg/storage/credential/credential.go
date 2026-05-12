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

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// AsS3AccessKey returns the access-key variant of a StorageCredential, or
// (nil, false) if the union holds a different variant.
func AsS3AccessKey(sc managementv1.StorageCredential) (*managementv1.StorageCredentialAccessKey, bool) {
	if sc.StorageCredentialAccessKey == nil {
		return nil, false
	}
	return sc.StorageCredentialAccessKey, true
}

// AsS3AwsSystemIdentity returns the AWS-system-identity variant, or
// (nil, false) if the union holds a different variant.
func AsS3AwsSystemIdentity(sc managementv1.StorageCredential) (*managementv1.StorageCredentialAwsSystemIdentity, bool) {
	if sc.StorageCredentialAwsSystemIdentity == nil {
		return nil, false
	}
	return sc.StorageCredentialAwsSystemIdentity, true
}

// AsS3CloudflareR2 returns the Cloudflare R2 variant, or (nil, false) if
// the union holds a different variant.
func AsS3CloudflareR2(sc managementv1.StorageCredential) (*managementv1.StorageCredentialCloudflareR2, bool) {
	if sc.StorageCredentialCloudflareR2 == nil {
		return nil, false
	}
	return sc.StorageCredentialCloudflareR2, true
}

// AsGCSServiceAccountKey returns the GCS service-account-key variant, or
// (nil, false) if the union holds a different variant.
func AsGCSServiceAccountKey(sc managementv1.StorageCredential) (*managementv1.StorageCredentialServiceAccountKey, bool) {
	if sc.StorageCredentialServiceAccountKey == nil {
		return nil, false
	}
	return sc.StorageCredentialServiceAccountKey, true
}

// AsGCSSystemIdentity returns the GCP-system-identity variant, or
// (nil, false) if the union holds a different variant.
func AsGCSSystemIdentity(sc managementv1.StorageCredential) (*managementv1.StorageCredentialGcpSystemIdentity, bool) {
	if sc.StorageCredentialGcpSystemIdentity == nil {
		return nil, false
	}
	return sc.StorageCredentialGcpSystemIdentity, true
}

// AsAZClientCredentials returns the Azure client-credentials variant, or
// (nil, false) if the union holds a different variant.
func AsAZClientCredentials(sc managementv1.StorageCredential) (*managementv1.StorageCredentialClientCredentials, bool) {
	if sc.StorageCredentialClientCredentials == nil {
		return nil, false
	}
	return sc.StorageCredentialClientCredentials, true
}

// AsAZSharedAccessKey returns the Azure shared-access-key variant, or
// (nil, false) if the union holds a different variant.
func AsAZSharedAccessKey(sc managementv1.StorageCredential) (*managementv1.StorageCredentialSharedAccessKey, bool) {
	if sc.StorageCredentialSharedAccessKey == nil {
		return nil, false
	}
	return sc.StorageCredentialSharedAccessKey, true
}

// AsAZManagedIdentity returns the Azure managed-identity (system-identity)
// variant, or (nil, false) if the union holds a different variant.
func AsAZManagedIdentity(sc managementv1.StorageCredential) (*managementv1.StorageCredentialAzureSystemIdentity, bool) {
	if sc.StorageCredentialAzureSystemIdentity == nil {
		return nil, false
	}
	return sc.StorageCredentialAzureSystemIdentity, true
}

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
