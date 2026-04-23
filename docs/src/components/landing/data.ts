import type { CommandSection } from "./types"

export const SECTIONS: CommandSection[] = [
  {
    id: "create",
    label: "create",
    sub: "LUU CREATE",
    heading: "Scaffold in seconds",
    command: "luu create my-game",
    points: [
      "Create a project in one command",
      "Sets up the tools and config for you",
      "Installs everything so you can start immediately",
      "Supports templates and --no-install when needed",
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
      "Install everything the project needs",
      "Handles tools and packages together",
      "Works in existing repos without extra setup",
      "Supports --tools and --packages when needed",
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
      "Add dependencies in one command",
      "Resolves common tools without full names",
      "Updates the project and installs immediately",
      "Supports tool: and pkg: prefixes when needed",
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
    heading: "Run your project",
    command: "luu dev",
    points: [
      "Run the project's main development workflow",
      "Supports multi-step flows like build and serve",
      "Uses the project's own configuration",
      "Same command across every repo",
    ],
    lines: [
      { text: "  [luu] workspace: my-game", kind: "muted" },
      { text: "  [luu] task: dev", kind: "muted" },
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
    heading: "Run linting in one command",
    command: "luu lint",
    points: [
      "Run linting in one command",
      "Uses whatever linter the project defines",
      "No need to remember tool-specific commands",
      "Fits into larger workflows like ci or check",
    ],
    lines: [
      { text: "  [luu] workspace: my-game", kind: "muted" },
      { text: "  [luu] task: lint", kind: "muted" },
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
    heading: "Check your setup",
    command: "luu doctor",
    points: [
      "Check for common setup issues",
      "Validates config and required tools",
      "Shows clear warnings and next steps",
      "Gives a quick summary you can act on",
    ],
    lines: [
      { text: "  [luu] Running health checks...", kind: "muted" },
      { text: "", kind: "blank" },
      { text: "  pass: .config.luau is valid. (luumen-config)", kind: "success" },
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
    title: "One CLI for your workflow",
    desc: "Create projects, install dependencies, and run workflows through a single interface.",
    pills: ["luu create", "luu install", "luu dev"],
  },
  {
    title: "Works with your existing tools",
    desc: "Use Rojo, Wally, and Rokit like before, without juggling commands.",
    pills: ["rokit", "wally", "rojo"],
  },
  {
    title: "Standard by default, flexible when needed",
    desc: "Use built-in commands for common workflows and define your own tasks when needed.",
    pills: ["luu lint", "luu format", "luu run ci"],
  },
]

export const CONFIG_SOURCE = `return {
  project = {
    name = "my-game",
    version = "0.1.0"
  },

  tasks = {
    dev = "rojo serve default.project.json",
    lint = "selene src",
    format = "stylua src",

    check = {
      "luu lint",
      "luu format"
    }
  },

  luu = {
    install = {
      tools = true,
      packages = true
    }
  }
}`
