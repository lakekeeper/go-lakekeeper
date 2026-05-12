package profile

import (
	"time"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// ADLSOption customises an ADLS profile during construction.
type ADLSOption func(*managementv1.StorageProfileAdls)

// NewADLSProfile constructs a StorageProfileAdls with the spec-required fields
// (account name, filesystem) populated. Apply With* options for anything else.
//
// Returns the StorageProfile union directly so callers can pass it to
// request setters (e.g. CreateWarehouseRequest.StorageProfile) without going
// through the generated *AsStorageProfile wrapper.
func NewADLSProfile(accountName, filesystem string, opts ...ADLSOption) managementv1.StorageProfile {
	p := &managementv1.StorageProfileAdls{
		AccountName: accountName,
		Filesystem:  filesystem,
		Type:        typeADLS,
	}
	for _, opt := range opts {
		opt(p)
	}
	return managementv1.StorageProfileAdlsAsStorageProfile(p)
}

func WithADLSKeyPrefix(prefix string) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.KeyPrefix = &prefix }
}

func WithADLSAuthorityHost(host string) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.AuthorityHost = &host }
}

func WithADLSHost(host string) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.Host = &host }
}

func WithADLSAlternativeProtocols() ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.AllowAlternativeProtocols = ptrTo(true) }
}

// WithADLSSASEnabled lets the caller explicitly set SAS-vended-credentials on
// or off. The spec default is true, so passing false is the typical use here.
func WithADLSSASEnabled(enabled bool) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.SasEnabled = &enabled }
}

func WithADLSSASTokenValidity(d time.Duration) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) {
		seconds := int64(d.Seconds())
		p.SasTokenValiditySeconds = &seconds
	}
}

func WithADLSStorageLayout(layout StorageLayout) ADLSOption {
	return func(p *managementv1.StorageProfileAdls) { p.StorageLayout = &layout }
}
