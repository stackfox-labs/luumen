package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tools"
)

type createCommandDeps struct {
	getwd          func() (string, error)
	writeConfig    func(path string, cfg *config.Config) error
	rokitInstaller rokitInstaller
	wallyInstaller wallyInstaller
	loadTemplates  func() ([]createTemplate, error)
}

func defaultCreateCommandDeps() createCommandDeps {
	return createCommandDeps{
		getwd:          os.Getwd,
		writeConfig:    config.Write,
		rokitInstaller: tools.NewRokit(nil, ""),
		wallyInstaller: tools.NewWally(nil, ""),
		loadTemplates:  loadBuiltinCreateTemplates,
	}
}

func newCreateCmd(deps createCommandDeps) *cobra.Command {
	if deps.getwd == nil {
		deps.getwd = os.Getwd
	}
	if deps.writeConfig == nil {
		deps.writeConfig = config.Write
	}
	if deps.rokitInstaller == nil {
		deps.rokitInstaller = tools.NewRokit(nil, "")
	}
	if deps.wallyInstaller == nil {
		deps.wallyInstaller = tools.NewWally(nil, "")
	}
	if deps.loadTemplates == nil {
		deps.loadTemplates = loadBuiltinCreateTemplates
	}

	var noInstall bool
	var template string
	var projectNameFlag string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new Luumen-enabled project",
		Long: "Create scaffolds a Luumen project from a template. Run with no arguments for " +
			"interactive prompts, or pass a name directly for non-interactive use.",
		Example: "luu create my-game\n" +
			"luu create\n" +
			"luu create --no-install my-game\n" +
			"luu create --name my-game --template rojo-wally\n" +
			"luu create --template minimal my-game",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templates, err := deps.loadTemplates()
			if err != nil {
				return fmt.Errorf("failed to load create templates: %w", err)
			}

			projectArg := strings.TrimSpace(projectNameFlag)
			selectedTemplate := strings.TrimSpace(template)
			selectedNoInstall := noInstall

			if interactive || shouldRunInteractiveCreate(cmd, args, projectNameFlag) {
				prompted, err := promptCreateOptions(cmd, templates, selectedTemplate)
				if err != nil {
					return err
				}
				projectArg = prompted.projectName
				selectedTemplate = prompted.templateName
				selectedNoInstall = prompted.noInstall
			} else {
				if len(args) == 1 && projectArg != "" {
					return fmt.Errorf("project name was provided twice. Next: use either create <name> or --name <name>, not both")
				}
				if len(args) == 1 {
					projectArg = strings.TrimSpace(args[0])
				}
				if projectArg == "" {
					return fmt.Errorf("project name is required. Next: run luu create <name>, luu create --name <name>, or run luu create for interactive setup")
				}
			}

			templateDefinition, ok := findCreateTemplate(templates, selectedTemplate)
			if !ok {
				available := strings.Join(createTemplateNames(templates), ", ")
				return fmt.Errorf("unsupported template %q. Available templates: %s", selectedTemplate, available)
			}

			if projectArg == "" || projectArg == "." || projectArg == string(filepath.Separator) {
				return fmt.Errorf("project name must be a non-empty path. Next: provide a project name like luu create my-game")
			}

			statusf(cmd, "Creating project: %s", projectArg)
			statusf(cmd, "Using template: %s", templateDefinition.Name)
			spacef(cmd)

			cwd, err := deps.getwd()
			if err != nil {
				return fmt.Errorf("failed to resolve current directory: %w. Next: verify filesystem permissions", err)
			}

			targetPath, err := resolveCreateTargetPath(cwd, projectArg)
			if err != nil {
				return err
			}

			if err := ensureTargetDoesNotExist(targetPath); err != nil {
				return err
			}

			runTemplateCommand := makeCreateTemplateCommandRunner(cmd)
			if err := scaffoldProjectFromTemplate(targetPath, filepath.Base(targetPath), templateDefinition, runTemplateCommand, deps.writeConfig); err != nil {
				return err
			}

			statusf(cmd, "Scaffolded project at %s", targetPath)

			installTools := templateDefinition.Project.Install.Tools
			installPackages := templateDefinition.Project.Install.Packages

			if selectedNoInstall {
				statusf(cmd, "Skipping dependency installation (--no-install)")
				nextStepsf(cmd, "Project scaffolded", fmt.Sprintf("cd %s", relativePathForShell(cwd, targetPath)), "luu install", "luu dev")
				return nil
			}

			if installTools {
				statusf(cmd, "Installing tools with Rokit...")
				if _, err := deps.rokitInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, targetPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("project scaffolded but tool install failed: Rokit executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("project scaffolded but tool install failed: %w", err)
				}
				successf(cmd, "Tools installed")
			}

			if installPackages {
				statusf(cmd, "Installing packages with Wally...")
				if _, err := deps.wallyInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, targetPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("project scaffolded but package install failed: Wally executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("project scaffolded but package install failed: %w", err)
				}
				successf(cmd, "Packages installed")
			}

			if !installTools && !installPackages {
				statusf(cmd, "Template does not define dependency installation")
			}

			successf(cmd, "Project created")
			nextStepsf(cmd, "Setup complete", fmt.Sprintf("cd %s", relativePathForShell(cwd, targetPath)), "luu dev")

			return nil
		},
	}

	cmd.Flags().BoolVar(&noInstall, "no-install", false, "skip post-scaffold dependency installation")
	cmd.Flags().StringVar(&template, "template", "rojo-wally", "project template to scaffold")
	cmd.Flags().StringVar(&projectNameFlag, "name", "", "project name (flags-first alternative to positional [name])")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "prompt for project name/template/install choices")

	return cmd
}

