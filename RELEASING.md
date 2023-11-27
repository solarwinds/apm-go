# Releasing the library

Release checklist

 - Update `internal/utils/version.go` with new version. Create and merge PR.
 - Tag the release `git tag vX.X.X && git push --tags`
 - Create a [Github Release](https://github.com/solarwinds/apm-go/releases/new)

Future consideration: add another step, after the release is complete, to update
`version.go` with a prerelease name. If you released `v0.2.0`, perhaps the next
prerelease version would be `v0.2.1-beta`.