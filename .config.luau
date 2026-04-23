return {
    project = {
        name = "Luumen",
        description = "The unified CLI for Luau development",
        license = "MIT",
    },

    tasks = {
        ["dev"] = "go run ./cmd/luu",
        ["format"] = "go fmt ./...",
        ["test"] = "go test -v ./...",
        ["docs:dev"] = "cd docs && npm run dev",
        ["docs:build"] = "cd docs && npm run build",
    },
}
