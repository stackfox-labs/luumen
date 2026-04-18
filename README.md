<img width="1150" height="881" alt="Luumen The Unified CLI for Roblox Development" src="https://github.com/user-attachments/assets/f652e297-5bcd-4d65-b7f7-f7d6fb0ea9a1" />

---

# Luumen

Luumen is a unified CLI for Roblox developers.

It provides one command surface for common project workflows while using the existing Roblox tooling ecosystem under the hood.

## Why Luumen

Roblox projects built outside Studio often rely on several separate tools for setup, dependency management, serving, builds, formatting, and linting.

Luumen exists to make those workflows easier to run and easier to standardize across repos.

Instead of learning a different set of setup steps and commands for every project, a repo can expose one consistent interface through `luu`.

## What Luumen uses

Luumen works with the existing ecosystem:

* **Rokit** for tool installation and management
* **Wally** for package dependencies
* **Rojo**, **Selene**, **StyLua**, **Lune**, and other tools for project workflows

It does not replace those tools.

## Installation

Install Luumen with the hosted installer scripts.

macOS and Linux:

```bash
curl -fsSL https://luumen.dev/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://luumen.dev/install.ps1 | iex
```

or manually download the latest release from the [Releases](https://github.com/stackfox-labs/luumen/releases) page and add it to your PATH.

Verify installation:

```bash
luu --version
```

## Commands

```bash
luu create
luu init
luu install
luu add
luu dev
luu build
luu lint
luu format
luu test
luu run <task>
luu doctor
```

## Examples

Create a project:

```bash
luu create my-game
cd my-game
luu dev
```

Create from a specific template:

```bash
luu create --template rojo-wally my-game
luu create --template rojo-only my-game
luu create --template lune-http my-script
luu create --template lute-guessing my-script
```

Set up an existing repo:

```bash
cd existing-repo
luu init
luu install
luu dev
```

Add a tool or package:

```bash
luu add rojo
luu add stylua
luu add sleitnick/knit
luu add tool:rojo-rbx/rojo
luu add pkg:sleitnick/knit
```

## Configuration

Luumen uses a shared Luau config file:

```text
project.config.luau
```

Instead of spreading configuration across multiple formats, `project.config.luau` provides a single place to describe how your project runs.

```lua
return {
    project = {
        name = "my-game",
    },

    install = {
        tools = true,
        packages = true,
    },

    tasks = {
        dev = {
            "rojo sourcemap default.project.json --output sourcemap.json",
            "rojo serve default.project.json",
        },
        build = "rojo build default.project.json --output build.rbxl",
        lint = "selene src",
        format = "stylua src",
        test = "lune run test",
        check = {
            "luu lint",
            "luu format",
            "luu test",
        },
    },
}
```

`project.config.luau` is designed as a shared standard for the Luau ecosystem.

Luumen reads from it, but other tools can adopt the same file and extend it with their own sections, allowing project configuration to live in one place instead of being split across multiple configs.

Tool-specific files such as `rokit.toml`, `wally.toml`, and Rojo project files are still used by their respective tools.

## Status

Luumen is currently in version 0.1.0.

## License

MIT License. See [LICENSE](LICENSE).
