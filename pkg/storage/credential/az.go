package credential

import managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"

// NewAZClientCredentials constructs an Azure credential authenticated via
// AAD client credentials (tenant + client ID + client secret).
func NewAZClientCredentials(tenantID, clientID, clientSecret string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialClientCredentials{
		TenantId:       tenantID,
		ClientId:       clientID,
		ClientSecret:   clientSecret,
		CredentialType: credAZClient,
		Type:           typeAZ,
	}
	return managementv1.StorageCredentialClientCredentialsAsStorageCredential(leaf)
}

// NewAZSharedAccessKey constructs an Azure credential authenticated via a
// storage account shared access key.
func NewAZSharedAccessKey(key string) managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialSharedAccessKey{
		Key:            key,
		CredentialType: credAZSharedAccessKey,
		Type:           typeAZ,
	}
	return managementv1.StorageCredentialSharedAccessKeyAsStorageCredential(leaf)
}

// NewAZManagedIdentity constructs a credential that delegates to the Azure
// system identity (managed identity) attached to Lakekeeper's runtime
// environment.
func NewAZManagedIdentity() managementv1.StorageCredential {
	leaf := &managementv1.StorageCredentialAzureSystemIdentity{
		CredentialType: credAZManagedIdentity,
		Type:           typeAZ,
	}
	return managementv1.StorageCredentialAzureSystemIdentityAsStorageCredential(leaf)
}
