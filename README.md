# Mission Control

Mission Control surfaces what actually needs your attention across your developer tools. It watches your configured sources, evaluates a set of signals per item, and shows you only the things that are waiting on you — filtered by your dismiss and snooze decisions.

It is not a notification system. It does not push. You pull when you want a briefing, or you leave the server running and let it stay in sync in the background.

---

## How it works

Each configured source is synced on its own schedule. For every item fetched (a PR, an issue, etc.) the source evaluates a set of attention signals. Only items where at least one signal fires end up in your summary. Signals are specific: "you opened this PR and someone reviewed it", not "something happened". The active signals are shown alongside each item so you can see at a glance what the ask is.

Dismiss and snooze are local decisions. Nothing is written back to the source.

---

## Interfaces

There are three ways to use Mission Control. All three talk to the same server, read the same data, and share the same dismiss and snooze state — switching between them mid-session is seamless.

**CLI** — `devbrief` is a command-line tool for quick terminal briefings. Run `devbrief summary` to see what needs your attention. Supports dismiss, snooze, and per-source sync.

**Dashboard** — a web UI served by the `mission-control` server. Shows the same items with charts, filters, and KPI tiles. Useful when you want more context or want to work through a longer list.

**GNOME Shell extension** — a panel indicator that shows the attention count in the top bar and lets you open items directly, without switching windows.

---

## Quick start

**1. Write a config file at `~/.config/mission-control/config.yaml`**

```yaml
sources:
  - type: github
    name: work
    token: ghp_...
    user: your-github-login
    repos:
      - owner/repo
```

**2. Run the server**

```sh
git clone https://github.com/theakshaypant/mission-control
cd mission-control
make docker-build
make docker-run
```

The server syncs sources in the background and serves the dashboard at `http://localhost:5040`.

**3. Install the CLI**

```sh
go install github.com/theakshaypant/mission-control/cmd/devbrief@latest
devbrief summary
```

---

## Documentation

- [CLI reference](docs/cli.md)
- [Dashboard guide](docs/dashboard.md)
- [GNOME Shell extension setup](docs/gnome-extension.md)

**Sources**

- [GitHub](docs/sources/github.md)
