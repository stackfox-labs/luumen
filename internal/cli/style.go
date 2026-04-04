package cli

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const (
	ansiReset      = "\x1b[0m"
	ansiBold       = "\x1b[1m"
	ansiDim        = "\x1b[2m"
	ansiLuumenPink = "\x1b[38;2;255;32;86m"
	ansiSuccess    = "\x1b[38;2;68;199;103m"
	ansiWarning    = "\x1b[38;2;245;158;11m"
	ansiError      = "\x1b[38;2;239;68;68m"
)

func styleAccent(writer io.Writer, value string) string {
	return applyStyle(writer, value, ansiBold+ansiLuumenPink)
}

func styleSuccess(writer io.Writer, value string) string {
	return applyStyle(writer, value, ansiBold+ansiSuccess)
}

func styleWarning(writer io.Writer, value string) string {
	return applyStyle(writer, value, ansiBold+ansiWarning)
}

func styleError(writer io.Writer, value string) string {
	return applyStyle(writer, value, ansiBold+ansiError)
}

func styleMuted(writer io.Writer, value string) string {
	return applyStyle(writer, value, ansiDim)
}

func styleCommand(writer io.Writer, value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return value
	}

	parts := make([]string, 0, len(fields))
	parts = append(parts, styleAccent(writer, fields[0]))
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "-") {
			parts = append(parts, styleMuted(writer, field))
			continue
		}
		parts = append(parts, field)
	}

	return strings.Join(parts, " ")
}

func RenderCLIError(writer io.Writer, err error) string {
	if err == nil {
		return ""
	}

	problem, next := splitNextHint(err.Error())
	lines := []string{
		"",
		fmt.Sprintf("%s %s", styleError(writer, "error:"), strings.TrimSpace(problem)),
	}

	if strings.TrimSpace(next) != "" {
		lines = append(lines, fmt.Sprintf("%s %s", styleAccent(writer, "→"), strings.TrimSpace(next)))
	}

	return strings.Join(lines, "\n")
}

func splitNextHint(message string) (string, string) {
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)
	index := strings.Index(lower, " next:")
	if index < 0 {
		index = strings.Index(lower, "next:")
	}
	if index < 0 {
		return trimmed, ""
	}

	problem := strings.TrimSpace(trimmed[:index])
	next := strings.TrimSpace(trimmed[index:])
	next = strings.TrimPrefix(next, "Next:")
	next = strings.TrimPrefix(next, "next:")
	return problem, strings.TrimSpace(next)
}

func applyStyle(writer io.Writer, value string, style string) string {
	if !colorEnabled(writer) || style == "" {
		return value
	}
	return style + value + ansiReset
}

func colorEnabled(writer io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") == "1" || os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	file, ok := writer.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	return term != "" && term != "dumb"
}
