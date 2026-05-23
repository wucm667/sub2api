package service

import (
	"log/slog"
	"regexp"
	"strings"
)

var (
	codexCLIVersionRe       = regexp.MustCompile(`^codex_cli_rs/\d+\.\d+\.\d+(?:-[\w.]+)?`)
	codexCLITargetVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[\w.]+)?$`)
)

func rewriteCodexCLIVersion(downstreamUA, targetVersion string) string {
	targetVersion = strings.TrimSpace(targetVersion)
	if targetVersion == "" {
		return downstreamUA
	}
	if !codexCLITargetVersionRe.MatchString(targetVersion) {
		slog.Warn("invalid codex cli version setting", "version", targetVersion)
		return downstreamUA
	}
	matched := codexCLIVersionRe.FindStringIndex(downstreamUA)
	if matched == nil {
		return downstreamUA
	}
	if matched[1] < len(downstreamUA) && downstreamUA[matched[1]] != ' ' {
		return downstreamUA
	}
	return "codex_cli_rs/" + targetVersion + downstreamUA[matched[1]:]
}
