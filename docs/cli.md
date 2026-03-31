# devbrief CLI

`devbrief` is a command-line tool that surfaces what actually needs your attention across your developer tools. It evaluates a set of configurable signals per source, applies your local dismiss and snooze decisions, and shows you a focused list of items that are waiting on you.

---

## Installation

```sh
go install github.com/theakshaypant/mission-control/cmd/devbrief@latest
```

Or build from source:

```sh
git clone https://github.com/theakshaypant/mission-control
cd mission-control
go build -o devbrief ./cmd/devbrief
```

---

## Configuration

devbrief reads its config from `~/.config/mission-control/config.yaml` by default. Pass `--config <path>` to use a different file.

A minimal configuration with one source:

```yaml
sources:
  - type: github
    name: my-github
    token: ghp_...
    user: your-github-login
    repos:
      - owner/repo
```

For source-specific configuration options, see the relevant source documentation:

- [GitHub](sources/github.md)

### Multiple sources

Each entry in `sources` is an independent source instance with its own configuration. You can define as many as you need — across different accounts, organizations, or tools:

```yaml
sources:
  - type: github
    name: work
    token: ghp_work_token
    user: your-work-login
    repos:
      - work-org/backend
      - work-org/frontend

  - type: github
    name: personal
    token: ghp_personal_token
    user: your-personal-login
    repos:
      - your-login/your-project
```

### Sync interval

Each source syncs on its own interval. The default is 3 hours. Override per source:

```yaml
sources:
  - type: github
    name: work
    sync_interval: 30m
    ...
```

Valid values are Go duration strings: `30m`, `1h`, `2h30m`, etc.

---

## Commands

### `devbrief summary`

Show items that currently need your attention.

```
devbrief summary [--fresh] [--json]
```

Items appear in the summary when:
- At least one attention signal fired during the last sync
- The item has not been dismissed
- The item is not snoozed (or its snooze has expired)

The **WHY** column shows which signals fired for each item. See your source documentation for a description of each signal.

| Flag | Description |
|---|---|
| `--fresh` | Sync all sources before showing results |
| `--json` | Output as a JSON array instead of a table |

**Example output:**

```
ID                      TYPE   TITLE                          UPDATED           WHY                URL
github:owner/repo#42    pr     Fix: handle nil pointer in...  2026-03-30 14:22  review_received    https://...
github:owner/repo#38    pr     Add retry logic to client      2026-03-29 09:10  unreviewed         https://...
```

---

### `devbrief dismiss <id>`

Permanently suppress an item from your summary.

```
devbrief dismiss <id>
```

Dismissed items never appear in `devbrief summary` again, regardless of whether their signals continue to fire in future syncs. Dismissal is a local decision that survives syncs.

Use `snooze` instead if you want the item to reappear after a set time.

The item ID is shown in the **ID** column of `devbrief summary`.

**Example:**

```sh
devbrief dismiss github:owner/repo#42
```

---

### `devbrief snooze <id>`

Hide an item from your summary until a specified time.

```
devbrief snooze <id> --for <duration>
devbrief snooze <id> --until <time>
```

Once the snooze expires the item reappears automatically on the next `devbrief summary` run, provided its attention signals still fire.

You must provide exactly one of `--for` or `--until`.

| Flag | Description |
|---|---|
| `--for <duration>` | Snooze for a duration from now. Supports Go durations (`1h30m`, `24h`) and days (`2d`, `7d`). |
| `--until <time>` | Snooze until an absolute point in time (local timezone). |

`--until` accepted formats:

| Format | Example | Meaning |
|---|---|---|
| `HH:MM` | `14:30` | Today at that time |
| `YYYY-MM-DD` | `2026-04-07` | Start of that day |
| RFC3339 | `2026-04-07T09:00:00Z` | Exact timestamp |

**Examples:**

```sh
devbrief snooze github:owner/repo#42 --for 24h
devbrief snooze github:owner/repo#42 --for 2d
devbrief snooze github:owner/repo#42 --until 14:30
devbrief snooze github:owner/repo#42 --until 2026-04-07
```

---

### `devbrief sync [source]`

Fetch the latest items from configured sources and update local state.

```
devbrief sync
devbrief sync <source-name>
```

Without arguments, all configured sources are synced in sequence. Pass a source name to sync only that source.

Source names come from the `name` field in your config file. If a source has no name configured, its type is used.

**Examples:**

```sh
devbrief sync
devbrief sync work
devbrief sync personal
```

> If the background server is running, sources sync automatically on their configured interval. Use this command to force an immediate refresh or to sync a specific source during debugging.

---

## How it works

### Syncing

`devbrief sync` fetches items from each configured source and stores them locally. Each source tracks a sync cursor so incremental syncs only fetch recently updated items.

### Attention signals

For each fetched item, the source evaluates a set of signals to determine whether it needs your attention. Only items where at least one signal fires are surfaced in the summary. The set of active signals is shown in the **WHY** column, so you can see at a glance what needs doing.

Signals are configured per source. See the relevant source documentation for the full list.

### Dismiss and snooze

Dismiss and snooze decisions are stored locally. They are not written back to the source. A dismissed item remains dismissed even after re-syncing; a snoozed item reappears once its deadline passes.

---

## Global flags

| Flag | Description |
|---|---|
| `--config <path>` | Path to config file. Default: `~/.config/mission-control/config.yaml` |
