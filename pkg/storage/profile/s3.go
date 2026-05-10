package profile

import (
	"time"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// S3Option customises an S3 profile during construction.
type S3Option func(*managementv1.StorageProfileS3)

// NewS3Profile constructs a StorageProfileS3 with the spec-required fields
// (bucket, region) populated and STS disabled by default. Apply With* options
// to set anything else.
//
// Returns the StorageProfile union directly so callers can pass it to
// request setters (e.g. CreateWarehouseRequest.StorageProfile) without going
// through the generated *AsStorageProfile wrapper.
func NewS3Profile(bucket, region string, opts ...S3Option) managementv1.StorageProfile {
	p := &managementv1.StorageProfileS3{
		Bucket:     bucket,
		Region:     region,
		StsEnabled: false,
		Type:       typeS3,
	}
	for _, opt := range opts {
		opt(p)
	}
	return managementv1.StorageProfileS3AsStorageProfile(p)
}

func WithS3Endpoint(endpoint string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.Endpoint = &endpoint }
}

func WithS3KeyPrefix(prefix string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.KeyPrefix = &prefix }
}

func WithS3Flavor(flavor S3Flavor) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.Flavor = &flavor }
}

func WithS3PathStyleAccess() S3Option {
	return func(p *managementv1.StorageProfileS3) { p.PathStyleAccess = ptrTo(true) }
}

func WithS3AlternativeProtocols() S3Option {
	return func(p *managementv1.StorageProfileS3) { p.AllowAlternativeProtocols = ptrTo(true) }
}

func WithS3AssumeRoleARN(arn string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.AssumeRoleArn = &arn }
}

func WithS3AWSKMSKeyARN(arn string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.AwsKmsKeyArn = &arn }
}

// WithS3STSEnabled enables STS-vended credentials. STSRoleARN or AssumeRoleARN
// must also be configured for STS to work; the server enforces this.
func WithS3STSEnabled() S3Option {
	return func(p *managementv1.StorageProfileS3) { p.StsEnabled = true }
}

func WithS3STSRoleARN(arn string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.StsRoleArn = &arn }
}

func WithS3STSEndpoint(endpoint string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.StsEndpoint = &endpoint }
}

func WithS3STSTokenValidity(d time.Duration) S3Option {
	return func(p *managementv1.StorageProfileS3) {
		seconds := int64(d.Seconds())
		p.StsTokenValiditySeconds = &seconds
	}
}

func WithS3RemoteSigningURLStyle(style S3UrlStyleDetectionMode) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.RemoteSigningUrlStyle = &style }
}

func WithS3StorageLayout(layout StorageLayout) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.StorageLayout = &layout }
}

// WithS3PushS3DeleteDisabled controls whether the `s3.delete-enabled=false`
// flag is pushed to clients (Spark `DROP TABLE PURGE` safety). Spec default
// is server-side; pass false explicitly to opt out.
func WithS3PushS3DeleteDisabled(disabled bool) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.PushS3DeleteDisabled = &disabled }
}

func WithS3LegacyMd5Behavior(enabled bool) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.LegacyMd5Behavior = &enabled }
}

// WithS3RemoteSigningEnabled toggles remote signing for S3 requests. Spec
// default is true; pass false to disable.
func WithS3RemoteSigningEnabled(enabled bool) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.RemoteSigningEnabled = &enabled }
}

func WithS3StsSessionTags(tags map[string]string) S3Option {
	return func(p *managementv1.StorageProfileS3) { p.StsSessionTags = tags }
}
