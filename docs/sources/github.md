# GitHub Source

Fetches open pull requests and issues from one or more repositories and surfaces only the ones that need your attention right now. Everything else stays out of your way.

## Quick start

```yaml
sources:
  - type: github
    name: my-github
    token: ghp_...
    user: your-github-login
    repos:
      - owner/repo
```

`token`, `user`, and `repos` are the only required fields. Everything else has a sensible default.

By default only pull requests are fetched. To enable issues, set `issue_scope`.

---

## Configuration reference

### `host`

The GitHub hostname. Omit this field for github.com.

For GitHub Enterprise Server, set this to your instance hostname:

```yaml
host: github.mycompany.com
```

The GraphQL endpoint is derived automatically — `api.github.com/graphql` for github.com, `<host>/api/graphql` for everything else. Do not include the protocol or a trailing slash.

### `token`

A GitHub personal access token.

- Public repositories: `public_repo` scope
- Private repositories: `repo` scope
- GitHub Enterprise: same scopes apply; generate the token from your GHE instance

### `user`

Your GitHub login (e.g. `octocat`). Used to tell your activity apart from everyone else's when evaluating signals.

### `repos`

Repositories to watch, each in `owner/repo` format. You can define multiple source instances with different repo lists and different signal configurations.

### `pr_scope`

Controls which PRs are fetched.

| Value | Behaviour | `max_prs` applies to |
|---|---|---|
| `involved` *(default)* | Only PRs where you are author, reviewer, or assignee | Total across all repos |
| `all` | Every open PR in each configured repo | Per repo |
| `none` | Disable PR syncing entirely | — |

Use `all` if you maintain a repo and want to track the full queue, not just PRs that have your name on them. Use `none` alongside `issue_scope` if you only care about issues — for example, an aspiring contributor looking for issues to work on.

### `max_prs`

Maximum number of PRs to fetch. Defaults to `50`.

With `pr_scope: all` this is a per-repo cap. With `pr_scope: involved` it is a cap across all repos combined.

### `issue_scope`

Controls which issues are fetched. **Empty by default — issues are not synced unless this is set.**

> **Note:** Issue syncing is significantly more expensive than PR syncing. Each issue requires fetching its full comment history to evaluate signals, and large repos can have thousands of open issues. This runs synchronously and blocks other sources from syncing until it completes. Keep `issue_scope: involved` and a tight `issue_updated_within_days` (default 7) unless you have a specific reason to widen the scope.

| Value | Behaviour | `max_issues` applies to |
|---|---|---|
| `involved` | Only issues where you are author, assignee, or mentioned | Total across all repos |
| `all` | Every open issue in each configured repo | Per repo |

### `max_issues`

Maximum number of issues to fetch. Defaults to `50` when `issue_scope` is set.

With `issue_scope: all` this is a per-repo cap. With `issue_scope: involved` it is a cap across all repos combined.

### `issue_updated_within_days`

Only fetch issues updated within this many days. Defaults to `7`. This keeps the query scope tight on repos with many open issues and avoids fetching comment history for issues that have been quiet for a week or more.

Set to `0` to disable the filter and fetch all open issues regardless of age.

### `issue_comment_limit`

Maximum number of recent comments fetched per issue. Defaults to `10`. Must be between `1` and `100`.

Lowering this reduces query complexity on busy issues — useful when repos have issues with hundreds of comments and you are hitting rate limits. The trade-off is that signal evaluation only sees the most recent N comments: if your last comment on an issue falls outside that window, signals like `author_updated` may fire as though you never engaged.

### `waits_on_me`

List of signals that determine whether a PR or issue needs your attention. An item is only surfaced when at least one configured signal fires. Each signal represents a single, specific state — there is no hidden OR logic inside a signal.

The same signals apply to both PRs and issues. PR-only signals (`approved`, `approved_not_merged`) are silently ignored for issues.

Defaults to `[unreviewed, author_updated, peer_activity, review_received]`.

