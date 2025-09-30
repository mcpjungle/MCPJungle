package version

import (
	"runtime/debug"
)

const defaultVersion = "dev"

// Version can be overridden at build time using:
// go build -ldflags="-X 'github.com/mcpjungle/mcpjungle/pkg/version.Version=v1.2.3'"
var Version = defaultVersion

// GetVersion returns the version string using build info or fallback to default.
func GetVersion() string {
	if Version != "" && Version != defaultVersion {
		return NormalizeVersion(Version)
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return NormalizeVersion(info.Main.Version)
	}

	return defaultVersion
}

// NormalizeVersion ensures a consistent version format:
// - If version starts with a digit (e.g., "1.2.3"), prefix with 'v' → "v1.2.3"
// - Leave values starting with 'v' or non-semver strings untouched
func NormalizeVersion(v string) string {
	if v == "" {
		return v
	}
	if v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}
