// Scope metadata for the integration register / edit flows — the Dart
// mirror of app/shared/src/lib/scopes.ts. Kept in lockstep with the web
// ScopePicker so an operator sees the same human-readable labels on both
// surfaces. The titles/descriptions are intentionally English literals
// (the web source is too — these strings are not i18n'd on either side).
//
// When the gateway gains a new scope: add it to ALL_SCOPES (web types.ts)
// AND here. A scope already on an integration but missing from this list
// is preserved on save (it just renders without prose) — see ScopePicker.

enum ScopeGroup { sessions, channels, events, misc }

class ScopeInfo {
  const ScopeInfo(this.id, this.title, this.description, this.group);
  final String id;
  final String title;
  final String description;
  final ScopeGroup group;
}

// Order mirrors ALL_SCOPES in app/shared/src/lib/types.ts.
const List<ScopeInfo> kScopeInfo = [
  ScopeInfo(
    'session:read',
    'Read sessions',
    'List sessions, view session metadata, read terminal buffer, fetch project history.',
    ScopeGroup.sessions,
  ),
  ScopeInfo(
    'session:create',
    'Create sessions',
    'Spawn new sessions, restart ended ones, delete sessions when done.',
    ScopeGroup.sessions,
  ),
  ScopeInfo(
    'session:input',
    'Send input',
    "Forward keystrokes / commands into a session's PTY. Required to drive an agent CLI.",
    ScopeGroup.sessions,
  ),
  ScopeInfo(
    'channel:send',
    'Send to channels',
    'Push notifications and messages out through a registered channel (Telegram, Slack, etc.).',
    ScopeGroup.channels,
  ),
  ScopeInfo(
    'channel:receive',
    'Receive from channels',
    'Verify incoming webhook traffic from chat platforms. Webhooks land at /api/v1/channels/<id>/inbound.',
    ScopeGroup.channels,
  ),
  ScopeInfo(
    'event:subscribe:session.*',
    'Subscribe: session events',
    'Stream session.started / session.idle / session.stopped / session.ended events over the integrations WebSocket.',
    ScopeGroup.events,
  ),
  ScopeInfo(
    'event:subscribe:channel.*',
    'Subscribe: channel events',
    'Stream channel.message_sent, channel.message_forwarded, channel.command_received events.',
    ScopeGroup.events,
  ),
  ScopeInfo(
    'event:subscribe:integration.*',
    'Subscribe: integration events',
    'Stream integration.registered, integration.health_changed, integration.key_rotated events.',
    ScopeGroup.events,
  ),
  ScopeInfo(
    'provider:read',
    'Read providers',
    'List installed agent providers (claude, codex, antigravity, shell) and their catalog metadata.',
    ScopeGroup.misc,
  ),
  ScopeInfo(
    'memory:read',
    'Read memory',
    'Search, list, and read stored memories. Used by the memory MCP for live cross-session recall. Does not grant delete/re-embed (admin only).',
    ScopeGroup.misc,
  ),
  ScopeInfo(
    'memory:write',
    'Write memory',
    'Store new memories. Used by the memory MCP so agents can persist durable facts. Does not grant delete/re-embed (admin only).',
    ScopeGroup.misc,
  ),
];

class ScopeGroupMeta {
  const ScopeGroupMeta(this.id, this.label, this.blurb);
  final ScopeGroup id;
  final String label;
  final String blurb;
}

const List<ScopeGroupMeta> kScopeGroups = [
  ScopeGroupMeta(
    ScopeGroup.sessions,
    'Sessions',
    'Drive PTY-backed agent sessions: list, spawn, send input.',
  ),
  ScopeGroupMeta(
    ScopeGroup.channels,
    'Channels',
    'Push to and receive from external chat platforms.',
  ),
  ScopeGroupMeta(
    ScopeGroup.events,
    'Event subscriptions',
    'Live-tail topics from the gateway event bus over WebSocket. Each subscription is a separate scope.',
  ),
  ScopeGroupMeta(
    ScopeGroup.misc,
    'Other',
    "Catalog reads + anything that doesn't belong above.",
  ),
];
