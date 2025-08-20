// Package version provides build and version information for Shylock.
package version

import (
	"fmt"
	"runtime"
)

// Build information - these will be set by GoReleaser at build time
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// Info represents version and build information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// GetInfo returns the current version and build information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	if Version == "dev" {
		return fmt.Sprintf("Shylock %s (commit: %s, built: %s)", Version, Commit, Date)
	}
	return fmt.Sprintf("Shylock v%s", Version)
}

// GetFullVersionString returns a detailed version string with all build info
func GetFullVersionString() string {
	info := GetInfo()
	return fmt.Sprintf(`Shylock AWS Cost Estimation Tool
Version:    %s
Commit:     %s
Built:      %s
Built by:   %s
Go version: %s
Platform:   %s`, info.Version, info.Commit, info.Date, info.BuiltBy, info.GoVersion, info.Platform)
}
