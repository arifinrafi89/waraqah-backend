# golangci-lint (tool)

**What:** Runs govet, staticcheck, errcheck and others in CI.

**Why not the standard library:** go vet alone misses unchecked errors and unused code.
