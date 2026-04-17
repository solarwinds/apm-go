# Releasing the library

Release checklist

 - Update `internal/utils/version.go` with new version. Create and merge PR.
 - Tag the release `git tag vX.X.X && git push --tags`
 - Create a [Github Release](https://github.com/solarwinds/apm-go/releases/new)

Future consideration: add another step, after the release is complete, to update
`version.go` with a prerelease name. If you released `v1.0.0`, perhaps the next
prerelease version would be `v1.0.1-beta`.

# Notes on major version bump

Per https://go.dev/wiki/Modules#releasing-modules-v2-or-higher, bumping to a 2.0.0 version would require corresponding module path and import path changes (i.e. updating all module imports from `github.com/solarwinds/apm-go` to `github.com/solarwinds/apm-go/v2`). In the interest of easier adoption we are currently making releases that do not break client API but are considered breaking capability changes such as removing AO support and legacy runtime metrics, without bumping to v2.