func makeCreateTemplateCommandRunner(cmd *cobra.Command) createTemplateCommandRunner {
	return func(command string, workingDir string) error {
		if isVerbose(cmd) {
			statusf(cmd, "Running scaffold command: %s", styleCommand(cmd.OutOrStdout(), command))
		}

		stdout, stderr := commandOutputWriters(cmd)
		_, err := process.RunShell(cmd.Context(), command, process.Options{
			WorkingDir: workingDir,
			Stdout:     stdout,
			Stderr:     stderr,
			Stdin:      cmd.InOrStdin(),
		})
		if err != nil {
			return err
		}

		return nil
	}
}

type createPromptResult struct {
	projectName  string
	templateName string
	noInstall    bool
}

func shouldRunInteractiveCreate(cmd *cobra.Command, args []string, projectNameFlag string) bool {
	if len(args) != 0 {
		return false
	}
	if strings.TrimSpace(projectNameFlag) != "" {
		return false
	}
	if cmd.Flags().Changed("name") || cmd.Flags().Changed("template") || cmd.Flags().Changed("no-install") || cmd.Flags().Changed("interactive") {
		return false
	}
	return true
}

func promptCreateOptions(cmd *cobra.Command, templates []createTemplate, defaultTemplate string) (createPromptResult, error) {
	writer := cmd.OutOrStdout()
	reader := bufio.NewReader(cmd.InOrStdin())
	prefix := promptPrefix(writer)
	useArrowSelector := canUseArrowTemplateSelector(cmd, writer)

	projectName, err := promptRequiredString(reader, writer, fmt.Sprintf("%s %s ", prefix, styleAccent(writer, "Project name:")), "project name")
	if err != nil {
		return createPromptResult{}, err
	}

	if !useArrowSelector {
		fmt.Fprintln(writer)
		fmt.Fprintf(writer, "%s %s\n", prefix, styleMuted(writer, "Available templates:"))
		for _, template := range templates {
			description := strings.TrimSpace(template.Description)
			if description == "" {
				description = "No description"
			}
			fmt.Fprintf(writer, "  %s %s %s\n", styleMuted(writer, "-"), styleAccent(writer, template.Name+":"), styleMuted(writer, description))
		}
	}

	templateName, err := promptTemplateSelection(cmd, reader, writer, templates, defaultTemplate)
	if err != nil {
		return createPromptResult{}, err
	}

	installNow, err := promptYesNo(reader, writer, fmt.Sprintf("%s %s", prefix, styleAccent(writer, "Install tools and packages now")), true)
	if err != nil {
		return createPromptResult{}, err
	}

	fmt.Fprintln(writer)

	return createPromptResult{
		projectName:  projectName,
		templateName: templateName,
		noInstall:    !installNow,
	}, nil
}

func promptRequiredString(reader *bufio.Reader, writer io.Writer, prompt string, fieldName string) (string, error) {
	value, err := readPromptLine(reader, writer, prompt)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("interactive create cancelled: no input received for %s", strings.ToLower(fieldName))
		}
		return "", fmt.Errorf("failed to read %s: %w", strings.ToLower(fieldName), err)
	}
	if value == "" {
		return "", fmt.Errorf("%s is required. Next: run luu create and provide a value", strings.ToLower(fieldName))
	}
	return value, nil
}

