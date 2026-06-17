# cascade

A terminal task tracker for nested software work. Vim-flavored, markdown-on-disk, themable. Built with [bubbletea](https://github.com/charmbracelet/bubbletea) and friends.

```
cascade › cascade-launch
  ..  (go up)                  │
  ▸ planning      [2/3]        │   Cascade Launch
  ▸ implementation [0/2]       │   ──────────────────────────────────
  ○ tag v1                     │   Release plan for v1.
                               │
                               │   Subtasks
                               │
                               │     ▸ planning      [2/3]
                               │     ▸ implementation [0/2]
                               │     ○ tag v1

l drill in · n new · e edit · / search · : actions · ? help · q quit
```

## Install

```sh
go install github.com/mwdomino/cascade/cmd/cascade@latest
# or, from a checkout:
just build && ./cascade
```

Requires Go ≥ 1.22 and a `$EDITOR` you actually like (cascade defers all body editing to it).

## Quick start

Launch with `cascade` — the first time it creates `~/.cascade/` and shows an empty pane with hints. Then:

| | |
|---|---|
| `n` | new item at the current tier |
| Type a title, `Enter` | confirm |
| `l` | drill into it (header changes from `cascade` to `cascade › <name>`) |
| `n` again | now creates a child of that item |
| `e` | open the selected item's `index.md` in `$EDITOR` for body editing |
| `x` | mark the selected task done (cycles status: todo → doing → done → blocked) |
| `h` | go up one tier (or hit `Enter` on the `..` row) |

The "where am I" cue is the breadcrumb. If it says `cascade`, `n` creates a project. If it says `cascade › Foo`, `n` creates a child of `Foo`.

## Concepts

Every node is a folder containing `index.md`. The TUI shows three node types:

- **Project** (`■`, accent purple) — a top-level container that has children.
- **Folder** (`▸`, dim) — a non-top-level container.
- **Task** (`○ ◐ ✓ ✗`) — a leaf, the only thing with a status.

Types are derived from position by default but can be pinned in frontmatter (`type: project | folder | task`). Top-level leaves are tasks until they sprout children, at which point they promote to projects automatically.

A container is "effectively done" when every descendant task is done. The sidebar glyph rolls up to `✓` (green) and the title gets dim + strikethrough. The `[done/total]` indicator next to each container counts effectively-done children, so the rollup glyph and the count always agree.

## On-disk layout

```
~/.cascade/
  010-cascade-launch/
    index.md                       # frontmatter + body
    010-planning/
      index.md
      010-scope-outline/index.md
      020-approval/index.md
    020-implementation/
      index.md
      …
  020-personal/
    index.md
  .trash/                          # soft-deleted nodes, recoverable
    20260618T100432-old-task/
```

- Sibling order = numeric filename prefix with gaps of 10 (`010-`, `020-`, `030-`). Inserting between two siblings uses `015-` so git diffs stay small.
- `index.md` has YAML frontmatter (title, status, type, created, updated, tags) plus any extra keys you want — they round-trip untouched and become `$CASCADE_FM_*` env vars for actions.
- Edit files outside cascade with any editor; press `R` inside cascade to reload from disk.

## Keybindings (default)

### Navigation
| Key | What |
|---|---|
| `j` / `k` / `↑` / `↓` | move cursor |
| `l` / `Enter` | drill into selected (or `..` to go up) |
| `h` | back one tier |
| `gg` / `G` | top / bottom |
| `R` | reload from disk |

### Capture & edit
| Key | What |
|---|---|
| `n` | new item at current tier |
| `gn` | quick-capture to the configured inbox |
| `r` | rename selected |
| `e` | open selected `index.md` in `$EDITOR` |

### Manipulation
| Key | What |
|---|---|
| `K` / `J` | move selected up / down (rewrites disk prefixes) |
| `m` | move to another parent (fuzzy picker) |
| `x` / space | cycle task status |
| `Z` | toggle hide-done (default: show with strikethrough) |
| `dd` | soft delete to `.trash/` |
| `D` | hard delete |

### Search & commands
| Key | What |
|---|---|
| `/` | filter current tier |
| `Ctrl-f` | global fuzzy search (title + tags + body) |
| `:` | command palette |
| `?` | help overlay |
| `q` / `Ctrl-c` | quit |

The hint bar at the bottom of the screen always shows what's relevant for the current mode.

## Configuration

cascade reads two yaml files and merges them, with project-local overrides winning per-key:

- `~/.config/cascade/config.yaml` (or `$XDG_CONFIG_HOME/cascade/config.yaml`) — global defaults
- `$PWD/.cascade.yaml` — per-project overrides

Example global config:

```yaml
tasks_dir: ~/.cascade           # default
inbox: 999-inbox                # gn target, relative to tasks_dir

theme: dracula                  # built-in name, OR an inline theme block

actions:
  create-github-issue:
    cmd: 'gh issue create -R "$CASCADE_FM_GITHUB_REPO" -t "$CASCADE_TITLE" -F -'
    stdin: body
    keybind: gi
    when:
      has_frontmatter: [github_repo]
```

## Themes

Dracula ships as the built-in default. You can name another built-in or define one inline:

```yaml
theme:
  palette:
    bg: "#282a36"
    fg: "#f8f8f2"
    dim: "#6272a4"
    border: "#44475a"
    accent: "#bd93f9"
  status: { todo: "#6272a4", doing: "#f1fa8c", done: "#50fa7b", blocked: "#ff5555" }
  selection: { cursor_bg: "#44475a", search_match: "#ffb86c" }
  markdown:
    heading_h1: "#bd93f9"
    heading_h2: "#ff79c6"
    heading_h3: "#8be9fd"
    heading_h4: "#50fa7b"
    heading_h5: "#f1fa8c"
    heading_h6: "#ffb86c"
    code: "#50fa7b"
    link: "#8be9fd"
    list: "#f8f8f2"
    checkbox_done: "#50fa7b"
    checkbox_todo: "#6272a4"
```

## Actions

Actions are shell commands invoked via the `:` palette or a bound key. cascade injects task context as env vars:

| Var | Value |
|---|---|
| `CASCADE_TITLE` | the node's title |
| `CASCADE_PATH` | absolute path to the node's folder |
| `CASCADE_STATUS` | current status (tasks only) |
| `CASCADE_TAGS` | space-separated tags |
| `CASCADE_BODY_FILE` | absolute path to `index.md` |
| `CASCADE_FM_<KEY>` | every frontmatter key (uppercased, non-alnum → `_`) |

`when.has_frontmatter` gates which actions are offered for the selected node — e.g. an action that needs `github_repo:` only shows up on nodes that declare one.

## Roadmap

Done since v1:
- `..` filesystem-style up navigation
- project / folder / task types with positional fallback and rollup
- which-key help overlay (`?`) and mode-aware hint bar
- per-level heading colors (H1–H6) and styled task checkboxes
- stable layout — overlays no longer push the view upward
- centered palette modal

Open ideas:
- fsnotify auto-reload on external edits
- scrollable details pane (long bodies currently clip at the bottom)
- proper `gg` / `gn` chord handling
- additional built-in themes (gruvbox, tokyonight, nord)
- sort within a tier by status
- async action execution (don't freeze the TUI while `gh` runs)
- configurable keybindings via yaml
- inline checkbox toggling for `- [ ]` items in the body

## License

[fill in]
