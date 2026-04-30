package credential

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// NewS3AccessKey constructs an access-key/secret-access-key S3 credential.
// To set the optional ExternalId, assign it on the returned struct.
func NewS3AccessKey(accessKeyID, secretAccessKey string) *managementv1.StorageCredentialS3 {
	return &managementv1.StorageCredentialS3{
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		CredentialType:  credS3AccessKey,
		Type:            typeS3,
	}
}

// NewS3AwsSystemIdentity constructs a credential that delegates to the AWS
// system identity (IAM role / instance profile) under which Lakekeeper itself
// runs. The optional ExternalId can be assigned on the returned struct.
func NewS3AwsSystemIdentity() *managementv1.StorageCredentialS3 {
	return &managementv1.StorageCredentialS3{
		CredentialType: credS3AwsSystemIdentity,
		Type:           typeS3,
	}
}

// NewS3CloudflareR2 constructs a Cloudflare R2 credential. All four fields are
// spec-required: the long-lived access key + secret are used by Lakekeeper for
// IO operations, and the account ID + API token drive the R2 vended-credential
// endpoint.
func NewS3CloudflareR2(accessKeyID, secretAccessKey, accountID, token string) *managementv1.StorageCredentialS3 {
	return &managementv1.StorageCredentialS3{
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		AccountId:       accountID,
		Token:           token,
		CredentialType:  credS3CloudflareR2,
		Type:            typeS3,
	}
}
