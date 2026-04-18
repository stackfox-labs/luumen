package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"luumen/internal/config"
	"luumen/internal/workspace"
)

type Severity string

const (
	SeverityPass    Severity = "pass"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type CheckResult struct {
	ID         string
	Severity   Severity
	Message    string
	Suggestion string
}

type Report struct {
	Results  []CheckResult
	Passes   int
	Warnings int
	Errors   int
}

func (r Report) HasErrors() bool {
	return r.Errors > 0
}

type Runner struct {
	lookPath func(string) (string, error)
}

func NewRunner(lookPath func(string) (string, error)) *Runner {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	return &Runner{lookPath: lookPath}
}

func (r *Runner) Run(state workspace.Workspace) Report {
	report := Report{}

	if state.HasLuumenConfig {
		if _, err := config.Load(state.LuumenConfigPath); err != nil {
			report.add(SeverityError, "luumen-config", fmt.Sprintf("Invalid %s: %v", workspace.LuumenConfigFile, err), "Fix project.config.luau syntax and field types.")
		} else {
			report.add(SeverityPass, "luumen-config", fmt.Sprintf("%s is valid.", workspace.LuumenConfigFile), "")
		}
	}

	if state.HasRokitConfig {
		if err := validateTOMLFile(state.RokitConfigPath); err != nil {
			report.add(SeverityError, "rokit-config", fmt.Sprintf("Invalid %s: %v", workspace.RokitConfigFile, err), "Fix rokit.toml syntax.")
		} else {
			report.add(SeverityPass, "rokit-config", fmt.Sprintf("%s is valid.", workspace.RokitConfigFile), "")
		}
		r.checkExecutable(&report, "rokit", "rokit", "Install Rokit or add it to PATH.")
	}

	if state.HasWallyConfig {
		if err := validateTOMLFile(state.WallyConfigPath); err != nil {
			report.add(SeverityError, "wally-config", fmt.Sprintf("Invalid %s: %v", workspace.WallyConfigFile, err), "Fix wally.toml syntax.")
		} else {
			report.add(SeverityPass, "wally-config", fmt.Sprintf("%s is valid.", workspace.WallyConfigFile), "")
		}
		r.checkExecutable(&report, "wally", "wally", "Install Wally or add it to PATH.")
		r.checkWallyPackagesDir(&report, state.RootPath)
	}

	if state.HasRojoProject {
		r.checkExecutable(&report, "rojo", "rojo", "Install Rojo or add it to PATH.")
		for _, path := range state.RojoProjectPaths {
			if err := validateRojoProjectFile(path); err != nil {
				report.add(SeverityError, "rojo-config", fmt.Sprintf("Invalid Rojo project file %s: %v", filepath.Base(path), err), "Fix JSON syntax and required Rojo fields.")
			} else {
				report.add(SeverityPass, "rojo-config", fmt.Sprintf("Rojo project file %s is valid.", filepath.Base(path)), "")
			}
		}
	} else {
		report.add(SeverityWarning, "rojo-config", "No Rojo project file (*.project.json) found.", "Add a Rojo project file or run luu init in an adoptable repository.")
	}

	return report
}

func (r *Runner) checkExecutable(report *Report, id string, binary string, suggestion string) {
	if _, err := r.lookPath(binary); err != nil {
		report.add(SeverityError, id+"-binary", fmt.Sprintf("%s executable not found in PATH.", binary), suggestion)
		return
	}
	report.add(SeverityPass, id+"-binary", fmt.Sprintf("%s executable found in PATH.", binary), "")
}

func (r *Runner) checkWallyPackagesDir(report *Report, rootPath string) {
	packagesPath := filepath.Join(rootPath, "Packages")
	info, err := os.Stat(packagesPath)
	if errors.Is(err, os.ErrNotExist) {
		report.add(SeverityWarning, "wally-packages", "Packages directory is missing; dependencies may not be installed.", "Run luu install --packages.")
		return
	}
	if err != nil {
		report.add(SeverityWarning, "wally-packages", fmt.Sprintf("Unable to inspect Packages directory: %v", err), "Check filesystem permissions and run luu install --packages.")
		return
	}
	if !info.IsDir() {
		report.add(SeverityWarning, "wally-packages", "Packages path exists but is not a directory.", "Remove the conflicting file and run luu install --packages.")
		return
	}
	report.add(SeverityPass, "wally-packages", "Packages directory exists.", "")
}

func validateTOMLFile(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return nil
	}

	var doc map[string]any
	if err := toml.Unmarshal(contents, &doc); err != nil {
		return err
	}
	return nil
}

func validateRojoProjectFile(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return errors.New("file is empty")
	}

	var doc map[string]any
	if err := json.Unmarshal(contents, &doc); err != nil {
		return err
	}
	if _, ok := doc["tree"]; !ok {
		return errors.New("missing required \"tree\" root field")
	}
	return nil
}

func (r *Report) add(severity Severity, id string, message string, suggestion string) {
	result := CheckResult{ID: id, Severity: severity, Message: message, Suggestion: suggestion}
	r.Results = append(r.Results, result)

	switch severity {
	case SeverityPass:
		r.Passes++
	case SeverityWarning:
		r.Warnings++
	case SeverityError:
		r.Errors++
	}
}
