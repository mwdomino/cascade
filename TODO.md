# TODO

Things we know we want, grouped by rough effort.

## Daily-use leverage

- [ ] **fsnotify auto-reload.** Watch `tasks_dir` for external changes (`:w` from another nvim, `git pull`) so cascade picks them up without `R`. Already a slot in the design; just needs wiring with debounce.
- [ ] **Configurable keybindings via yaml.** `keybindings:` block in `config.yaml` that merges over `keys.Default()`. Validate at load time (see action-collision below).
- [ ] **Undo (`u`).** Snapshot store mutations so soft delete / rename / move / status-cycle / checkbox-toggle can be reversed. Needs a small in-memory undo stack — N most recent ops, no persistence.
- [ ] **Bulk select / operate.** Visual-mode (`v`) then `j`/`k` extends a selection; subsequent action (`x`, `dd`, `m`) applies to all selected.

## Polish & robustness

- [ ] **Multi-key chord support for action keybinds.** Currently action `keybind:` must be a single key (or single key + modifier). General chord parsing would let users bind e.g. `gi` to a GitHub-issue action without conflicting with built-in chords.
- [ ] **Action keybind collision warning.** Validate at config load that no user action keybind shadows a built-in binding; emit a startup warning (and refuse to install the colliding bind).
- [ ] **Insert-between prefixes.** v0.1.x always appends (next gap-of-10) on Create and swaps on K/J. Wire `store.PrefixBetween` + a dedicated "insert here" action so the gap-of-10 spacing is actually exploited.
- [ ] **`nextPrefix` overflow past 99 siblings.** `%03d` gives `1000-foo` which sorts before `990-foo` in `ls`; switch to `%04d` or run a compaction pass at threshold.
- [ ] **`WriteIndex` defensive keys.** A poisoned `fm.Extra` map can currently clobber canonical fields (`status`, `type`, ...) on the next save. Skip reserved keys when iterating `Extra`.
- [ ] **`store.Rename` collision pre-check.** If `010-foo` is renamed to a slug another sibling already owns, `os.Rename` either fails mid-way or silently clobbers depending on OS; pre-check siblings.
- [ ] **Editor precedence.** Conventional POSIX order is `VISUAL → EDITOR → vi`; we currently do `EDITOR → VISUAL → vi`. Small swap.
- [ ] **`MoveTo` / `HardDelete` unit tests.** They have flow coverage via app tests but no direct store-layer tests like `MoveUp`/`SoftDelete`.

## UX features

- [ ] **Sort by status: toggle, not default.** Right now status-band sort is always on; some users may want strict manual prefix order. Add a `:sort manual` / `:sort status` palette action and a config default.
- [ ] **More built-in themes.** We have dracula / gruvbox / tokyonight / nord. Solarized (light + dark), catppuccin, rose-pine, everforest are easy adds.
- [ ] **Filter by tag.** `T` opens a tag picker showing every distinct tag with a count; selecting one filters the current tier (or scope: all-tasks) by that tag.
- [ ] **Tag picker integration with global search.** `Ctrl-f` query like `@p0` filters by tag rather than fuzzy-matching the body text.
- [ ] **Move-up promotes to top level.** Currently `m` lets you re-parent under any node OR the synthetic root via `(top level)`. Worth a dedicated keybinding (`gh`?) for the common "promote to top" case.
- [ ] **Show node count in the breadcrumb.** Something like `cascade › work › ship-v1 [3/7]` would let the user see overall progress without drilling into details.
- [ ] **Auto-archive done containers older than N days.** Configurable; moves them to `.archive/<year>-<month>/`. Keeps the tree shallow over time without losing history.

## Dead code / cleanup

- [ ] **Drop `editor.Open` if unused.** App now invokes the editor via `externalEditorCmd` in `app.go`; the helper in `internal/editor/editor.go` is dead.
- [ ] **Wire `store.PrefixBetween` / `RenumberGapOfTen` or drop them.** They're tested but never called from production. Natural users would be a sibling-renumber action or insert-between when reordering.
- [ ] **Remove the unused `builtinNames` function.** Replaced by exported `BuiltinNames`; the lowercase one is orphaned.

## Documentation

- [ ] **Animated demo / GIF in the README.** A short capture of `n → l → n → e` would convey the flow better than the static layout sketch.
- [ ] **Theme gallery screenshots.** One per built-in theme so users can pick visually.
- [ ] **Recipes for actions.** Worked examples for the GitHub issue, Jira ticket, Slack-to-task patterns.
- [ ] **License.** Pick one (MIT? Apache?) and fill it in.
