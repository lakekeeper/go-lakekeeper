// Package lakekeeper is an umbrella that re-exports the most-used types and
// constructors from the underlying packages so a normal call site can import
// a single path:
//
//	import "github.com/lakekeeper/go-lakekeeper/pkg/lakekeeper"
//
//	c, err := lakekeeper.New(ctx, baseURL, token)
//	sp := lakekeeper.NewS3Profile("bucket", "us-east-1",
//	    lakekeeper.WithS3Endpoint("http://minio:9000"),
//	)
//	sc := lakekeeper.NewS3AccessKey("ak", "sk")
//	req := lakekeeper.NewCreateWarehouseRequest(sp, "main")
//	req.SetProjectId(projectID)
//	req.SetStorageCredential(sc)
//	wh, err := c.Warehouses.Create(ctx, req)
//
// What it re-exports:
//
//   - Client + constructors from pkg/client
//   - Storage profile/credential builders from pkg/storage/{profile,credential}
//   - Permission builders from pkg/permissions
//   - The most common Management API request/response types
//
// What it does NOT re-export:
//
//   - The full ~189-schema Management API surface. Reach for managementv1
//     when you need a less-common type — type aliases here preserve type
//     identity, so values flow between the two packages without conversion.
//   - Generated enum constants. Use managementv1.<EnumValue> directly; the
//     enum-varnames preprocessor pass produces idiomatic Go names there
//     (e.g. managementv1.WarehouseStatusActive) so an alias would only add
//     a second name for the same value.
package lakekeeper

import (
	"context"

	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

// NewOAuthClientCredentials constructs a Client authenticated via the OAuth
// 2.0 client-credentials flow. Tokens are cached and refreshed before expiry
// by the underlying golang.org/x/oauth2 stack — no manual ticker required.
//
// scopes may be empty; pass each scope as a separate argument
// (`"lakekeeper", "openid"`) to avoid space-splitting confusion.
func NewOAuthClientCredentials(ctx context.Context, baseURL, tokenURL, clientID, clientSecret string, scopes []string, options ...client.Option) (*client.Client, error) {
	cfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       scopes,
	}
	as := &core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(ctx)}
	return client.NewWithAuthSource(ctx, baseURL, as, options...)
}

// NewK8sServiceAccount constructs a Client authenticated via the projected
// Kubernetes service-account token. Pass tokenPath="" to use the standard
// mount at core.DefaultK8sServiceAccountTokenPath.
//
// The token file is re-read on every request so the kubelet's hourly
// rotation is picked up without process restart.
func NewK8sServiceAccount(ctx context.Context, baseURL, tokenPath string, options ...client.Option) (*client.Client, error) {
	as := &core.K8sServiceAccountAuthSource{}
	if tokenPath != "" {
		as.ServiceAccountTokenPath = &tokenPath
	}
	return client.NewWithAuthSource(ctx, baseURL, as, options...)
}

// Client and its constructors.
type (
	Client = client.Client
	Option = client.Option
)

var (
	New                  = client.New
	NewWithAuthSource    = client.NewWithAuthSource
	WithUserAgent        = client.WithUserAgent
	WithoutRetries       = client.WithoutRetries
	WithRetryMax         = client.WithRetryMax
	WithRetryWait        = client.WithRetryWait
	WithCheckRetry       = client.WithCheckRetry
	WithBackoff          = client.WithBackoff
	WithErrorHandler     = client.WithErrorHandler
	WithInitialBootstrap = client.WithInitialBootstrap
)

// Storage profile builders + option types.
type (
	StorageProfile = managementv1.StorageProfile

	S3Option   = profile.S3Option
	GCSOption  = profile.GCSOption
	ADLSOption = profile.ADLSOption
)

var (
	NewS3Profile   = profile.NewS3Profile
	NewGCSProfile  = profile.NewGCSProfile
	NewADLSProfile = profile.NewADLSProfile

	// Read-side oneOf accessors — symmetric with the New*Profile
	// builders, so calling code does not have to navigate the union
	// struct (`if sp.StorageProfileS3 != nil { … }`).
	AsS3Profile   = profile.AsS3
	AsGCSProfile  = profile.AsGCS
	AsADLSProfile = profile.AsADLS

	WithS3Endpoint              = profile.WithS3Endpoint
	WithS3KeyPrefix             = profile.WithS3KeyPrefix
	WithS3Flavor                = profile.WithS3Flavor
	WithS3PathStyleAccess       = profile.WithS3PathStyleAccess
	WithS3AlternativeProtocols  = profile.WithS3AlternativeProtocols
	WithS3AssumeRoleARN         = profile.WithS3AssumeRoleARN
	WithS3AWSKMSKeyARN          = profile.WithS3AWSKMSKeyARN
	WithS3STSEnabled            = profile.WithS3STSEnabled
	WithS3STSRoleARN            = profile.WithS3STSRoleARN
	WithS3STSEndpoint           = profile.WithS3STSEndpoint
	WithS3STSTokenValidity      = profile.WithS3STSTokenValidity
	WithS3RemoteSigningURLStyle = profile.WithS3RemoteSigningURLStyle
	WithS3StorageLayout         = profile.WithS3StorageLayout
	WithS3PushS3DeleteDisabled  = profile.WithS3PushS3DeleteDisabled
	WithS3LegacyMd5Behavior     = profile.WithS3LegacyMd5Behavior
	WithS3RemoteSigningEnabled  = profile.WithS3RemoteSigningEnabled
	WithS3StsSessionTags        = profile.WithS3StsSessionTags

	WithGCSKeyPrefix     = profile.WithGCSKeyPrefix
	WithGCSSTSEnabled    = profile.WithGCSSTSEnabled
	WithGCSStorageLayout = profile.WithGCSStorageLayout

	WithADLSKeyPrefix            = profile.WithADLSKeyPrefix
	WithADLSAuthorityHost        = profile.WithADLSAuthorityHost
	WithADLSHost                 = profile.WithADLSHost
	WithADLSAlternativeProtocols = profile.WithADLSAlternativeProtocols
	WithADLSSASEnabled           = profile.WithADLSSASEnabled
	WithADLSSASTokenValidity     = profile.WithADLSSASTokenValidity
	WithADLSStorageLayout        = profile.WithADLSStorageLayout
)

