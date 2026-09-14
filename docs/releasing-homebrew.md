# ordercli Homebrew Release Playbook

The shared release workflow updates `steipete/homebrew-tap` automatically after independently verified archives are published. The formula installs prebuilt binaries for macOS and Linux on Intel and ARM; starting with v0.2.1, macOS binaries are Developer ID signed and notarized.

## Automated handoff

`.github/workflows/release.yml` selects `homebrew-tap: steipete/homebrew-tap` and `homebrew-formula: ordercli`. The shared workflow dispatches `update-formula.yml` with the release tag, repository, and exact inventory-derived asset names and SHA-256 values. Names remain `ordercli_<version>_<target>.tar.gz`. `HOMEBREW_TAP_TOKEN` maps to `TAP_TOKEN` and needs Contents read plus Actions write on the tap. The handoff waits for the correlated run and verifies every resulting formula URL and checksum against the independently verified release.

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

If the release assets are complete but the handoff fails, fix the reported cause and rerun the failed jobs on the original release run. Do not rebuild or replace published signed assets.
