package credential

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// NewS3AccessKey constructs an S3 access-key/secret-access-key credential.
func NewS3AccessKey(accessKeyID, secretAccessKey string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialAccessKey{
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		CredentialType:  credS3AccessKey,
		Type:            typeS3,
	}
	return managementv1.StorageCredentialAccessKeyAsStorageCredential(leaf)
}

// NewS3AccessKeyWithExternalID is like NewS3AccessKey but also sets the
// optional external-id used by some assume-role flows.
func NewS3AccessKeyWithExternalID(accessKeyID, secretAccessKey, externalID string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialAccessKey{
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		ExternalId:      &externalID,
		CredentialType:  credS3AccessKey,
		Type:            typeS3,
	}
	return managementv1.StorageCredentialAccessKeyAsStorageCredential(leaf)
}

// NewS3AwsSystemIdentity constructs an S3 credential that delegates to the
// AWS system identity (IAM role / instance profile) under which Lakekeeper
// itself runs.
func NewS3AwsSystemIdentity() managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialAwsSystemIdentity{
		CredentialType: credS3AwsSystemIdentity,
		Type:           typeS3,
	}
	return managementv1.StorageCredentialAwsSystemIdentityAsStorageCredential(leaf)
}

// NewS3AwsSystemIdentityWithExternalID is like NewS3AwsSystemIdentity but
// sets the optional external-id used by some assume-role flows.
func NewS3AwsSystemIdentityWithExternalID(externalID string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialAwsSystemIdentity{
		ExternalId:     &externalID,
		CredentialType: credS3AwsSystemIdentity,
		Type:           typeS3,
	}
	return managementv1.StorageCredentialAwsSystemIdentityAsStorageCredential(leaf)
}

// NewS3CloudflareR2 constructs a Cloudflare R2 credential. All four fields
// are spec-required: the long-lived access key + secret are used by
// Lakekeeper for IO operations, and the account ID + API token drive the
// R2 vended-credential endpoint.
func NewS3CloudflareR2(accessKeyID, secretAccessKey, accountID, token string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialCloudflareR2{
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		AccountId:       accountID,
		Token:           token,
		CredentialType:  credS3CloudflareR2,
		Type:            typeS3,
	}
	return managementv1.StorageCredentialCloudflareR2AsStorageCredential(leaf)
}
