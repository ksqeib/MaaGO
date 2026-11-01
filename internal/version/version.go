// Package version exposes build-time application metadata.
package version

// These values are overridable at build time with -ldflags "-X".
var (
	// Version is the user-facing application version.
	Version = "0.1.0"
	// Commit is the short git revision, when available.
	Commit = "dev"
	// BuildDate is the UTC build timestamp.
	BuildDate = "unknown"
)
