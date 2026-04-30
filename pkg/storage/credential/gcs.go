package credential

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// NewGCSServiceAccountKey constructs a GCS credential that uses an explicit
// service account key. The key argument matches the JSON shape Google emits
// when downloading a service account key file.
func NewGCSServiceAccountKey(key managementv1.GcsServiceKey) *managementv1.StorageCredentialGcs {
	return &managementv1.StorageCredentialGcs{
		Key:            key,
		CredentialType: credGCSServiceAccount,
		Type:           typeGCS,
	}
}

// NewGCSSystemIdentity constructs a credential that delegates to the GCP
// system identity (workload identity / service account) attached to
// Lakekeeper's runtime environment.
func NewGCSSystemIdentity() *managementv1.StorageCredentialGcs {
	return &managementv1.StorageCredentialGcs{
		CredentialType: credGCSSystemIdentity,
		Type:           typeGCS,
	}
}
