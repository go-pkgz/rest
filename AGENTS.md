# AGENTS.md

This file provides guidance to coding agents working with code in this repository.

## What this is

`github.com/go-pkgz/rest` is a library of HTTP middlewares and small helpers for REST services. It is not an application — there is no `main`, no server, no CLI. It is consumed by other projects (remark42, etc.). Keep the public API stable and minimal.

Dependencies are deliberately tiny: only `stretchr/testify` (tests) and `golang.org/x/crypto` (Argon2/bcrypt for BasicAuth). Do not add a routing framework or logging library dependency — middlewares are plain stdlib and must stay router-agnostic.

## Commands

- Test everything: `go test ./...`
- Single test: `go test -run TestName ./...` (e.g. `go test -run TestMiddleware_Ping ./...`)
- Race + coverage (mirrors CI): `TZ="America/Chicago" go test -timeout=60s -race -covermode=atomic ./...`
- Lint (run from repo root): `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0`

## Architecture

Three public packages and one internal:
- **`rest`** (root) — all middlewares plus JSON/error/file-server helpers.
- **`logger`** — request-logging middleware, split out so it can be wired to any backend via the `logger.Backend` interface (`Logf(format, args...)`). Configured with functional options (`logger.New(logger.Prefix(...), logger.WithBody, ...)`).
- **`realip`** — `realip.Get(r)` extracts the client IP from proxy headers. Used by both `rest.RealIP` and the `logger` package. Only public IPs are accepted from headers.
- **`internal/requestid`**: the request ID that `Trace` keeps in the request context, shared with the `logger` package, which can't import `rest` because `rest` imports it.

### Middleware conventions

Every middleware is a `func(http.Handler) http.Handler` (or a `func(...) func(http.Handler) http.Handler` when it takes config). This is the chi/stdlib-compatible shape — match it for any new middleware. Some middlewares short-circuit the chain (`Ping`, `Health`, metrics) by writing a response and returning without calling the next handler.

Configurable middlewares use the **functional-options pattern**, not option structs:
- `CORS(CorsAllowedOrigins(...), CorsAllowCredentials(true), ...)` — options named `CorsXxx`.
- `Secure(SecFrameOptions(...), SecHSTS(...), ...)` — options named `SecXxx`, plus `SecAllHeaders()` convenience.
- `logger.New(logger.Prefix(...), ...)` — options in `logger/options.go`.

When adding an option to one of these, follow the existing prefix/naming and keep defaults sensible so calling the constructor with no options is safe.

### CSRF build-tag split (important)

CSRF protection has **two implementations behind one identical public API**, selected by Go version:
- `csrf_go125.go` (`//go:build go1.25`) — thin wrapper over stdlib `http.CrossOriginProtection`.
- `csrf.go` (`//go:build !go1.25`) — self-contained equivalent for older Go.

`NewCrossOriginProtection`, `AddTrustedOrigin`, `AddBypassPattern`, `SetDenyHandler`, `Check`, `Handler` must exist with the **same signatures in both files**. When you change the CSRF API or behavior, edit both files and keep `csrf_test.go` passing under both build tags. The toolchain's version selects the file, not go.mod: any 1.25+ toolchain builds `csrf_go125.go`, and only a 1.24 toolchain (the `go.mod` entry of the CI matrix) builds `csrf.go`. The two differ in bypass-pattern syntax and conflict handling: the stdlib takes `http.ServeMux` patterns and panics on a conflicting one, the fallback matches exact paths and `/prefix/` only and accepts repeats. Run the tests with `GOTOOLCHAIN=go1.24.0` too when touching either file.

### Helpers

`rest.go` holds the JSON render/encode/decode helpers (`RenderJSON`, `EncodeJSON`/`DecodeJSON` generics, `RenderJSONWithHTML`) and `ParseFromTo`. `httperrors.go` has `SendErrorJSON`/`NewErrorLogger`. `file_server.go` has `NewFileServer` with `FsOpt*` options (`FsOptSPA`, `FsOptListing`, `FsOptCustom404`), listing off by default; `FileServer` and `FileServerSPA` are deprecated shortcuts. `benchmarks.go` keeps in memory one data point per second for the last 15 minutes by default (`WithTimeRange` changes it), queried via `Stats(duration)`.

## Conventions

- **One test file per source file**: `foo.go` → `foo_test.go` only. Table-driven with testify. Note `depricattion.go`/`depricattion_test.go` is misspelled but is the real filename — don't "fix" it without intent, it's the established path.
- After changing or adding a middleware/helper, update `README.md` — it documents every middleware and helper and is the primary user-facing doc.
- Lint config (`.golangci.yml`) is strict (`govet enable-all`, revive, gocritic with performance/style/experimental). `modernize` is enabled — prefer `any` over `interface{}`, `slices`/`maps` stdlib, etc.
- `errDetailsMsg` reads `runtime.Caller(2)` for its `[caused by file:line func]` suffix, so a test must call it through a closure (as `TestErrorDetailsMsg` does). A direct call makes frame 2 `testing.tRunner`, and under `go test -trimpath` its two-element path panics the 3-element slice in `httperrors.go`.
