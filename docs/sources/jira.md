# Jira Source

> **Not recommended for most users.** Jira tickets are project-management artifacts that evolve over days or weeks — they are not urgent by nature. In practice this source tends to flood the feed with noise rather than surface things that need immediate action. The GitHub source is a much better fit for mission-control's urgency model. The Jira source is kept for completeness, but personal experience showed it was more distracting than useful and was turned off within a week.
>
> If you do use it, keep your JQL narrow and limit `waits_on_me` to `comment_received` only.

---

Fetches Jira Cloud tickets matching one or more JQL queries. Each query is called a "board" and its name is used as the item namespace in the feed.

## Quick start

```yaml
sources:
  - type: jira
    name: my-jira
    host: mycompany.atlassian.net
    email: you@example.com
    token: top-secret-token
    boards:
      - name: In Progress
        jql: assignee = currentUser() AND status = "In Progress"
```

`host`, `email`, `token`, and at least one `board` are required. Everything else has a sensible default.

---

## Configuration reference

### `host`

The bare hostname of your Jira Cloud site, e.g. `mycompany.atlassian.net`. Do not include `https://` or a trailing slash.

### `email`

Your Atlassian account email address. Used for Basic authentication and signal evaluation (identifying you in assignee, reporter, and comment fields).

### `token`

An Atlassian API token. Generate one at [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens).

### `sync_interval`

How often to sync this source. Accepts Go duration strings: `30m`, `1h`, `2h30m`. Defaults to `3h`.

```yaml
sync_interval: 30m
```

### `boards`

A list of named JQL queries. Each board becomes a namespace in the feed.

```yaml
boards:
  - name: In Progress
    jql: assignee = currentUser() AND status = "In Progress"
    max_results: 50
```

| Field | Description | Default |
|---|---|---|
| `name` | Human-readable label, used as the item namespace | required |
| `jql` | Any valid JQL filter; `ORDER BY` clauses are supported | required |
| `max_results` | Cap on tickets fetched per sync for this board | `50` |

If a ticket matches multiple boards it appears once in the feed — the first board's name is used as the namespace and signals from all matching boards are unioned.

On incremental syncs (all syncs after the first), an `AND updated > "<last sync time>"` clause is automatically injected into each board's JQL before any `ORDER BY`, so only recently changed tickets are re-fetched.

### `waits_on_me`

Signals that mark a ticket as needing your attention. At least one must fire for a ticket to surface in the feed. Defaults to all four signals.

```yaml
waits_on_me:
  - assigned
  - comment_received
  - stale
  - status_changed
```

See [Attention signals](#attention-signals) for full details.

### `stale_days`

Number of days without activity before a ticket is considered stale. Only used by the `stale` signal. Defaults to `14`.

### `interactions`

What counts as you having acted on a ticket. Advances the `my activity` timestamp shown in the feed. Currently only `comment` is supported.

```yaml
interactions:
  - comment
```

Defaults to `[comment]` when not set.

### `done_statuses`

Workflow status names considered terminal. Tickets that transition into one of these statuses are tombstoned (removed from the feed) on the next incremental sync.

```yaml
done_statuses:
  - Done
  - Closed
  - Resolved
  - "Won't Do"
  - Cancelled
```

Defaults to `["Done", "Closed", "Resolved", "Won't Do"]` when not set. Comparison is case-insensitive.

### `api_version`

Jira REST API version to target. Only `3` is currently supported (Jira Cloud REST API v3). Can be omitted.

---

## Attention signals

An item surfaces in the feed only when at least one configured signal fires. Every fired signal is listed in the item's `active_signals` attribute so you can see at a glance what triggered it.

### `assigned`

The ticket is currently assigned to you (matched by email). Fires for every ticket in your boards where you are the assignee — which on most backlogs means it fires for everything, all the time. This is the primary source of noise when using this source.

### `comment_received`

Someone other than you has commented on the ticket more recently than your last comment. Only fires when you are the **assignee or reporter** of the ticket — it does not fire for tickets you are merely watching.

This is the most actionable signal: someone replied to your work or is waiting on you.

### `stale`

No one has touched the ticket in more than `stale_days` days (default 14). On an active backlog this is true for a large fraction of tickets and tends to add noise rather than urgency. Useful only if your boards are tightly scoped to actively worked items.

### `status_changed`

The ticket's status changed since the last sync. **Only fires on incremental syncs** — skipped on the first (full) sync to avoid surfacing every ticket in the store at startup.

In many teams this fires constantly due to CI bots and automation rules advancing statuses with no human intent behind the change.

---

## Item types

The Jira issue type field is mapped to a mission-control item type:

| Jira issue type | Item type |
|---|---|
| Bug | `bug` |
| Story | `story` |
| Task, Sub-task, Subtask | `task` |
| Epic | `epic` |
| Feature | `feature` |
| Anything else | `ticket` |

---

## Item attributes

Each Jira item carries the following source-specific attributes (available as JSON in `Item.Attributes`):

| Field | Type | Description |
|---|---|---|
| `issue_key` | string | Jira issue key, e.g. `PROJ-123` |
| `issue_type` | string | Raw Jira issue type name, e.g. `Bug`, `Story` |
| `status` | string | Current workflow status name, e.g. `In Review` |
| `priority` | string | Priority name, e.g. `High`. Omitted if unset. |
| `reporter` | string | Email address of the issue reporter. Omitted if unset. |
| `assignee` | string | Email address of the current assignee. Omitted if unset. |
| `labels` | []string | Labels attached to the issue. Omitted if empty. |
| `active_signals` | []string | Which `waits_on_me` signals fired for this item. |

---

## Example configurations

### Minimal — comment replies only

The least noisy configuration. Only surfaces tickets where someone replied to you.

```yaml
sources:
  - type: jira
    name: jira
    host: mycompany.atlassian.net
    email: you@example.com
    token: top-secret-token
    sync_interval: 30m
    boards:
      - name: In Progress
        jql: assignee = currentUser() AND status in ("In Progress", "Code Review")
        max_results: 25
    waits_on_me:
      - comment_received
    done_statuses:
      - Done
      - Closed
      - Resolved
      - "Won't Do"
      - Cancelled
```

### Multiple boards

```yaml
sources:
  - type: jira
    name: jira
    host: mycompany.atlassian.net
    email: you@example.com
    token: top-secret-token
    boards:
      - name: In Progress
        jql: assignee = currentUser() AND status = "In Progress"
      - name: On QA
        jql: assignee = currentUser() AND status = "On QA"
    waits_on_me:
      - comment_received
      - status_changed
    stale_days: 14
```

### Multiple Jira Cloud instances

Each source instance is fully independent — separate host, credentials, and signal configuration.

```yaml
sources:
  - type: jira
    name: work-jira
    host: mycompany.atlassian.net
    email: you@mycompany.com
    token: top-secret-token
    boards:
      - name: My Work
        jql: assignee = currentUser() AND status = "In Progress"
    waits_on_me:
      - comment_received

  - type: jira
    name: client-jira
    host: client.atlassian.net
    email: you@contractor.com
    token: another-token
    boards:
      - name: Client Tickets
        jql: assignee = currentUser() AND status != Done
    waits_on_me:
      - comment_received
      - status_changed
```
