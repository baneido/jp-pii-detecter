package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionFrom(t *testing.T) {
	tests := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{
			name:    "ldflags override wins",
			ldflags: "v9.9.9",
			info:    &debug.BuildInfo{Main: debug.Module{Version: "v0.1.3"}},
			ok:      true,
			want:    "v9.9.9",
		},
		{
			name:    "go install module version",
			ldflags: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Version: "v0.1.3"}},
			ok:      true,
			want:    "v0.1.3",
		},
		{
			name:    "local build falls back to vcs revision",
			ldflags: "dev",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "0123456789abcdef0123"},
				},
			},
			ok:   true,
			want: "dev-0123456789ab",
		},
		{
			name:    "local build dirty tree",
			ldflags: "dev",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abcdef123456"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			ok:   true,
			want: "dev-abcdef123456-dirty",
		},
		{
			name:    "no build info",
			ldflags: "dev",
			info:    nil,
			ok:      false,
			want:    "dev",
		},
		{
			name:    "devel without vcs info",
			ldflags: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			ok:      true,
			want:    "dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFrom(tt.ldflags, tt.info, tt.ok); got != tt.want {
				t.Errorf("versionFrom(%q, ...) = %q, want %q", tt.ldflags, got, tt.want)
			}
		})
	}
}
