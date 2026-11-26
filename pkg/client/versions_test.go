package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func TestSupportedVersions_SpecMinorAndThePreviousOne(t *testing.T) {
	t.Parallel()

	r := client.SupportedVersionsFrom(client.EditionLakekeeper, "0.15.0")

	assert.Equal(t, client.VersionRange{Edition: client.EditionLakekeeper, Oldest: "0.14", Newest: "0.15"}, r)
	assert.Equal(t, "Lakekeeper 0.14.x and 0.15.x", r.String())
}

func TestSupportedVersions_Contains(t *testing.T) {
	t.Parallel()
	r := client.SupportedVersionsFrom(client.EditionLakekeeperPlus, "0.15.1")

	for version, want := range map[string]bool{
		"0.15.0":       true,  // same minor, older patch than the spec
		"0.15.7":       true,  // same minor, newer patch
		"0.14.2":       true,  // previous minor
		"v0.14.0":      true,  // tag form
		"0.15.1-rc.1":  true,  // pre-release of the same minor
		"0.13.9":       false, // older minor
		"0.16.0":       false, // newer minor
		"1.15.0":       false, // other major
		"":             false, // unknown
		"not-a-semver": false,
	} {
		assert.Equal(t, want, r.Contains(version), version)
	}
	assert.Equal(t, "Lakekeeper+ 0.14.x and 0.15.x", r.String())
}

func TestSupportedVersions_FirstMinorOfAMajorHasNoPreviousMinor(t *testing.T) {
	t.Parallel()

	r := client.SupportedVersionsFrom(client.EditionLakekeeper, "1.0.2")

	assert.Equal(t, "Lakekeeper 1.0.x", r.String())
	assert.True(t, r.Contains("1.0.0"))
	assert.False(t, r.Contains("0.15.0"))
}

func TestSupportedVersions_PerEditionFromItsSpecRelease(t *testing.T) {
	t.Parallel()

	assert.Equal(t, client.SupportedVersionsFrom(client.EditionLakekeeper, managementv1.LakekeeperSpecRelease),
		client.SupportedVersions(client.EditionLakekeeper))
	assert.Equal(t, client.SupportedVersionsFrom(client.EditionLakekeeperPlus, managementv1.LakekeeperPlusSpecRelease),
		client.SupportedVersions(client.EditionLakekeeperPlus))
}

func TestCapabilities_SupportedVersionsCheckTheServersEdition(t *testing.T) {
	t.Parallel()
	lk := client.SupportedVersions(client.EditionLakekeeper)
	plus := client.SupportedVersions(client.EditionLakekeeperPlus)

	for name, tc := range map[string]struct {
		caps      client.Capabilities
		wantRange client.VersionRange
		supported bool
	}{
		"Lakekeeper on the spec minor": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeper, LakekeeperVersion: lk.Newest + ".0"},
			wantRange: lk, supported: true,
		},
		"Lakekeeper on the previous minor": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeper, LakekeeperVersion: lk.Oldest + ".9"},
			wantRange: lk, supported: true,
		},
		"Lakekeeper too old": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeper, LakekeeperVersion: "0.1.0"},
			wantRange: lk, supported: false,
		},
		"Lakekeeper too new": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeper, LakekeeperVersion: "99.0.0"},
			wantRange: lk, supported: false,
		},
		// The Lakekeeper+ version decides for Lakekeeper+, whatever
		// Lakekeeper version it is built on.
		"Lakekeeper+ on the spec minor": {
			caps: client.Capabilities{
				Edition: client.EditionLakekeeperPlus, LakekeeperVersion: "0.1.0", LakekeeperPlusVersion: plus.Newest + ".3",
			},
			wantRange: plus, supported: true,
		},
		"Lakekeeper+ too old": {
			caps: client.Capabilities{
				Edition: client.EditionLakekeeperPlus, LakekeeperVersion: lk.Newest + ".0", LakekeeperPlusVersion: "0.1.0",
			},
			wantRange: plus, supported: false,
		},
		// No version to judge by: not reported as outside the range.
		"Lakekeeper+ without its version": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeperPlus, LakekeeperVersion: lk.Newest + ".0"},
			wantRange: plus, supported: true,
		},
		"Lakekeeper without a version": {
			caps:      client.Capabilities{Edition: client.EditionLakekeeper},
			wantRange: lk, supported: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.wantRange, tc.caps.SupportedVersions())
			assert.Equal(t, tc.supported, tc.caps.InSupportedVersions())
		})
	}
}
