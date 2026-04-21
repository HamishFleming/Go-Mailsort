# mailsort Agent Guidance

- Update `CONTEXT.md` with a synopsis and recent changes after each session.
- The module path in `go.mod` is set to a placeholder (`github.com/yourname/mailsort`) - update to the actual repository URL, then run `go mod tidy` before building.

## Commands

- `go build ./...` - build all packages
- `go test ./...` - run all tests