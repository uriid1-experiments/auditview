# auditview

Local web UI for browsing auditd logs. A single binary: the Go backend
parses `/var/log/audit/audit.log` (via elastic/go-libaudit — multi-record
event coalescing, uid/syscall resolution), the Svelte frontend is embedded
through `embed.FS`.

Russian documentation: [README.ru.md](README.ru.md). The UI language
(English by default, Russian available) is switched in the ⚙ settings menu.

![Screenshot](screenshot.png)

## Features

- Live log tail (SSE) with pause
- Filters: rule key (`-k`), exe, result, free-text search
- Per-key chips with event counts
- Event drill-down: process, user, syscall, touched files, raw records
- Reads rotated `audit.log.1..N` on start
- Rules editor for `/etc/audit/rules.d` with a form-based builder, expert
  mode, syntax validation and one-click `augenrules --load`
- Alerting on live events: desktop notifications, Telegram, custom command

## Build

```sh
cd web && npm install && npm run build && cd ..
go build -o auditview ./cmd/auditview
```

## Run

Reading `/var/log/audit` requires root:

```sh
sudo ./auditview
# → http://127.0.0.1:8654
```

Flags: `-log` (path to audit.log), `-addr` (default `127.0.0.1:8654` —
do not expose it: the UI has no authentication), `-rotated`, `-keyless`
(also keep events without a rule key — BPF, systemd service events and
the like; dropped on ingest by default), `-rules-dir` (auditd rules.d
directory, empty disables the editor), `-db` (SQLite, default
`/var/lib/auditview/auditview.db`), `-retain` (days of history, default
90), `-alerts` (alerting config, default `/etc/auditview/alerts.json`).

## Storage

Events are written to SQLite (pure-Go driver, no cgo): history survives
restarts, and the startup backfill from audit.log is deduplicated by
(ts, seq). Events older than `-retain` days are purged once a day.

## Alerting

The config `/etc/auditview/alerts.json` is created on first run: rules
keyed on audit keys (cred_*, persist_* enabled, shell_history disabled)
with a cooldown per rule+exe pair. Channels:

- `desktop` — notify-send into the user's session (the user is taken
  from SUDO_USER when the config is created)
- `telegram` — token + chat_id
- `command` — your own script; details arrive in env vars `AV_TITLE`,
  `AV_BODY`, `AV_RULE`, `AV_KEY`, `AV_EXE`, `AV_OBJECT`, `AV_USER`, `AV_HOST`

Alerts fire only on live events: the historical backfill on startup never
alerts. The "Alerts" tab is the log of firings with delivery status and a
test button.

## Autostart (systemd)

```sh
sudo cp auditview /usr/local/bin/
sudo cp contrib/auditview.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now auditview
```

After rebuilding, replace the binary atomically (a plain `cp` over the
running file fails with "Text file busy"):

```sh
sudo cp auditview /usr/local/bin/auditview.new \
  && sudo mv /usr/local/bin/auditview.new /usr/local/bin/auditview \
  && sudo systemctl restart auditview
```

## Development

```sh
sudo ./auditview &          # API on :8654
cd web && npm run dev       # Vite dev server proxying /api
```

To try the UI without root, point every path at a scratch location:

```sh
./auditview -log /tmp/av/audit.log -db /tmp/av/av.db \
  -alerts /tmp/av/alerts.json -rules-dir /tmp/av/rules.d
```
