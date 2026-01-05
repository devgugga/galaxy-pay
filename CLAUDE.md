# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Galaxy Pay is a Go-based payment service backend currently in early development. The project uses Go 1.25 and follows the standard Go project layout.

## Build and Development Commands

### Running the Application

```bash
# Build and run manually
go build -o ./tmp/main.exe ./cmd/app
./tmp/main.exe

# Run with Air (hot reload for development)
air

# Run directly
go run ./cmd/app
```

### Building

```bash
# Standard build
go build -o ./tmp/main.exe ./cmd/app

# Build for production (no CGO, Linux)
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./main ./cmd/app
```

### Docker

```bash
# Build Docker image
docker build -t galaxy-pay .

# Run container
docker run -p 8080:8080 galaxy-pay
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests in a specific package
go test ./internal/handlers

# Run a specific test
go test -run TestFunctionName ./path/to/package
```

### Dependencies

```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Architecture

### Project Structure

The project follows the standard Go project layout:

- `cmd/app/` - Application entry point, contains main.go with HTTP server setup
- `internal/` - Private application code not intended for external import
  - `config/` - Configuration management (currently empty, ready for implementation)
  - `handlers/` - HTTP request handlers (currently empty, ready for implementation)
  - `repositories/` - Data access layer (currently empty, ready for implementation)
  - `services/` - Business logic layer (currently empty, ready for implementation)
- `pkg/` - Public libraries that can be imported by external projects (currently empty)
- `tmp/` - Temporary build artifacts (excluded from git)

### Application Entry Point

The main application (`cmd/app/main.go`) sets up:
- HTTP server with graceful shutdown
- Three endpoints: `/health`, `/ready`, and `/` (root)
- Configurable port via PORT environment variable (defaults to 8080)
- Request timeouts: Read/Write 15s, Idle 60s
- Graceful shutdown with 30s timeout

### Dependencies

Key dependencies (from go.mod):
- `github.com/gofiber/fiber/v2` - Web framework (imported but not yet used in main.go)
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/google/uuid` - UUID generation

### Development Setup

The project uses:
- **Air** for hot reload during development (configured in `.air.toml`)
  - Watches `.go`, `.tpl`, `.tmpl`, `.html` files
  - Excludes `_test.go` files from triggering rebuilds
  - Build output: `./tmp/main.exe`
  - Build errors logged to `build-errors.log`
- **mise** for tool version management (Go latest)
- **Docker** for containerized deployment with multi-stage builds

### Architecture Patterns

This codebase is structured to follow a layered architecture:

1. **Handlers Layer** (`internal/handlers/`) - HTTP request handling, input validation, response formatting
2. **Services Layer** (`internal/services/`) - Business logic, orchestration between repositories
3. **Repositories Layer** (`internal/repositories/`) - Data access, database interactions
4. **Config Layer** (`internal/config/`) - Application configuration, environment variables

When implementing new features:
- HTTP handlers should be thin and delegate to services
- Business logic belongs in services, not handlers
- Database queries and data access belong in repositories
- Configuration should be centralized in the config package
