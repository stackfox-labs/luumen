package resolver

import (
	"errors"
	"testing"
)

func TestResolveToolPrefix(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve("tool:rojo-rbx/rojo")
	if err != nil {
		t.Fatalf("expected tool prefix resolution success, got: %v", err)
	}
	if resolved.Kind != DependencyKindTool || resolved.Value != "rojo-rbx/rojo@7.6.1" {
		t.Fatalf("unexpected tool resolution: %+v", resolved)
	}
}

func TestResolvePkgPrefix(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve("pkg:sleitnick/knit")
	if err != nil {
		t.Fatalf("expected pkg prefix resolution success, got: %v", err)
	}
	if resolved.Kind != DependencyKindPackage || resolved.Value != "sleitnick/knit" {
		t.Fatalf("unexpected package resolution: %+v", resolved)
	}
}

func TestResolveKnownAlias(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve("rojo")
	if err != nil {
		t.Fatalf("expected alias resolution success, got: %v", err)
	}
	if resolved.Kind != DependencyKindTool || resolved.Value != "rojo-rbx/rojo@7.6.1" {
		t.Fatalf("unexpected alias resolution: %+v", resolved)
	}
}

func TestResolveUnknownToolWithoutVersion(t *testing.T) {
	t.Parallel()

	_, err := Resolve("tool:my-org/my-tool")
	if err == nil {
		t.Fatal("expected invalid dependency input for missing tool version")
	}
	if !errors.Is(err, ErrInvalidDependencyInput) {
		t.Fatalf("expected ErrInvalidDependencyInput, got: %v", err)
	}
}

func TestResolveConfidentPackage(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve("sleitnick/knit")
	if err != nil {
		t.Fatalf("expected package heuristic success, got: %v", err)
	}
	if resolved.Kind != DependencyKindPackage || resolved.Value != "sleitnick/knit" {
		t.Fatalf("unexpected package heuristic resolution: %+v", resolved)
	}
}

func TestResolveAmbiguousValue(t *testing.T) {
	t.Parallel()

	_, err := Resolve("knit")
	if err == nil {
		t.Fatal("expected ambiguous error")
	}
	if !errors.Is(err, ErrAmbiguousDependency) {
		t.Fatalf("expected ErrAmbiguousDependency, got: %v", err)
	}
}

func TestResolveUnknownPrefix(t *testing.T) {
	t.Parallel()

	_, err := Resolve("plugin:abc")
	if err == nil {
		t.Fatal("expected unknown dependency kind error")
	}
	if !errors.Is(err, ErrUnknownDependencyKind) {
		t.Fatalf("expected ErrUnknownDependencyKind, got: %v", err)
	}
}
