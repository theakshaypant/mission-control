# Mission Control dashboard

The dashboard is a web UI that gives you the same view as `devbrief summary` but lives in a browser tab and updates itself automatically. It's served by the same `mission-control` server that powers the CLI and the GNOME extension.

---

## Starting the server

**Docker (recommended):**

```sh
make docker-build
make docker-run
```

The dashboard is then at `http://localhost:5040`.

**From source:**

```sh
go run ./cmd/server --config ~/.config/mission-control/config.yaml
```

The server syncs your configured sources in the background on their configured intervals, so the dashboard stays fresh without you having to do anything.

---

## Layout

### Header

The top bar shows the timestamp of the last briefing and gives you the **Sync** button. Click it to open a dropdown where you can sync all sources at once or pick a specific source by name. The sync spinner replaces the icon while a sync is in progress.

The small toggle on the right (○ / ●) switches between dark and light mode. Your preference is saved in the browser.

### KPI tiles

Four numbers at a glance:

| Tile | What it shows |
|---|---|
| **Priority** | Items with at least one active signal — the ones that most need your attention |
| **Sources** | How many sources are configured |
| **Snoozed** | Items you've snoozed that aren't currently in the list |
| **Last Briefing** | When the most recent sync completed |

### Charts

Three interactive charts sit below the tiles. Clicking any bar or slice applies that value as a filter on the item list. Click it again to clear the filter.

- **Signal chart** — breaks down items by which attention signal fired (e.g. `unreviewed`, `review_received`)
- **Source donut** — shows the share of items per source
- **Namespace chart** — groups items by namespace (org, team, or project depending on the source)

### Sync status bar

A compact row showing the last sync time for each of your configured sources. Clicking a source name triggers a sync for just that source.

### Filter bar

Use this to narrow the item list without affecting the charts or KPI tiles.

- **Source** — pick a specific source by name, or leave it on "All"
- **Type** — toggle one or more item types (e.g. PR, issue). Multiple selections show items of any selected type
- **Sort** — order by last updated (default), source, or type

The **REFRESH** button reloads the item list from the server without triggering a full sync.

---

## Item list

Each card shows the item type, title, source, namespace, the signals that fired, and a link to open the item directly.

Two actions are available on every card:

### Dismiss

Permanently removes the item from your list. It won't reappear on future syncs, even if its signals fire again. Use this when you've dealt with something and don't want to see it again.

Dismissal is a local decision — nothing is written back to GitHub or wherever the item came from.

### Snooze

Hides the item until a time you choose. Once the snooze expires, the item reappears automatically on the next sync if it still has active signals.

Clicking **Snooze** on a card opens a modal with two modes:

**For duration** — snooze for a set amount of time from now. Quick picks: 1h, 4h, 24h, 7d. Or type any value in the input field (e.g. `2h30m`, `3d`).

**Until date** — pick a specific date using the date picker. The snooze expires at midnight local time on that date.

