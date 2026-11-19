# Multi-user (Tier 1): several operator logins, shared resources

Status: implemented (migration `0099_users.sql`).

## Decision

opendray supports more than one operator login. Each login has a role:

| Role | Reaches |
|---|---|
| `admin` | Everything. |
| `member` | Sessions, providers and accounts, channels, memory, Cortex, round tables, canvas, and the workspace tools (files, git, git host, search, notes, custom tasks, project scan, version info). |

Members are refused (HTTP 403) on the admin-only group: user management,
integrations admin and call log, skills and MCP vaults (including MCP
secrets), vault git and sync, audit log, server settings and logs,
backups, self-update, and the memory pipeline configuration
(summarizers, workers, capture, injector, cleaner).

## This is not isolation

Roles gate the HTTP API only. Every session still runs as the single
`opendray` OS user, so any session's agent can read every project,
`secrets.env`, the MCP vault and every provider account directory.
Every login sees every project and every session. Give member logins
only to people you would trust with the admin login. Isolated users
(Tier 2) would need per-user OS identity for sessions, per-project
access rules on every route, and per-user secrets; a separate opendray
instance per person is the cheaper way to get isolation today.

## Shape

- The **primary admin** stays where it was: keyfile, config or
  `OPENDRAY_ADMIN_*` env (`internal/auth/keyfile.go`). It is always
  `admin`, is not a row in `users`, and remains the recovery path.
- **Additional logins** live in the `users` table: case-insensitive
  unique username, bcrypt hash, role, disabled flag, last login. A user
  cannot take the primary admin's username.
- **Tokens** carry `role` and `user_id` (empty for the primary admin).
  `auth.RequireAdmin` guards the nested admin group in `app.go`.
- **Revocation is immediate**: disabling, deleting, re-roling or
  resetting a user's password drops that user's tokens. Rotating the
  primary admin's credentials revokes only the primary admin's tokens.
- **Self-service**: `POST /auth/change-credentials` lets a users-table
  login change its own password (not its username). `GET /auth/me`
  returns `role` and `primary`.
- **Sessions** record `owner` (the creating login's username) from the
  authenticated principal, never the request body. Sessions created
  before this change have no owner and count as the primary admin's.
  `GET /sessions?owner=me` filters server-side; the web list shows a
  Mine/All toggle once more than one login owns sessions.

## API

`GET /users`, `POST /users`, `PATCH /users/{id}`, `DELETE /users/{id}`
(admin role). An admin cannot demote, disable or delete their own
users-table login.

## Not changed

Integration keys and the mobile app (members can sign in; its admin
screens get 403). Login success and failure events now name the login
that signed in instead of the configured admin user.
