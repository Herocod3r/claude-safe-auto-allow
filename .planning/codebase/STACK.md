# Technology Stack

**Analysis Date:** 2026-03-15

## Languages

**Primary:**
- Go 1.21+ - Core application language for all production code
  - Builds cross-platform binaries (darwin/linux, amd64/arm64)
  - Single compiled binary with zero runtime dependencies

**Secondary:**
- Bash - Installation and uninstallation scripts
  - `install.sh`: Downloads binary, patches Claude settings.json
  - `uninstall.sh`: Removes hooks from settings

## Runtime

**Environment:**
- Go Runtime (built binaries, no external dependencies)
- Supported OS: macOS (darwin), Linux
- Supported Architectures: amd64 (x86_64), arm64 (aarch64)

**Package Manager:**
- Go Modules (go.mod)
- Lockfile: `go.mod` (minimal, no external dependencies)

## Frameworks

**Core:**
- Standard Library only
  - `encoding/json` - JSON marshaling/unmarshaling
  - `os` - File I/O, environment variables
  - `regexp` - Pattern matching for dangerous/safe commands
  - `strings` - String manipulation
  - `time` - Timestamp handling for command learning
  - `io` - stdin reading
  - `filepath` - Path operations
  - `fmt` - Output formatting

**Testing:**
- Standard Go `testing` package (built-in)
- No external test frameworks

**Build/Dev:**
- `go build` - Compile binaries with ldflags
- `go test` - Unit testing
- GitHub Actions (Actions CI/CD for release builds)

## Key Dependencies

**Critical:**
- None - Codebase has zero external runtime dependencies
- All functionality implemented with Go standard library only
- This enables single-binary distribution with minimal attack surface

**Infrastructure:**
- Bash (shell scripts for installation)
- Python3 or Node.js (optional, for settings.json patching during install)
  - Used in `install.sh` to modify `~/.claude/settings.json`
  - Falls back if Python unavailable

## Configuration

**Environment:**
- Binary location: `~/.claude-safe-auto-allow/claude-safety`
- Allowlist location: `~/.claude-safe-auto-allow/safety-allowlist.json`
- Version file: `~/.claude-safe-auto-allow/version.txt`
- Settings patched: `~/.claude/settings.json`

**Build:**
- `Makefile`: Not used, direct go build commands
- Compilation flags:
  - `-ldflags "-X main.buildVersion=$VERSION"` - Version embedding at build time
  - Version injected from git tag during CI/CD build

**Release Artifacts:**
- Cross-platform binaries: `claude-safety-{os}-{arch}`
  - `claude-safety-darwin-arm64`
  - `claude-safety-darwin-amd64`
  - `claude-safety-linux-arm64`
  - `claude-safety-linux-amd64`

## Platform Requirements

**Development:**
- Go 1.21 or later (only required for building from source)
- Bash shell (for build/install scripts)
- Git (for development)

**Production:**
- macOS (10.12+) or Linux (glibc 2.17+)
- No package manager required - single binary distribution
- Optional: Python3 or Node.js (for install script settings.json patching)

**Binary Size:**
- Single compiled executable (~3.5MB precompiled for darwin)

---

*Stack analysis: 2026-03-15*
