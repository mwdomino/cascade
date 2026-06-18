# CLAUDE.md

Operating notes for Claude Code sessions in this repo. Keep brief; update when something surprises a future session.

## Workspace

- `/home/matt/dev/cascade` — main worktree, branch `main`.
- `/home/matt/dev/cascade__worktrees/cascade-plan` — feature worktree, branch `cascade-plan`.
- Push cadence: branch → fast-forward main → push main. Both refs stay aligned.
- Remote: `git@github.com:mwdomino/cascade.git` (SSH).

## Workflow rules (user-set)

- **Never tag a release on your own.** Push commits and let CI run, but only `git tag vX.Y.Z` when the user explicitly says "release" or "tag".
- Prefer terse responses; the user reads diffs and tool output directly.
- One commit per logical fix when working through a batched review. Aggregate doc-only commits.
- Use `just`, not `make`. `just build`, `just test`, `just run`, `just clean`.

## Architecture (load-bearing constraints)

- `internal/store` is the only package that touches the filesystem. Other packages consume `model.*`.
- Every node on disk = folder + `index.md`. No exceptions, including the synthetic root.
- Sibling order = numeric filename prefix, gap-of-10 (`010-`, `020-`, ...). True insert-between (`015-`) is NOT implemented yet (`store.PrefixBetween` exists but is unused) — `Create` appends, `K`/`J` swap.
- `Tree.byPath` is the canonical lookup. ANY method that changes a node's `Path` must update `byPath` and recurse via `rebuildByPathSubtree` (for renames/moves) or `purgeByPath` (for deletes — descendants must be removed too).
- `visibleSiblings()` applies, in order: GlobalMode override → LocalQuery filter → hide-done (when `!ShowDone`) → status-band stable sort.
- Cursor math goes through `cursorAtChild` / `childIndex` / `cursorMax` — ALWAYS via `visibleSiblings`, never raw `Current.Children`. Mixing those scopes is the #1 regression source.

## Type system (project / folder / task)

- `Frontmatter.Type` is optional. Positional default: top-level + children → project; non-top with children → folder; leaf at any depth → task.
- A top-level leaf is a TASK (so `x` can mark it done). It promotes to project automatically when it grows children.
- `Node.EffectivelyDone()` is "all descendant tasks done"; drives the rollup glyph (`✓`), the dim+strike, the progress count, and the `Z` filter. `ProgressDoneTotal` uses this — not raw `FM.Status`.
- `x`/space on a container is a silent no-op. Rollup is the only feedback. Don't reintroduce the "status only applies to tasks" error.

## Theme rendering — gotchas

- `GlamourStyle()` overlays user-themable slots on top of `styles.DraculaStyleConfig` from glamour. DO NOT build a fresh `ansi.StyleConfig` from scratch — every slot we don't populate (Emph, Strong, Strikethrough, List, ...) loses styling.
- **NEVER set `cfg.Text.Color`.** glamour's cascade rule gives a non-nil child color priority over parent. Setting Text.Color globally kills all heading colors because heading-text nodes inherit the override. `Document.Color` is enough as the unstyled-paragraph baseline.
- Heading per-level colors require both: setting the H1/H2/... `Color` field AND clearing the literal `Prefix` (`"# "`, `"## "`, ...) glamour ships with.
- Strikethrough requires DOUBLE tildes (`~~text~~`). Single tildes aren't strikethrough in any markdown parser.
- Glamour renderer is cached on `details.Model` keyed by width. `details.ClearCache()` invalidates it — call this whenever the theme swaps (already wired in palette's theme:* and theme-preview paths).

## Palette + theme preview

- The palette and move-picker render as floating overlays via `overlay(base, top, x, y)` (`ansi.Cut` line splice). Both cards have a solid background via `cardStyle()` so the panes underneath stay visible at the edges only.
- Theme preview on hover: `SavedTheme` is captured at palette-open, `*m.Theme` is mutated in place per hover, `revertThemePreviewIfActive()` rolls back on Esc / non-theme commit.
- Action keybinds are SINGLE KEY ONLY (or single key + modifier). Multi-char chord sequences for actions are NOT implemented — only the built-in `gg`, `gn`, `dd` chords. README/docs say this; don't promise more without writing the chord buffer.

## Key chords

- `gg`, `gn`, `dd` are real 500ms chords via `PendingG`/`PendingDD`. Single `g` is a no-op (primes the chord); cancelled by any non-chord key.
- The `Top` / `QuickNew` bindings in `keys.Map` are intentionally empty — they're dispatched by the chord handler in `app.Update`. Don't restore key strings to them.

## CI / release

- `.github/workflows/ci.yml`: vet + build + race tests on push to main and on PRs.
- `.github/workflows/release.yml`: on `v*` tag, run tests then `goreleaser release --clean`.
- `.goreleaser.yaml`: 5 binaries (linux amd64/arm64, macos amd64/arm64, windows amd64), `-s -w -X main.version={{.Version}}`.
- Tags MUST be valid semver (`vMAJOR.MINOR.PATCH`) — goreleaser v2 rejects `v0.1` style.
- `go.mod` declares `go 1.26.2`. Both CI workflows pin to `go-version-file: go.mod`.

## Useful spots

- `docs/superpowers/plans/2026-06-17-cascade-v1.md` — original implementation plan (frozen v1 design).
- `TODO.md` — open backlog grouped by effort.
- `README.md` — install (prebuilt / `go install` / source), keybinds, palette commands, config + theme schema.
- `internal/theme/render_smoke_test.go` — guards against the heading-color regression we already hit twice.
- `internal/store/store_test.go::TestMoveUpWithDuplicateSlugs` — guards the temp-path swap (duplicate-slug reorder).
- `internal/tui/app/app_test.go::TestCursorTracksNodeThroughSort` — guards the sort/filter cursor desync class.