// Storage credential builders.
type StorageCredential = managementv1.StorageCredential

var (
	NewS3AccessKey                       = credential.NewS3AccessKey
	NewS3AccessKeyWithExternalID         = credential.NewS3AccessKeyWithExternalID
	NewS3AwsSystemIdentity               = credential.NewS3AwsSystemIdentity
	NewS3AwsSystemIdentityWithExternalID = credential.NewS3AwsSystemIdentityWithExternalID
	NewS3CloudflareR2                    = credential.NewS3CloudflareR2

	NewGCSServiceAccountKey = credential.NewGCSServiceAccountKey
	NewGCSSystemIdentity    = credential.NewGCSSystemIdentity

	NewAzClientCredentials = credential.NewAZClientCredentials
	NewAzSharedAccessKey   = credential.NewAZSharedAccessKey
	NewAzManagedIdentity   = credential.NewAZManagedIdentity

	// Read-side oneOf accessors for storage credentials, symmetric with
	// the New* builders.
	AsS3AccessKey          = credential.AsS3AccessKey
	AsS3AwsSystemIdentity  = credential.AsS3AwsSystemIdentity
	AsS3CloudflareR2       = credential.AsS3CloudflareR2
	AsGCSServiceAccountKey = credential.AsGCSServiceAccountKey
	AsGCSSystemIdentity    = credential.AsGCSSystemIdentity
	AsAzClientCredentials  = credential.AsAZClientCredentials
	AsAzSharedAccessKey    = credential.AsAZSharedAccessKey
	AsAzManagedIdentity    = credential.AsAZManagedIdentity
)

// Permission helpers.
type (
	PrincipalKind = permissions.PrincipalKind
	PrincipalSet  = permissions.PrincipalSet
	AssignmentRow = permissions.AssignmentRow
)

const (
	PrincipalUser = permissions.PrincipalUser
	PrincipalRole = permissions.PrincipalRole
)

var (
	BuildAssignment    = permissions.BuildAssignment[managementv1.ServerAssignment] // re-export the generic root; users instantiate with their own T
	BuildAssignmentSet = permissions.BuildAssignmentSet[managementv1.ServerAssignment]
	DescribeAssignment = permissions.DescribeAssignment
)

// Common Management API request and response types.
type (
	// Warehouse types.
	Warehouse              = managementv1.GetWarehouseResponse
	WarehouseStatus        = managementv1.WarehouseStatus
	CreateWarehouseRequest = managementv1.CreateWarehouseRequest
	RenameWarehouseRequest = managementv1.RenameWarehouseRequest
	WarehouseAssignment    = managementv1.WarehouseAssignment

	// Project types.
	Project              = managementv1.GetProjectResponse
	CreateProjectRequest = managementv1.CreateProjectRequest
	RenameProjectRequest = managementv1.RenameProjectRequest
	ProjectAssignment    = managementv1.ProjectAssignment

	// Role types.
	Role              = managementv1.Role
	CreateRoleRequest = managementv1.CreateRoleRequest
	UpdateRoleRequest = managementv1.UpdateRoleRequest
	RoleAssignment    = managementv1.RoleAssignment

	// User types.
	User              = managementv1.User
	UserType          = managementv1.UserType
	CreateUserRequest = managementv1.CreateUserRequest
	UpdateUserRequest = managementv1.UpdateUserRequest

	// Server types.
	ServerInfo       = managementv1.ServerInfo
	BootstrapRequest = managementv1.BootstrapRequest
	ServerAssignment = managementv1.ServerAssignment

	// Misc.
	SetProtectionRequest = managementv1.SetProtectionRequest
	ProtectionResponse   = managementv1.ProtectionResponse
)

// Constructors for the request types listed above.
var (
	NewCreateWarehouseRequest = managementv1.NewCreateWarehouseRequest
	NewRenameWarehouseRequest = managementv1.NewRenameWarehouseRequest

	NewCreateProjectRequest = managementv1.NewCreateProjectRequest
	NewRenameProjectRequest = managementv1.NewRenameProjectRequest

	NewCreateRoleRequest = managementv1.NewCreateRoleRequest
	NewUpdateRoleRequest = managementv1.NewUpdateRoleRequest

	NewCreateUserRequest = managementv1.NewCreateUserRequest
	NewUpdateUserRequest = managementv1.NewUpdateUserRequest

	NewBootstrapRequest = managementv1.NewBootstrapRequest

	NewSetProtectionRequest = managementv1.NewSetProtectionRequest
)
