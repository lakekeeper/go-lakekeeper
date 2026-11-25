package client

import (
	"regexp"
	"strconv"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// VersionRange is the server versions of one edition the SDK supports: the
// minor of the spec it was generated from and the minor before it, for
// example Lakekeeper 0.14.x and 0.15.x. Lakekeeper minors can make breaking
// changes to the Management API; patch releases have not, so the range is by
// minor. The SDK never refuses a call because of it; it is for tooling that
// wants to warn, as lkctl does.
type VersionRange struct {
	Edition Edition
	// Oldest and Newest are the oldest and newest supported minor, as
	// "major.minor" (for example "0.14" and "0.15"). They are equal when the
	// spec minor is the first of its major, whose previous minor is unknown.
	Oldest string
	Newest string
}

// String names the range, for example "Lakekeeper 0.14.x and 0.15.x".
func (r VersionRange) String() string {
	if r.Oldest == r.Newest {
		return string(r.Edition) + " " + r.Newest + ".x"
	}
	return string(r.Edition) + " " + r.Oldest + ".x and " + r.Newest + ".x"
}

// Contains reports whether version, as a server reports it ("0.15.1",
// optionally with a leading v or a pre-release suffix), is in the range. A
// version that does not parse is not.
func (r VersionRange) Contains(version string) bool {
	v, ok := parseVersion(version)
	if !ok {
		return false
	}
	minor := v.minorString()
	return minor == r.Oldest || minor == r.Newest
}

// SupportedVersions returns the server versions of edition the SDK supports,
// derived from the release of that edition's spec
// (managementv1.LakekeeperSpecRelease, managementv1.LakekeeperPlusSpecRelease).
func SupportedVersions(edition Edition) VersionRange {
	return supportedVersionsFrom(edition, specRelease(edition))
}

func supportedVersionsFrom(edition Edition, release string) VersionRange {
	v, _ := parseVersion(release)
	r := VersionRange{Edition: edition, Oldest: v.minorString(), Newest: v.minorString()}
	if v.minor > 0 {
		r.Oldest = serverVersion{major: v.major, minor: v.minor - 1}.minorString()
	}
	return r
}

// specRelease is the release of the spec edition's operations came from.
func specRelease(edition Edition) string {
	if edition == EditionLakekeeperPlus {
		return managementv1.LakekeeperPlusSpecRelease
	}
	return managementv1.LakekeeperSpecRelease
}

// SupportedVersions returns the supported range for the server's edition.
func (c *Capabilities) SupportedVersions() VersionRange {
	return SupportedVersions(c.Edition)
}

// InSupportedVersions reports whether the server's version, for its edition
// (the Lakekeeper+ version on Lakekeeper+), is in SupportedVersions. A
// server whose version is missing or does not parse counts as supported:
// there is nothing to warn about.
func (c *Capabilities) InSupportedVersions() bool {
	v := c.version(c.Edition)
	if _, ok := parseVersion(v); !ok {
		return true
	}
	return c.SupportedVersions().Contains(v)
}

type serverVersion struct{ major, minor, patch int }

func (v serverVersion) minorString() string {
	return strconv.Itoa(v.major) + "." + strconv.Itoa(v.minor)
}

func (v serverVersion) less(o serverVersion) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	return v.patch < o.patch
}

// versionPattern matches major.minor.patch, with an optional leading v and
// any pre-release or build suffix, which is ignored.
var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$`)

func parseVersion(s string) (serverVersion, bool) {
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return serverVersion{}, false
	}
	var v serverVersion
	var err error
	for i, p := range []*int{&v.major, &v.minor, &v.patch} {
		if *p, err = strconv.Atoi(m[i+1]); err != nil {
			return serverVersion{}, false
		}
	}
	return v, true
}
