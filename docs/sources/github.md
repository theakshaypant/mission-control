# GitHub Source

Fetches open pull requests from one or more repositories and surfaces only the ones that need your attention right now. Everything else stays out of your way.

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

Use `all` if you maintain a repo and want to track the full queue, not just PRs that have your name on them.

### `max_prs`

Maximum number of PRs to fetch. Defaults to `50`.

With `pr_scope: all` this is a per-repo cap. With `pr_scope: involved` it is a cap across all repos combined.

### `waits_on_me`

List of signals that determine whether a PR needs your attention. A PR is only surfaced when at least one configured signal fires. Each signal represents a single, specific state — there is no hidden OR logic inside a signal.

Defaults to `[unreviewed, author_updated, peer_activity, review_received]`.

See [Attention signals](#attention-signals) for the full list.

### `stale_days`

Number of days without activity before a PR is considered stale. Only relevant when `stale` is included in `waits_on_me`. Defaults to `30`.

### `interactions`

What counts as you having interacted with a PR. Controls the `my activity` timestamp on each item — it does not affect which PRs are surfaced (that is driven entirely by `waits_on_me`).

| Value | Counts when you... |
|---|---|
| `review` | Submit any review, including comment-only reviews |
| `approve` | Approve a PR |
| `request_changes` | Request changes on a PR |
| `comment` | Leave a top-level comment (not part of a review) |

Defaults to all four when not set.

### `is_assigned`

What marks a PR as "assigned to you". This is metadata on the item — it does not affect which PRs are surfaced.

| Value | Marks as assigned when... |
|---|---|
| `author` | You opened the PR |
| `assignee` | You are in the PR's assignees list |
| `reviewer` | You have been explicitly requested as a reviewer |

Defaults to all three when not set.

---

## Attention signals

A PR is only surfaced when at least one signal in `waits_on_me` fires. The item's output includes a `why` field listing every signal that fired, so you can see at a glance what needs doing.

### `unreviewed`

You have never left a review or comment on this PR, and you did not open it.

This is the starting state for every PR in a repo you maintain. A PR drops out of this state the moment you engage with it.

### `author_updated`

You reviewed or commented, and since then the author has pushed new commits or replied. The ball is back in your court.

Only fires when you have already engaged. It will not surface PRs you have never looked at — that is what `unreviewed` is for.

### `peer_activity`

Someone who is neither the author nor you has reviewed or commented. If you have already engaged, this only fires when the peer activity is newer than your last review or comment — meaning it is something you have not seen yet.

Useful when you co-maintain a repo with others and want to stay aware of discussions even on PRs you are not the primary reviewer for.

### `review_received`

You opened this PR, and someone else has commented or reviewed since your last commit push or comment. Someone responded to your work.

### `approved`

You opened this PR and GitHub considers it fully approved — all required reviewers have approved and no one has an outstanding request for changes. Time to merge or act on it.

### `approved_not_merged`

GitHub considers this PR fully approved but it has not been merged yet. Fires regardless of who the author is, so it is useful for tracking PRs that are ready to land across your whole repo.

### `stale`

No one has touched this PR in more than `stale_days` days (default 30). Useful for periodic cleanup — finding PRs that have gone quiet and might need a nudge or a close.

---

## Example configurations

### OSS maintainer — track the full review queue

You want to see every PR that still needs your eyes, know when authors respond to your feedback, hear about discussions from other contributors, and be notified when your own PRs get reviewed.

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
    waits_on_me:
      - unreviewed
      - author_updated
      - peer_activity
      - review_received
```

### Contributor — only what is explicitly on your plate

You work across several repos but are not responsible for reviewing everything. You only want to act when you have been formally requested, and you want to know when your own PRs get feedback.

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
    waits_on_me:
      - unreviewed
      - author_updated
      - peer_activity
      - review_received
      - stale
    stale_days: 14
```
