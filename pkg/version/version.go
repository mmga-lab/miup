package version

import (
	"fmt"
	"runtime"
)

var (
	// MiUpVersion is the version of MiUp (set at build time)
	MiUpVersion = "v0.1.0-dev"
	// GitHash is the git commit hash (set at build time)
	GitHash = "unknown"
	// GitBranch is the git branch (set at build time)
	GitBranch = "unknown"
	// BuildTime is the build time (set at build time)
	BuildTime = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitHash   string `json:"gitHash"`
	GitBranch string `json:"gitBranch"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetVersionInfo returns the version information
func GetVersionInfo() Info {
	return Info{
		Version:   MiUpVersion,
		GitHash:   GitHash,
		GitBranch: GitBranch,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func (v Info) String() string {
	return fmt.Sprintf(`MiUp %s
Git Commit: %s
Git Branch: %s
Build Time: %s
Go Version: %s
OS/Arch:    %s/%s`,
		v.Version,
		v.GitHash,
		v.GitBranch,
		v.BuildTime,
		v.GoVersion,
		v.OS,
		v.Arch,
	)
}

// ShortString returns a short version string
func (v Info) ShortString() string {
	return fmt.Sprintf("miup version %s", v.Version)
}
