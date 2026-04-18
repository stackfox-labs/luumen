return {
    project = {
        name = "Luumen",
    },

    tasks = {
        ["dev"] = "go run ./cmd/luu",
        ["format"] = "go fmt ./...",
        ["test"] = "go test -v ./...",
        ["docs:dev"] = "cd docs && npm run dev",
        ["docs:build"] = "cd docs && npm run build",
    }
}
