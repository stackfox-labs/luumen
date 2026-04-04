package resolver

import (
	"errors"
	"fmt"
	"strings"
)

type DependencyKind string

const (
	DependencyKindTool    DependencyKind = "tool"
	DependencyKindPackage DependencyKind = "package"
)

var (
	ErrInvalidDependencyInput = errors.New("invalid dependency input")
	ErrUnknownDependencyKind  = errors.New("unknown dependency kind")
	ErrAmbiguousDependency    = errors.New("ambiguous dependency")
)

var toolAliases = map[string]string{
	"rojo":   "rojo-rbx/rojo@7.6.1",
	"wally":  "UpliftGames/wally@0.3.2",
	"stylua": "JohnnyMorganz/StyLua@2.4.0",
	"selene": "Kampfkarren/selene@0.30.1",
	"lune":   "lune-org/lune@0.10.4",
	"lute":   "luau-lang/lute@0.1.0-nightly.20260327",
	"luau":   "luau-lang/luau@0.680.0",
}

type Resolution struct {
	Kind     DependencyKind
	Value    string
	Alias    string
	Source   string
	Original string
}

func Resolve(input string) (Resolution, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Resolution{}, fmt.Errorf("%w: dependency value cannot be empty", ErrInvalidDependencyInput)
	}

	if explicit, ok, err := resolveExplicitPrefix(raw); ok || err != nil {
		return explicit, err
	}

	aliasKey := strings.ToLower(raw)
	if canonical, ok := toolAliases[aliasKey]; ok {
		return Resolution{
			Kind:     DependencyKindTool,
			Value:    canonical,
			Alias:    aliasKey,
			Source:   "alias",
			Original: raw,
		}, nil
	}

	if looksLikePackageReference(raw) {
		return Resolution{
			Kind:     DependencyKindPackage,
			Value:    raw,
			Source:   "package-heuristic",
			Original: raw,
		}, nil
	}

	return Resolution{}, fmt.Errorf("%w: %q is not a known tool alias and not a confident package reference; use tool:<owner/repo> or pkg:<scope/name>", ErrAmbiguousDependency, raw)
}

func ToolAliases() map[string]string {
	copied := make(map[string]string, len(toolAliases))
	for key, value := range toolAliases {
		copied[key] = value
	}
	return copied
}

func resolveExplicitPrefix(raw string) (Resolution, bool, error) {
	index := strings.Index(raw, ":")
	if index <= 0 {
		return Resolution{}, false, nil
	}

	prefix := strings.ToLower(strings.TrimSpace(raw[:index]))
	value := strings.TrimSpace(raw[index+1:])

	switch prefix {
	case "tool":
		if value == "" {
			return Resolution{}, true, fmt.Errorf("%w: tool prefix requires a value", ErrInvalidDependencyInput)
		}
		normalized, normalizeErr := normalizeToolRef(value)
		if normalizeErr != nil {
			return Resolution{}, true, normalizeErr
		}
		return Resolution{Kind: DependencyKindTool, Value: normalized, Source: "prefix", Original: raw}, true, nil
	case "pkg":
		if value == "" {
			return Resolution{}, true, fmt.Errorf("%w: pkg prefix requires a value", ErrInvalidDependencyInput)
		}
		return Resolution{Kind: DependencyKindPackage, Value: value, Source: "prefix", Original: raw}, true, nil
	default:
		if strings.Contains(prefix, "/") {
			return Resolution{}, false, nil
		}
		return Resolution{}, true, fmt.Errorf("%w: %q (supported: tool:, pkg:)", ErrUnknownDependencyKind, prefix)
	}
}

func normalizeToolRef(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(trimmed, "@") {
		parts := strings.SplitN(trimmed, "@", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("%w: tool references must use tool:<owner/repo>@x.y.z", ErrInvalidDependencyInput)
		}
		return trimmed, nil
	}

	if looksLikeToolReference(trimmed) {
		return trimmed, nil
	}

	if known, ok := knownToolWithDefaultVersion(trimmed); ok {
		return known, nil
	}

	return "", fmt.Errorf("%w: tool references must use tool:<owner/repo> or tool:<owner/repo>@x.y.z, or use a known alias like rojo", ErrInvalidDependencyInput)
}

func looksLikeToolReference(value string) bool {
	return looksLikePackageReference(value)
}

func knownToolWithDefaultVersion(value string) (string, bool) {
	if canonical, ok := toolAliases[strings.ToLower(strings.TrimSpace(value))]; ok {
		return canonical, true
	}

	needle := strings.ToLower(strings.TrimSpace(value))
	for _, canonical := range toolAliases {
		base := canonical
		if index := strings.Index(canonical, "@"); index >= 0 {
			base = canonical[:index]
		}
		if strings.EqualFold(base, needle) {
			return canonical, true
		}
	}

	return "", false
}

func looksLikePackageReference(value string) bool {
	if strings.Contains(value, " ") {
		return false
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return false
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return false
	}
	return true
}
