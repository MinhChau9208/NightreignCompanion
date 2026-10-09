# Releasing

Releases are built by [.github/workflows/release.yml](../.github/workflows/release.yml) when a version tag is pushed.

1. Merge everything for the release into `main` and wait for CI to pass.
2. Make sure `info.productVersion` in `wails.json` is the version you are releasing (the workflow refuses a tag that does not match).
3. Smoke-test a local build: `wails build`, then run `build/bin/nightreign-companion.exe` (dashboard, network page, FPS with the game running, overlay).
4. Tag and push:
   ```sh
   git switch main && git pull
   git tag -a v0.1.0 -m "Nightreign Companion v0.1.0"
   git push origin v0.1.0
   ```
5. Watch the **Release** workflow in the Actions tab. It publishes a GitHub Release with
   `NightreignCompanion-vX.Y.Z-windows-amd64.zip` and its `.sha256`. `v0.*` tags are marked as pre-releases.
6. Download the zip from the release page and run it once to confirm it starts.

If the workflow fails, fix it on a branch, merge, then move the tag:

```sh
git tag -d v0.1.0 && git push origin :refs/tags/v0.1.0   # delete local + remote tag
# (delete the draft/failed release on GitHub if one was created)
git tag -a v0.1.0 -m "Nightreign Companion v0.1.0" && git push origin v0.1.0
```

For the next version, bump `info.productVersion` in `wails.json` (and the `version` default in `main.go`) in the same PR as the changes.
