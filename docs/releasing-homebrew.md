# ordercli Homebrew Release Playbook

The release workflow updates `steipete/homebrew-tap` automatically after GoReleaser publishes the archives. The formula installs prebuilt binaries for macOS and Linux on Intel and ARM; it does not use the source archive.

## Automated handoff

`.github/workflows/release.yml` dispatches `update-formula.yml` in the tap with the release tag, repository, and artifact template `{formula}_{version}_{target}.tar.gz`. `HOMEBREW_TAP_TOKEN` must have workflow access to the tap. The release job waits for the exact dispatched run and fails if the update fails.

## Verify the formula

Read `Formula/ordercli.rb` from the tap and confirm the version, archive URLs, and all four SHA-256 values against the release's `checksums.txt`.

Refresh the local tap, then install or upgrade without removing an existing tap:

```sh
brew tap steipete/tap
git -C "$(brew --repo steipete/tap)" pull --ff-only
brew install steipete/tap/ordercli # use brew upgrade if already installed
brew test steipete/tap/ordercli
ordercli --version
ordercli --help
```

If the release assets are complete but the handoff fails, fix the reported cause and rerun only the failed Homebrew job. The legacy `scripts/release-homebrew.sh` prints source archive fields and is not used for the binary formula.