See [Attention signals](#attention-signals) for the full list.

### `stale_days`

Number of days without activity before a PR or issue is considered stale. Only relevant when `stale` is included in `waits_on_me`. Defaults to `30`.

### `interactions`

What counts as you having interacted with a PR. Controls the `my activity` timestamp on each PR item — it does not affect which items are surfaced (that is driven entirely by `waits_on_me`).

| Value | Counts when you... |
|---|---|
| `review` | Submit any review, including comment-only reviews |
| `approve` | Approve a PR |
| `request_changes` | Request changes on a PR |
| `comment` | Leave a top-level comment (not part of a review) |

Defaults to all four when not set. For issues, comments are always tracked regardless of this setting (review-type interactions don't exist on issues).

### `is_assigned`

What marks an item as "assigned to you". This is metadata on the item — it does not affect which items are surfaced.

| Value | Marks as assigned when... | Applies to |
|---|---|---|
| `author` | You opened the PR or issue | PRs and issues |
| `assignee` | You are in the assignees list | PRs and issues |
| `reviewer` | You have been explicitly requested as a reviewer | PRs only |

Defaults to all three when not set. For issues, `reviewer` is silently ignored.

---

## Attention signals

An item is only surfaced when at least one signal in `waits_on_me` fires. The item's output includes a `why` field listing every signal that fired, so you can see at a glance what needs doing.

Most signals apply to both PRs and issues. Where the behaviour differs, both are described.

### `unreviewed`

**PRs**: You have never left a review or comment on this PR, and you did not open it.

**Issues**: You have never commented on this issue, and you did not open it.

This is the starting state for every item in a repo you maintain. It drops out of this state the moment you engage with it.

### `author_updated`

**PRs**: You reviewed or commented, and since then the author has pushed new commits or replied.

**Issues**: You commented, and since then the author has replied.

Only fires when you have already engaged. It will not surface items you have never looked at — that is what `unreviewed` is for.

### `peer_activity`

Someone who is neither the author nor you has reviewed or commented. If you have already engaged, this only fires when the peer activity is newer than your last review or comment — meaning it is something you have not seen yet.

Useful when you co-maintain a repo with others and want to stay aware of discussions even on items you are not the primary reviewer for. Applies identically to PRs and issues.

### `review_received`

**PRs**: You opened this PR, and someone else has commented or reviewed since your last commit push or comment.

**Issues**: You opened this issue, and someone else has commented since your last comment.

Someone responded to your work.

### `approved`

You opened this PR and GitHub considers it fully approved — all required reviewers have approved and no one has an outstanding request for changes. Time to merge or act on it.

*PR only. Not applicable to issues.*

### `approved_not_merged`

GitHub considers this PR fully approved but it has not been merged yet. Fires regardless of who the author is, so it is useful for tracking PRs that are ready to land across your whole repo.

*PR only. Not applicable to issues.*

### `stale`

No one has touched this item in more than `stale_days` days (default 30). Useful for periodic cleanup — finding PRs and issues that have gone quiet and might need a nudge or a close.

Applies to both PRs and issues.

---

## Example configurations

### OSS maintainer — track the full queue including issues

```yaml
sources:
  - type: github
    name: my-oss-project
    token: ghp_...
    user: your-login
    repos:
      - your-org/your-repo
    pr_scope: all
    max_prs: 100
    issue_scope: all
    max_issues: 100
    waits_on_me:
      - unreviewed
      - author_updated
      - peer_activity
      - review_received
```

### Contributor — only what is explicitly on your plate

You work across several repos but are not responsible for reviewing everything. You only want to act when you have been formally requested, and you want to know when your own PRs and issues get feedback.

```yaml
sources:
  - type: github
    name: work
    token: ghp_...
    user: your-login
    repos:
      - org/backend
      - org/frontend
    pr_scope: involved
    issue_scope: involved
    waits_on_me:
      - unreviewed
      - author_updated
      - review_received
```

### Aspiring contributor — issues only, no PRs

You are exploring repos to contribute to and want to track open issues you are involved in or have commented on, without the noise of PR reviews.

```yaml
sources:
  - type: github
    name: oss-contribution
    token: ghp_...
    user: your-login
    repos:
      - org/project
    pr_scope: none
    issue_scope: involved
    waits_on_me:
      - unreviewed
      - author_updated
      - review_received
```

### GitHub Enterprise Server

Point `host` at your GHE instance hostname. Everything else works identically.

```yaml
sources:
  - type: github
    name: ghe
    host: github.mycompany.com
    token: ghp_...
    user: your-login
    repos:
      - org/repo
    pr_scope: all
    issue_scope: involved
    waits_on_me:
      - unreviewed
      - author_updated
      - review_received
```

### Multiple accounts or roles

Each source instance is fully independent — separate token, repos, and signal configuration.

```yaml
sources:
  - type: github
    name: work
    token: ghp_work_token
    user: your-work-login
    repos:
      - work-org/backend
      - work-org/frontend
    pr_scope: involved
    issue_scope: involved
    waits_on_me:
      - unreviewed
      - author_updated
      - review_received

  - type: github
    name: personal-oss
    token: ghp_personal_token
    user: your-personal-login
    repos:
      - your-login/your-project
    pr_scope: all
    max_prs: 100
    issue_scope: all
    max_issues: 100
    waits_on_me:
      - unreviewed
      - author_updated
      - peer_activity
      - review_received
      - stale
    stale_days: 14
```
