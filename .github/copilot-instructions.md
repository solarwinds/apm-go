# GitHub Copilot Instructions

## Priority Guidelines

When generating code for this repository:

1. **Version Compatibility**: Always detect and respect the exact versions of languages, frameworks, and libraries used in this project
2. **Context Files**: Prioritize patterns and standards defined in the .github/instructions directory
3. **Codebase Patterns**: When context files don't provide specific guidance, scan the codebase for established patterns
4. **Architectural Consistency**: Maintain our layered architectural style and established boundaries
5. **Code Quality**: Prioritize maintainability, performance, security, and testability in all generated code

## Repository Purpose

The apm-go repository is a Go instrumentation library that provides Application Performance Monitoring (APM) capabilities for SolarWinds Observability. It wraps and extends the OpenTelemetry SDK to deliver production-ready observability with SolarWinds-specific features and optimizations.

## Golang Versions

- Support the **two most recent stable Go versions** (following OpenTelemetry SDK compatibility policy)
- Check `go.mod` for the minimum supported Go version (e.g., `go 1.21`)
- Never use language features from Go versions newer than what's specified in `go.mod`
- When suggesting code, ensure compatibility with the minimum version defined in the project

## Architecture Overview

### Architectural Boundaries

1. **Public API (`swo/`)**:
   - Minimal, clean API surface for end users
   - Main entry points: `swo.Start()`, `swo.StartLambda()`
   - Logging utilities: `swo.NewLogHandler()`
   - Export only what's necessary for library consumers

2. **Internal Implementation (`internal/`)**:
   - Not accessible to external users (Go internal package visibility)
   - Contains all implementation details
   - Well-organized into focused, single-responsibility packages
   - Each package has a clear purpose (oboe for sampling, config for configuration, etc.)

3. **Instrumentation (`instrumentation/`)**:
   - Framework-specific wrappers and helpers
   - Follows OpenTelemetry contrib naming and structure patterns
   - Minimal, focused on seamless integration

4. **Examples (`examples/`)**:
   - Demonstrate real-world usage patterns
   - Should be runnable with `go run`
   - Include both basic and advanced scenarios