func promptTemplateSelection(cmd *cobra.Command, reader *bufio.Reader, writer io.Writer, templates []createTemplate, defaultTemplate string) (string, error) {
	if strings.TrimSpace(defaultTemplate) == "" {
		defaultTemplate = "rojo-wally"
	}

	if selected, usedArrows, err := promptTemplateSelectionWithArrows(cmd, writer, templates, defaultTemplate); usedArrows || err != nil {
		if err != nil {
			return "", err
		}
		return selected, nil
	}

	response, err := readPromptLine(reader, writer, fmt.Sprintf("%s %s ", promptPrefix(writer), styleAccent(writer, fmt.Sprintf("Template [%s]:", defaultTemplate))))
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("interactive create cancelled: no input received for template")
		}
		return "", fmt.Errorf("failed to read template selection: %w", err)
	}

	chosen := strings.TrimSpace(response)
	if chosen == "" {
		chosen = defaultTemplate
	}

	if _, ok := findCreateTemplate(templates, chosen); !ok {
		available := strings.Join(createTemplateNames(templates), ", ")
		return "", fmt.Errorf("unknown template %q. Available templates: %s", chosen, available)
	}

	return chosen, nil
}

func promptTemplateSelectionWithArrows(cmd *cobra.Command, writer io.Writer, templates []createTemplate, defaultTemplate string) (string, bool, error) {
	if len(templates) == 0 {
		return "", false, fmt.Errorf("no templates are available")
	}

	inputFile, ok := cmd.InOrStdin().(*os.File)
	if !ok || !canUseArrowTemplateSelector(cmd, writer) {
		return "", false, nil
	}

	selectedIndex := 0
	for index, template := range templates {
		if template.Name == defaultTemplate {
			selectedIndex = index
			break
		}
	}

	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "%s %s\n", promptPrefix(writer), styleMuted(writer, "Select template (use ↑/↓ and Enter):"))

	if _, err := fmt.Fprint(writer, "\x1b[?25l"); err == nil {
		defer fmt.Fprint(writer, "\x1b[?25h")
	}

	previousState, err := term.MakeRaw(int(inputFile.Fd()))
	if err != nil {
		return "", true, fmt.Errorf("failed to start interactive template picker: %w", err)
	}
	defer term.Restore(int(inputFile.Fd()), previousState)

	terminalWidth := 0
	if outputFile, ok := writer.(*os.File); ok {
		if width, _, sizeErr := term.GetSize(int(outputFile.Fd())); sizeErr == nil {
			terminalWidth = width
		}
	}

	input := bufio.NewReader(inputFile)
	redraw := false
	for {
		if err := renderTemplateSelection(writer, templates, selectedIndex, redraw, terminalWidth); err != nil {
			return "", true, fmt.Errorf("failed to render template picker: %w", err)
		}
		redraw = true

		key, err := input.ReadByte()
		if err != nil {
			return "", true, fmt.Errorf("failed to read template selection: %w", err)
		}

		switch key {
		case '\r', '\n':
			fmt.Fprintln(writer)
			return templates[selectedIndex].Name, true, nil
		case 3:
			fmt.Fprintln(writer)
			return "", true, fmt.Errorf("interactive create cancelled. Next: rerun luu create and choose a template")
		case 0x1b:
			next, err := input.ReadByte()
			if err != nil {
				return "", true, fmt.Errorf("failed to read template selection: %w", err)
			}
			if next != '[' {
				continue
			}

			code, err := input.ReadByte()
			if err != nil {
				return "", true, fmt.Errorf("failed to read template selection: %w", err)
			}

			switch code {
			case 'A':
				selectedIndex = (selectedIndex - 1 + len(templates)) % len(templates)
			case 'B':
				selectedIndex = (selectedIndex + 1) % len(templates)
			}
		}
	}
}

func renderTemplateSelection(writer io.Writer, templates []createTemplate, selectedIndex int, redraw bool, terminalWidth int) error {
	if redraw {
		if _, err := fmt.Fprintf(writer, "\x1b[%dA", len(templates)); err != nil {
			return err
		}
	}

	for index, template := range templates {
		displayName := strings.TrimSpace(template.Name)
		description := strings.TrimSpace(template.Description)
		if description == "" {
			description = "No description"
		}

		displayName, description = fitTemplateRow(displayName, description, terminalWidth)

		if _, err := fmt.Fprint(writer, "\x1b[2K\r"); err != nil {
			return err
		}

		marker := styleMuted(writer, "  ")
		name := displayName
		if index == selectedIndex {
			marker = styleAccent(writer, "◇ ")
			name = styleAccent(writer, name)
		}

		if _, err := fmt.Fprintf(writer, "%s%s %s\n", marker, name, styleMuted(writer, "- "+description)); err != nil {
			return err
		}
	}

	return nil
}

