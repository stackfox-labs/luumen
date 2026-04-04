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
      { text: "  Creating Luumen project my-game...", kind: "muted" },
      { text: "", kind: "blank" },
      { text: "  ✓  Scaffolded project structure", kind: "success" },
      { text: "  ✓  Initialized rokit.toml", kind: "success" },
      { text: "  ✓  Initialized wally.toml", kind: "success" },
      { text: "  ✓  Generated default.project.json", kind: "success" },
      { text: "  ✓  Generated luumen.toml", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  ✓  rojo-rbx/rojo@7.4.1  (Rokit)", kind: "success" },
      { text: "  ✓  sleitnick/knit@1.5.1  (Wally)", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  → Next: cd my-game && luu dev", kind: "accent" },
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
      "Works without luumen.toml when possible",
      "Fine-grained control with --tools / --packages",
    ],
    lines: [
      { text: "  Installing tools via Rokit...", kind: "muted" },
      { text: "  ✓  rojo-rbx/rojo@7.4.1", kind: "success" },
      { text: "  ✓  UpliftGames/wally@0.3.2", kind: "success" },
      { text: "  ✓  JohnnyMorganz/StyLua@0.20.0", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  Installing packages via Wally...", kind: "muted" },
      { text: "  ✓  sleitnick/knit@1.5.1", kind: "success" },
      { text: "  ✓  evaera/promise@4.0.0", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  All dependencies installed.", kind: "plain" },
    ],
  },
  {
    id: "dev",
    label: "dev",
    sub: "LUU DEV",
    heading: "Your full dev workflow",
    command: "luu dev",
    points: [
      "Generates sourcemaps before serving",
      "Starts the Rojo server for Studio sync",
      "Fully overridable per-repo via luumen.toml",
      "Separate from luu serve for granular control",
    ],
    lines: [
      { text: "  Generating sourcemap...", kind: "muted" },
      { text: "  ✓  sourcemap.json written", kind: "success" },
      { text: "", kind: "blank" },
      { text: "  Starting Rojo server...", kind: "muted" },
      { text: "  Rojo v7.4.1 server listening:", kind: "plain" },
      { text: "  http://localhost:34872", kind: "accent" },
      { text: "", kind: "blank" },
      { text: "  Watching for file changes...", kind: "muted" },
    ],
  },
  {
    id: "add",
    label: "add",
    sub: "LUU ADD",
    heading: "Add tools and packages",
    command: "luu add sleitnick/knit",
    points: [
      "Resolves packages vs tools automatically",
      "Built-in aliases: rojo, stylua, selene, wally",
      "Installs immediately after updating config",
      "Explicit tool: and pkg: prefixes available",
    ],
    lines: [
      { text: "  Resolving sleitnick/knit...", kind: "muted" },
      { text: "  Identified as Wally package.", kind: "plain" },
      { text: "", kind: "blank" },
      { text: "  ✓  Added to wally.toml", kind: "success" },
      { text: "  Installing...", kind: "muted" },
      { text: "  ✓  sleitnick/knit@1.5.1 installed.", kind: "success" },
    ],
  },
  {
    id: "run",
    label: "run",
    sub: "LUU RUN",
    heading: "Custom task runner",
    command: "luu run lint",
    points: [
      "Runs tasks defined in [tasks] in luumen.toml",
      "Supports sequential array composition",
      "Tasks run through a shell - any binary works",
      "Composes naturally: luu run ci, luu run check",
    ],
    lines: [
      { text: "  Running task: lint", kind: "muted" },
      { text: "  → selene src", kind: "plain" },
      { text: "", kind: "blank" },
      { text: "  Checking src/...", kind: "muted" },
      { text: "  ✓  No issues found.", kind: "success" },
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
    pills: ["rokit", "wally", "rojo"],
  },
  {
    title: "Standardizes every repo",
    desc: "One consistent command surface across all projects. Clone any Luumen repo and the workflow is immediately familiar.",
    pills: ["luu install", "luu dev", "luu build"],
  },
  {
    title: "Custom task system",
    desc: "Define your workflows in luumen.toml. Tasks are shell commands or sequential arrays - like npm scripts, without the complexity.",
    pills: ["luu run lint", "luu run fmt"],
  },
]

export const CONFIG_SOURCE = `[project]
name = "my-game"

[install]
tools    = true
packages = true

[commands]
dev   = ["luu sourcemap", "rojo serve"]
build = "rojo build default.project.json"

[tasks]
lint  = "selene src"
fmt   = "stylua src"
ci    = ["luu run lint", "luu run fmt"]`
