# Releasing the library

## Standard release workflow

 1. Create a release branch, update `internal/utils/version.go` with the new version, and open a PR against `main`.
 2. Once the PR is merged, the [`release`](.github/workflows/release.yaml) workflow triggers automatically (it runs on push to `main` when `internal/utils/version.go` changes). It tags the commit, pushes the tag, and creates a draft [GitHub Release](https://github.com/solarwinds/apm-go/releases).
 3. Review the draft release notes and publish once the new version has been tested.

Future consideration: add another step, after the release is complete, to update
`version.go` with a prerelease name. If you released `v1.0.0`, perhaps the next
prerelease version would be `v1.0.1-beta`.

## Manual workflow (fallback)

The `release` workflow can also be triggered manually via `workflow_dispatch` if the automatic push trigger is unavailable or a re-run is needed:

 - Update `internal/utils/version.go` with new version. Create and merge PR.
 - Run the `release` workflow with the `version` input (e.g. `v1.2.3`). It tags the commit, pushes the tag, and creates a draft Github Release.

## Releasing swolambda

swolambda is a separate module and should be tagged and released independently.
Its module tag should use the same version number as the APM release.

 1. Once the new apm-go release is tested, run the `Lambda Update` workflow. It automatically opens a PR updating `go.mod`/`go.sum` for the Lambda instrumentation and examples to the new version. Review and merge that PR.
 2. Run the `release` workflow manually with the `swolambda_version` input set to the released apm-go version. It tags and pushes the swolambda module tag (`instrumentation/github.com/aws/aws-lambda-go/swolambda/vX.X.X`); no release is created for this step.

For example, if the APM release is `v1.3.7`, run the `release` workflow with `swolambda_version: v1.3.7`, which produces the tag `instrumentation/github.com/aws/aws-lambda-go/swolambda/v1.3.7`.

# Notes on major version bump

Per https://go.dev/wiki/Modules#releasing-modules-v2-or-higher, bumping to a v2.0.0 version would require corresponding module path and import path changes (i.e. updating all module imports from `github.com/solarwinds/apm-go` to `github.com/solarwinds/apm-go/v2`). In the interest of easier adoption we are currently making releases that do not break client API but are considered breaking capability changes such as removing AO support and legacy runtime metrics, without bumping to v2.
