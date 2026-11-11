package profile

import (
	"fmt"
	"time"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Discriminator values of the TabularDeleteProfile oneOf (the "type" field),
// taken from the spec.
const (
	DeleteProfileTypeHard = "hard"
	DeleteProfileTypeSoft = "soft"
)

// NewHardDeleteProfile returns a delete profile that removes dropped tables
// and views immediately.
func NewHardDeleteProfile() managementv1.TabularDeleteProfile {
	return managementv1.TabularDeleteProfileHardAsTabularDeleteProfile(
		managementv1.NewTabularDeleteProfileHard(DeleteProfileTypeHard))
}

// NewSoftDeleteProfile returns a delete profile that keeps dropped tables and
// views for expiration before removing them; until then they can be undropped.
//
// The API counts whole seconds, so expiration is truncated to the second. An
// expiration under one second is rejected: the server does not check the
// value, and would store a zero or negative retention as given.
func NewSoftDeleteProfile(expiration time.Duration) (managementv1.TabularDeleteProfile, error) {
	if expiration < time.Second {
		return managementv1.TabularDeleteProfile{}, fmt.Errorf("soft delete profile: expiration must be at least 1s, got %v", expiration)
	}
	return managementv1.TabularDeleteProfileSoftAsTabularDeleteProfile(
		managementv1.NewTabularDeleteProfileSoft(int64(expiration/time.Second), DeleteProfileTypeSoft)), nil
}

// AsHardDeleteProfile returns the hard variant of a TabularDeleteProfile
// union, or (nil, false) if the union holds a different variant.
func AsHardDeleteProfile(dp managementv1.TabularDeleteProfile) (*managementv1.TabularDeleteProfileHard, bool) {
	if dp.TabularDeleteProfileHard == nil {
		return nil, false
	}
	return dp.TabularDeleteProfileHard, true
}

// AsSoftDeleteProfile returns the soft variant of a TabularDeleteProfile
// union, or (nil, false) if the union holds a different variant. Read the
// retention with GetExpirationSeconds.
func AsSoftDeleteProfile(dp managementv1.TabularDeleteProfile) (*managementv1.TabularDeleteProfileSoft, bool) {
	if dp.TabularDeleteProfileSoft == nil {
		return nil, false
	}
	return dp.TabularDeleteProfileSoft, true
}
