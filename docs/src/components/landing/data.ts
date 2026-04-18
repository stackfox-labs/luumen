import type { CommandSection } from "./types"

export const SECTIONS: CommandSection[] = [
  {
    id: "create",
    label: "create",
    sub: "LUU CREATE",
    heading: "Scaffold in seconds",
    command: "luu create my-game",
    points: [
      "Generates all config files in one shot",
      "Initializes Rokit, Wally, and Rojo configs",
      "Installs tools and packages immediately",
      "Skip install with --no-install for CI use",
    ],
    lines: [
      { text: "  [luu] Creating project: my-game", kind: "muted" },
      { text: "  [luu] Using template: rojo-wally", kind: "muted" },
      { text: "", kind: "blank" },
      { text: "  [luu] Scaffolded project at ./my-game", kind: "muted" },
      { text: "  [luu] Installing tools with Rokit...", kind: "muted" },
      { text: "  [ok] Tools installed", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  [luu] Installing packages with Wally...", kind: "muted" },
      { text: "  [ok] Packages installed", kind: "success" },
      { text: "  [ok] Project created", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  [next] Setup complete", kind: "accent" },
      { text: "  [next] cd my-game", kind: "accent" },
      { text: "  [next] luu dev", kind: "accent" },
    ],
  },
  {
    id: "install",
    label: "install",
    sub: "LUU INSTALL",
    heading: "One install for everything",
    command: "luu install",
    points: [
      "Runs rokit install for all dev tools",
      "Runs wally install for runtime packages",
      "Works without project.config.luau when possible",
      "Fine-grained control with --tools / --packages",
    ],
    lines: [
      { text: "  [luu] Resolving install scope...", kind: "muted" },
      { text: "  [luu] Installing tools with Rokit...", kind: "muted" },
      { text: "  [ok] Tools installed", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  [luu] Installing packages with Wally...", kind: "muted" },
      { text: "  [ok] Packages installed", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  [ok] Install completed", kind: "success" },
    ],
  },
  {
    id: "add",
    label: "add",
    sub: "LUU ADD",
    heading: "Add tools and packages",
    command: "luu add selene",
    points: [
      "Resolves packages vs tools automatically",
      "Built-in aliases: rojo, stylua, selene, wally",
      "Installs immediately after updating config",
      "Explicit tool: and pkg: prefixes available",
    ],
    lines: [
      { text: "  [luu] Resolving dependency: selene", kind: "muted" },
      { text: "  [luu] Running: rokit add kampfkarren/selene", kind: "plain" },
      { text: "", kind: "blank" },
      { text: "  Added version 0.30.1 of tool selene", kind: "plain" },
      { text: "  [luu] Tool added and installed successfully", kind: "success" },
    ],
  },
  {
    id: "dev",
    label: "dev",
    sub: "LUU DEV",
    heading: "Your full dev workflow",
    command: "luu dev",
    points: [
      "Runs your repo's commands.dev sequence",
      "Default flow: sourcemap generation then Rojo serve",
      "Fully overridable per-repo via project.config.luau",
      "Pairs naturally with luu lint and luu format",
    ],
    lines: [
      { text: "  [luu] workspace: my-game", kind: "muted" },
      { text: "  [luu] command: dev", kind: "muted" },
      { text: "  [luu] resolved: 2 steps", kind: "muted" },
      { text: "", kind: "blank" },
      { text: "  [luu] step 1/2: rojo sourcemap default.project.json --output sourcemap.json", kind: "plain" },
      { text: "", kind: "blank" },
      { text: "  [luu] step 2/2: rojo serve default.project.json", kind: "plain" },
      { text: "  Rojo server listening: http://localhost:34872", kind: "accent" },
    ],
  },
  {
    id: "lint",
    label: "lint",
    sub: "LUU LINT",
    heading: "Built-in quality checks",
    command: "luu lint",
    points: [
      "Runs commands.lint from project.config.luau",
      "Use luu format for formatter workflows",
      "No custom task names needed for standard checks",
      "Keep luu run <task> for project-specific automation",
    ],
    lines: [
      { text: "  [luu] workspace: my-game", kind: "muted" },
      { text: "  [luu] command: lint", kind: "muted" },
      { text: "  [luu] running: selene src", kind: "plain" },
      { text: "", kind: "blank" },
      { text: "  Results:", kind: "plain" },
      { text: "  0 errors", kind: "success" },
      { text: "  0 warnings", kind: "success" },
      { text: "  0 parse errors", kind: "success" },
    ],
  },
  {
    id: "doctor",
    label: "doctor",
    sub: "LUU DOCTOR",
    heading: "Health check your repo",
    command: "luu doctor",
    points: [
      "Validates project.config.luau and rokit.toml",
      "Checks required executables in PATH",
      "Surfaces actionable warnings and next steps",
      "Summarizes pass, warning, and error counts",
    ],
    lines: [
      { text: "  [luu] Running health checks...", kind: "muted" },
      { text: "", kind: "blank" },
      { text: "  pass: project.config.luau is valid. (luumen-config)", kind: "success" },
      { text: "  pass: rokit.toml is valid. (rokit-config)", kind: "success" },
      { text: "  pass: rokit executable found in PATH. (rokit-binary)", kind: "success" },
      { text: "  warning: No Rojo project file (*.project.json) found. (rojo-config)", kind: "accent" },
      { text: "  [next] Add a Rojo project file or run luu init in an adoptable repository.", kind: "accent" },
      { text: "", kind: "blank" },
      { text: "  summary", kind: "plain" },
      { text: "    pass: 3", kind: "success" },
      { text: "    warning: 1", kind: "accent" },
      { text: "    error: 0", kind: "success" },
    ],
  },
]

export const INSTALL_LINES = [
  { label: "MACOS / LINUX", cmd: "curl -fsSL https://luumen.dev/install.sh | bash" },
  { label: "WINDOWS (POWERSHELL)", cmd: "irm https://luumen.dev/install.ps1 | iex" },
]

export const FEATURES = [
  {
    title: "Manages your full stack",
    desc: "Luumen orchestrates Rokit, Wally, and Rojo together - one command installs your tools and packages without thinking about it.",
    pills: ["luu install", "rokit", "wally", "rojo"],
  },
  {
    title: "Standardizes every repo",
    desc: "One consistent command surface across all projects. Clone any Luumen repo and the workflow is immediately familiar.",
    pills: ["luu dev", "luu lint", "luu format", "luu test"],
  },
  {
    title: "Built-ins plus custom tasks",
    desc: "Use luu lint, luu format, and luu test for common workflows. Keep luu run <task> for project-specific pipelines.",
    pills: ["luu lint", "luu format", "luu run ci"],
  },
]

export const CONFIG_SOURCE = `return {
  project = {
    name = "my-game",
  },

  commands = {
    dev = {
      "rojo sourcemap default.project.json --output sourcemap.json",
      "rojo serve default.project.json",
    },
    lint = "selene src",
  },

  tasks = {
    check = {
      "luu lint",
      "luu format",
    },
  },
}`