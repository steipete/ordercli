---
summary: 'Release checklist for ordercli (GitHub release + Homebrew tap)'
---

# Releasing ordercli

Releases use an annotated `v<version>` tag and `.github/workflows/release.yml`, as in 0.1.0. GoReleaser publishes the binary archives and `checksums.txt`; the dependent job updates `steipete/homebrew-tap` and waits for its result. Title GitHub releases as `ordercli <version>`.

## Prepare and validate

- Start with a clean, current `main` and prepare the release on a task branch.
- Update `internal/version/version.go` and finalize `CHANGELOG.md` with the local release date, preserving contributor credits.
- Run `go build ./cmd/ordercli`, `go test ./... -coverprofile=cover.out`, `go tool cover -func=cover.out` (at least 75% total), `make lint`, and `goreleaser check`. Check formatting with `gofumpt -l .`.
- Run the Docker build and smoke commands in `.github/workflows/docker.yml`.
- Review and land the release commit. Wait for CI and Docker checks on that exact commit before tagging.

## Publish

- Confirm the version does not already exist as a local/remote tag or GitHub release.
- Create an annotated tag: `git tag -a v<version> -m "Release <version>"`.
- Push only that tag: `git push origin v<version>`.
- Watch the release workflow, including its Homebrew handoff, to completion.
- Set the GitHub release title to `ordercli <version>` and use the finalized changelog section as its body, with Highlights first and links to the downloads, checksums, and Homebrew formula. Supply the body using a file and `--notes-file`.

The release workflow can also be dispatched with the existing tag for recovery. Inspect partial uploads before retrying; do not move an already published tag. Release credentials stay in GitHub Actions secrets.

## Verify and finish

- Read the GitHub release and asset inventory back, download an archive and `checksums.txt`, and verify its SHA-256.
- Check both macOS archives with `codesign`, `spctl`, and `otool`. GoReleaser cross-compiles these binaries on Linux without Developer ID signing or notarization; the Go linker may apply an ad-hoc signature. Their minimum macOS version must match the README (macOS 13).
- Verify `go list -m github.com/steipete/ordercli@v<version>` through the Go proxy.
- Verify the Homebrew formula version and checksums, install or upgrade it, run `brew test steipete/tap/ordercli`, and check `ordercli --version`. See [the Homebrew playbook](releasing-homebrew.md).
- Open the next empty `## Unreleased` section, review and commit it, then leave `main` clean and synchronized with `origin/main`.
