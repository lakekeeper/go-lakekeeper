package client

// SupportedVersionsFrom exposes the range derivation for a spec release other
// than the one the SDK is built from.
func SupportedVersionsFrom(edition Edition, specRelease string) VersionRange {
	return supportedVersionsFrom(edition, specRelease)
}
