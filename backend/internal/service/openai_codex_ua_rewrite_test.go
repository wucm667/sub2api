//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteCodexCLIVersion(t *testing.T) {
	tests := []struct {
		name          string
		downstreamUA  string
		targetVersion string
		want          string
	}{
		{
			name:          "rewrite full codex cli ua keeps platform details",
			downstreamUA:  "codex_cli_rs/0.98.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "0.131.0",
			want:          "codex_cli_rs/0.131.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "rewrite bare codex cli ua",
			downstreamUA:  "codex_cli_rs/0.125.0",
			targetVersion: "0.131.0",
			want:          "codex_cli_rs/0.131.0",
		},
		{
			name:          "rewrite prerelease downstream version",
			downstreamUA:  "codex_cli_rs/0.125.0-beta.1 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "0.131.0",
			want:          "codex_cli_rs/0.131.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "empty target leaves unchanged",
			downstreamUA:  "codex_cli_rs/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "",
			want:          "codex_cli_rs/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "codex vscode leaves unchanged",
			downstreamUA:  "codex_vscode/0.45.0 (Darwin 14.5.0; arm64) VSCode/1.101.0",
			targetVersion: "0.131.0",
			want:          "codex_vscode/0.45.0 (Darwin 14.5.0; arm64) VSCode/1.101.0",
		},
		{
			name:          "codex tui leaves unchanged",
			downstreamUA:  "codex-tui/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "0.131.0",
			want:          "codex-tui/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "codex atlas leaves unchanged",
			downstreamUA:  "codex_atlas/1.0.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "0.131.0",
			want:          "codex_atlas/1.0.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "embedded historical hash ua leaves unchanged",
			downstreamUA:  "Mozilla/5.0 codex_cli_rs/0.1.0",
			targetVersion: "0.131.0",
			want:          "Mozilla/5.0 codex_cli_rs/0.1.0",
		},
		{
			name:          "curl leaves unchanged",
			downstreamUA:  "curl/8.0",
			targetVersion: "0.131.0",
			want:          "curl/8.0",
		},
		{
			name:          "empty downstream leaves unchanged",
			downstreamUA:  "",
			targetVersion: "0.131.0",
			want:          "",
		},
		{
			name:          "short semver downstream leaves unchanged",
			downstreamUA:  "codex_cli_rs/0.125",
			targetVersion: "0.131.0",
			want:          "codex_cli_rs/0.125",
		},
		{
			name:          "four segment downstream leaves unchanged",
			downstreamUA:  "codex_cli_rs/0.125.0.0",
			targetVersion: "0.131.0",
			want:          "codex_cli_rs/0.125.0.0",
		},
		{
			name:          "invalid target leaves unchanged",
			downstreamUA:  "codex_cli_rs/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "abc",
			want:          "codex_cli_rs/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
		{
			name:          "target prerelease is allowed",
			downstreamUA:  "codex_cli_rs/0.125.0 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
			targetVersion: "0.131.0-beta.1",
			want:          "codex_cli_rs/0.131.0-beta.1 (Darwin 14.5.0; arm64) iTerm.app/3.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, rewriteCodexCLIVersion(tt.downstreamUA, tt.targetVersion))
		})
	}
}
