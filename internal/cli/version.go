package cli

import "strings"

var version = "dev"

func currentVersion() string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "dev"
	}
	return trimmed
}
