# Contributing to solarwinds-apm-go

Thank you for contributing and helping us improve `solarwinds-apm-go`.

----

## Issues

### Security issues

Please report any security issues privately to the SolarWinds Product Security
Incident Response Team (PSIRT)
at [psirt@solarwinds.com](mailto:psirt@solarwinds.com).

### All other issues

For non-security issues, please submit your ideas, questions, or problems
as [GitHub issues](https://github.com/solarwinds/apm-go/issues).
Please add as much information as you can, such as: Go version, platform,
installed dependencies and their version numbers, hosting, code examples or
gists, steps to reproduce, stack traces, and logs. SolarWinds project
maintainers may ask for clarification or more context after submission.

----

## Contributing

Any changes to this project must be made through a pull request to `main`. Major
changes should be linked to an
existing [GitHub issue](https://github.com/solarwinds/apm-go/issues).
Smaller contributions like typo corrections don't require an issue.

A PR is ready to merge when all tests pass, any major feedback has been
resolved, and at least one SolarWinds maintainer has approved. Once ready, a PR
can be merged by a SolarWinds maintainer.

----

## Development

### Prerequisites

* Go (see supported versions in [README.md](README.md))
* openssl
* make

Run tests with `make testfast` which will only run the tests with related 
changes since the last run.

Run all tests with `make test`.

### Formatting and Linting

We enforce the following in CI:

* [`staticcheck ./...`](https://staticcheck.dev/docs/)
* [`go vet ./...`](https://pkg.go.dev/cmd/vet)
* [`gofmt`](https://pkg.go.dev/cmd/gofmt)
 
Please run these locally to assure your PRs do not cause checks to fail.

