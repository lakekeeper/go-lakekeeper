package credential

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// NewGCSServiceAccountKey constructs a GCS credential that uses an explicit
// service account key. The key argument matches the JSON shape Google emits
// when downloading a service account key file.
func NewGCSServiceAccountKey(key managementv1.GcsServiceKey) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialServiceAccountKey{
		Key:            key,
		CredentialType: credGCSServiceAccount,
		Type:           typeGCS,
	}
	return managementv1.StorageCredentialServiceAccountKeyAsStorageCredential(leaf)
}

// NewGCSSystemIdentity constructs a credential that delegates to the GCP
// system identity (workload identity / service account) attached to
// Lakekeeper's runtime environment.
func NewGCSSystemIdentity() managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialGcpSystemIdentity{
		CredentialType: credGCSSystemIdentity,
		Type:           typeGCS,
	}
	return managementv1.StorageCredentialGcpSystemIdentityAsStorageCredential(leaf)
}
