---
name: giis-desktop
description: Use when the user wants to control the local GiiS desktop app running on the same machine (check status, install feature packs, etc.). Calls the desktop app's local HTTP control server over localhost.
---

# GiiS Desktop Control

Talks to a GiiS desktop app running on the same machine over a loopback-only HTTP server. This
lets you check the app's status and install feature packs without going through the GiiS cloud.

## Discovering the desktop app

The desktop app writes a bridge descriptor file when it starts. To check if the desktop app is
running locally:

1. Read the descriptor file from `~/.local/share/giis-desktop/giis-desktop-bridge.json` on Linux
   (or `~/Library/Application Support/giis-desktop/giis-desktop-bridge.json` on macOS). The
   exact path may vary if `XDG_DATA_HOME` is set to a non-standard location on Linux.
2. If the file exists, parse it as JSON to extract `port` (a number) and `token` (a string).
3. If the file does not exist, or if `GET http://127.0.0.1:<port>/health` fails to connect,
   the desktop app is not running locally right now. Tell the user "the desktop app is not
   available" rather than erroring hard.

Example descriptor file:

```json
{
  "port": 52847,
  "token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890-1a2b3c4d-5e6f-7890-abcd-ef1234567890-5f6g7h8i-9j0k-1l2m-3n4o-5p6q7r8s9t0u"
}
```

## Authentication

Use the `token` from the descriptor file in an `Authorization: Bearer <token>` header on every
request except `/health`. The token is persisted encrypted on disk and survives app restarts
(it only regenerates if that stored secret is ever deleted) — it's the **port** that changes on
every restart, which is exactly why you should always re-read the descriptor file each session
rather than caching either value.

## Checking status

```
GET http://127.0.0.1:<port>/status
Authorization: Bearer <token>
```

Example curl command:

```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:$PORT/status
```

Response:

```json
{
  "version": "0.4.0",
  "logged_in": null
}
```

**Important:** The `logged_in` field is currently always `null` and should not be relied on to
determine whether the user is actually logged in to the desktop app. Real login state tracking
will be added in a future phase.

## Health check (no auth)

```
GET http://127.0.0.1:<port>/health
```

Example curl command:

```bash
curl http://127.0.0.1:$PORT/health
```

Response:

```json
{
  "status": "ok"
}
```

Use this endpoint to check if the desktop app is reachable without needing the token.

## Installing a feature pack

```
POST http://127.0.0.1:<port>/install-pack
Authorization: Bearer <token>
Content-Type: application/json

{
  "pack_name": "core"
}
```

Example curl command:

```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"pack_name": "core"}' \
  http://127.0.0.1:$PORT/install-pack
```

Valid pack names (exactly these five):

- `"core"` — core services (runs `docker compose up -d`)
- `"flows"` — Flowise (runs `docker compose -f docker-compose.flowise.yml up -d`)
- `"social"` — social tools (runs Postiz and Kitsune together)
- `"training"` — Unsloth training (runs `docker compose -f docker-compose.unsloth.yml up -d`)
- `"chatdev"` — ChatDev (runs `docker compose -f docker-compose.chatdev.yml up -d`)

Response:

```json
{
  "success": true,
  "output": "container_id created\nService started",
  "error": null
}
```

Or on error:

```json
{
  "success": false,
  "output": null,
  "error": "Unknown install pack: invalid-name"
}
```

**Important:** Installation can take a while since `docker compose up -d` runs synchronously.
If `success` is `false`, either the pack name was invalid (check the exact five valid names
above) or Docker failed to start the services (check the error message for Docker-specific
issues).

## What this does not do

- There is no endpoint to stop or uninstall a pack yet. Use Docker commands directly to manage
  running containers.
- Desktop app data beyond version is not accessible (no way to read login state, installed
  packs, etc., beyond `GET /status`).
- There is no way to trigger any UI action on the desktop app over this API.
- The port changes every time the desktop app restarts, so the descriptor file must be re-read
  each session rather than cached in a config file or hardcoded.