func canUseArrowTemplateSelector(cmd *cobra.Command, writer io.Writer) bool {
	inputFile, inOK := cmd.InOrStdin().(*os.File)
	outputFile, outOK := writer.(*os.File)
	if !inOK || !outOK {
		return false
	}

	if !term.IsTerminal(int(inputFile.Fd())) || !term.IsTerminal(int(outputFile.Fd())) {
		return false
	}

	return supportsInteractiveCursorControl()
}

func supportsInteractiveCursorControl() bool {
	if runtime.GOOS != "windows" {
		return true
	}

	if strings.TrimSpace(os.Getenv("WT_SESSION")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("ANSICON")) != "" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ConEmuANSI")), "ON") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM_PROGRAM")), "vscode") {
		return true
	}

	termName := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	return strings.Contains(termName, "xterm") || strings.Contains(termName, "ansi") || strings.Contains(termName, "vt")
}

func fitTemplateRow(name string, description string, terminalWidth int) (string, string) {
	if terminalWidth <= 0 {
		return name, description
	}

	budget := terminalWidth - 6
	if budget <= 1 {
		return truncateText(name, 1), ""
	}

	nameLen := len([]rune(name))
	if nameLen >= budget {
		return truncateText(name, budget), ""
	}

	sepLen := 3
	descBudget := budget - nameLen - sepLen
	if descBudget <= 0 {
		return name, ""
	}

	return name, truncateText(description, descBudget)
}

func truncateText(value string, limit int) string {
	if limit <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return "…"
	}

	return string(runes[:limit-1]) + "…"
}

func promptYesNo(reader *bufio.Reader, writer io.Writer, label string, defaultYes bool) (bool, error) {
	defaultPrompt := "[y/N]"
	if defaultYes {
		defaultPrompt = "[Y/n]"
	}

	response, err := readPromptLine(reader, writer, fmt.Sprintf("%s %s: ", label, styleMuted(writer, defaultPrompt)))
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, fmt.Errorf("interactive create cancelled: no input received for install selection")
		}
		return false, fmt.Errorf("failed to read install selection: %w", err)
	}

	choice := strings.ToLower(strings.TrimSpace(response))
	if choice == "" {
		return defaultYes, nil
	}

	switch choice {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid choice %q. Next: answer y or n", choice)
	}
}

func readPromptLine(reader *bufio.Reader, writer io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(writer, prompt); err != nil {
		return "", err
	}

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	value := strings.TrimSpace(line)
	if errors.Is(err, io.EOF) && value == "" {
		return "", io.EOF
	}

	return value, nil
}

func relativePathForShell(cwd string, targetPath string) string {
	rel, err := filepath.Rel(cwd, targetPath)
	if err != nil || strings.TrimSpace(rel) == "" {
		return quoteIfNeeded(targetPath)
	}

	rel = filepath.Clean(rel)
	if rel == "." {
		return "."
	}

	return quoteIfNeeded(rel)
}

func quoteIfNeeded(path string) string {
	if strings.Contains(path, " ") {
		return fmt.Sprintf("\"%s\"", path)
	}
	return path
}

func resolveCreateTargetPath(cwd string, projectArg string) (string, error) {
	var combined string
	if filepath.IsAbs(projectArg) {
		combined = projectArg
	} else {
		combined = filepath.Join(cwd, projectArg)
	}

	targetPath, err := filepath.Abs(combined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project path %q: %w", projectArg, err)
	}

	return targetPath, nil
}

func ensureTargetDoesNotExist(targetPath string) error {
	_, err := os.Stat(targetPath)
	if err == nil {
		return fmt.Errorf("destination already exists: %s", targetPath)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect destination %s: %w", targetPath, err)
	}
	return nil
}

func scaffoldMinimalProject(targetPath string, projectName string, runCommand createTemplateCommandRunner, writeConfig func(path string, cfg *config.Config) error) (createTemplateInstall, error) {
	templates, err := loadBuiltinCreateTemplates()
	if err != nil {
		return createTemplateInstall{}, fmt.Errorf("failed to load minimal template: %w", err)
	}

	template, ok := findCreateTemplate(templates, "minimal")
	if !ok {
		return createTemplateInstall{}, fmt.Errorf("failed to load minimal template: template \"minimal\" not found")
	}

	if err := scaffoldProjectFromTemplate(targetPath, projectName, template, runCommand, writeConfig); err != nil {
		return createTemplateInstall{}, err
	}

	return template.Project.Install, nil
}

var packageNameSanitizer = regexp.MustCompile(`[^a-z0-9-_]+`)

func normalizePackageName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	normalized := packageNameSanitizer.ReplaceAllString(lower, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "project"
	}
	return normalized
}
