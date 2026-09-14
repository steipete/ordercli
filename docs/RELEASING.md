---
summary: 'Release checklist for ordercli (GitHub release + Homebrew tap)'
---

# Releasing ordercli

Dispatch `.github/workflows/release.yml` on the current protected `main` with a SemVer `version`. The caller pins the shared Go CLI archetype at v1.9.0 (`f613cbfed2b043159c850c353e7facb8c89833b0`). It freezes the source, creates the annotated `v<version>` tag, builds with GoReleaser, signs and notarizes macOS binaries, verifies on native arm64 and Intel runners, then publishes and updates `steipete/homebrew-tap`.

The signing policy is `personal`: Developer ID Application: Peter Steinberger, team `Y5PE65HELJ`, identifier `com.steipete.ordercli.ordercli`. Native archives retain `ordercli_<version>_<os>_<arch>` names, tar.gz/Windows ZIP formats, and the README, license, and changelog. The release includes `checksums.txt`, `ASSET-INVENTORY.json`, `SIGNING-MANIFEST.json`, and `RELEASE-NOTES.md`; independent verifier attestations bind publication to the frozen source and exact signed bytes.

Repository setup requires protected main with required CI checks, Actions `default_workflow_permissions=write`, and `can_approve_pull_request_reviews=true` for the closeout PR. Workflows declare their own least-privilege permissions. Provision `MACOS_SIGN_P12`, `MACOS_SIGN_P12_PASSWORD`, `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_PRIVATE_KEY`, and `HOMEBREW_TAP_TOKEN` as repository secrets. The caller maps these to the shared workflow's signing, notary, and tap secrets; no credentials belong in source.

## Prepare and validate

- Start with a clean, current `main` and prepare the release on a task branch.
- Update `internal/version/version.go` and finalize `CHANGELOG.md` with the local release date, preserving contributor credits.
- Run `./scripts/check-release-metadata <version>` to check the requested version against the Go constant and latest finalized changelog section.
- Run `go build ./cmd/ordercli`, `go test ./... -coverprofile=cover.out`, `go tool cover -func=cover.out` (at least 75% total), `make lint`, and `goreleaser check`. Check formatting with `gofumpt -l .`.
- Run the Docker build and smoke commands in `.github/workflows/docker.yml`.
- Run `actionlint`, `python3 -m unittest discover -s scripts -p 'test_*.py'`, and `goreleaser build --snapshot --clean` on macOS. The same build runs in CI.
- Review and land the release commit. Wait for CI, macOS release-build, and Docker checks on that exact commit before dispatching.

The release builds on macOS so the GoReleaser post-build hook can inspect each actual Darwin binary with `otool -arch all -l`. Every `LC_BUILD_VERSION` must have `minos 13.0`, matching the README. GoReleaser sets `MACOSX_DEPLOYMENT_TARGET=13.0` and `CGO_ENABLED=0`; the Go 1.27 linker supplies this minimum for pure Go. If cgo is introduced, also set explicit `-mmacosx-version-min=13.0` in both `CGO_CFLAGS` and `CGO_LDFLAGS`; the environment variable alone is not sufficient proof.

## Publish

- Confirm the version does not already exist as a local/remote tag or GitHub release.
- Dispatch `gh workflow run release.yml --ref main -f version=<version>`. The workflow owns annotated tag creation; do not create or push a tag manually.
- Watch the release workflow, including its Homebrew handoff, to completion.
- The shared workflow publishes the finalized changelog section as both the release body and `RELEASE-NOTES.md`. Check it against `./scripts/release-notes v<version>`; preserve those exact notes after publication.

For recovery, rerun failed jobs on the original run. The shared workflow also supports redispatch from current main with an existing annotated version tag, reusing its frozen commit; the caller's current metadata must still match that version. Inspect partial uploads before retrying and never move an existing tag or rebuild an already published release. Release credentials stay in GitHub Actions secrets.

## Verify and finish

- Read the GitHub release and asset inventory back, download an archive and `checksums.txt`, and verify its SHA-256.
- Download both macOS archives freshly with `curl -fL`, verify their hashes, and extract them. curl does not normally apply quarantine: explicitly mark each downloaded archive and extracted binary with `xattr -w com.apple.quarantine "0083;$(printf '%x' "$(date +%s)");ordercli-release-verification;" <path>` and show it with `xattr -l`.
- Require `codesign -dvv <binary>` to show Peter Steinberger's Developer ID and `TeamIdentifier=Y5PE65HELJ`, then run `codesign --verify --deep --strict --verbose=4 <binary>` and `spctl -a -vv -t open --context context:primary-signature <binary>` (accepted, Notarized Developer ID).
- Run `./scripts/check-macos-target <binary>` and inspect `otool -l <binary>` for `LC_BUILD_VERSION` / `minos 13.0`. Run the quarantined native binary with `--version`; the shared verifier jobs cover execution on both architectures.
- Verify `go list -m github.com/steipete/ordercli@v<version>` through the Go proxy.
- Verify the Homebrew formula version and checksums, install or upgrade it, run `brew test steipete/tap/ordercli`, and check `ordercli --version`. See [the Homebrew playbook](releasing-homebrew.md).
- Review and merge the shared workflow's next empty `## Unreleased` PR (or open it if recovery requires it), then leave `main` clean and synchronized with `origin/main`.
