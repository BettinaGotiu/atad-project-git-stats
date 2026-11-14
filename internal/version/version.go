package version

import (
	"encoding/json"
	"io"
	"runtime/debug"
)

var (
	App     = "Git Repository Statistics"
	Version = "0.0.0-dev"
	Commit  = ""
	BuiltAt = ""
)

type Info struct {
	App     string `json:"app"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
}

// Current prefers linker values and falls back to Go's local build metadata.
func Current() Info {
	commit, builtAt := Commit, BuiltAt
	if commit == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				if s.Key == "vcs.revision" {
					commit = s.Value
					break
				}
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	if builtAt == "" {
		builtAt = "unknown"
	}
	return Info{App: App, Version: Version, Commit: commit, BuiltAt: builtAt}
}

// WriteJSON writes the build information as one line of JSON.
func WriteJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(Current())
}
