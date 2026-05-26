// Scope metadata for the integration registration / edit flows.
//
// Every entry in ALL_SCOPES (lib/types.ts) needs a row here so the
// ScopePicker can render a human-readable label + a one-line
// summary of what the scope unlocks. Group is the section header
// — keep them small and predictable.
//
// When the gateway gains a new scope: add it to ALL_SCOPES first,
// then add a SCOPE_INFO entry below. The picker falls back to the
// raw token + "no description" hint when missing.

import { ALL_SCOPES } from './types'

export type ScopeGroup = 'sessions' | 'channels' | 'events' | 'misc'

export interface ScopeInfo {
  id: string
  title: string
  description: string
  group: ScopeGroup
}

const RAW: Record<string, Omit<ScopeInfo, 'id'>> = {
  'session:read': {
    title: 'Read sessions',
    description:
      'List sessions, view session metadata, read terminal buffer, fetch project history.',
    group: 'sessions',
  },
  'session:create': {
    title: 'Create sessions',
    description:
      'Spawn new sessions, restart ended ones, delete sessions when done.',
    group: 'sessions',
  },
  'session:input': {
    title: 'Send input',
    description:
      'Forward keystrokes / commands into a session\'s PTY. Required to drive an agent CLI.',
    group: 'sessions',
  },
  'channel:send': {
    title: 'Send to channels',
    description:
      'Push notifications and messages out through a registered channel (Telegram, Slack, etc.).',
    group: 'channels',
  },
  'channel:receive': {
    title: 'Receive from channels',
    description:
      'Verify incoming webhook traffic from chat platforms. Webhooks land at /api/v1/channels/<id>/inbound.',
    group: 'channels',
  },
  'event:subscribe:session.*': {
    title: 'Subscribe: session events',
    description:
      'Stream session.started / session.idle / session.stopped / session.ended events over the integrations WebSocket.',
    group: 'events',
  },
  'event:subscribe:channel.*': {
    title: 'Subscribe: channel events',
    description:
      'Stream channel.message_sent, channel.message_forwarded, channel.command_received events.',
    group: 'events',
  },
  'event:subscribe:integration.*': {
    title: 'Subscribe: integration events',
    description:
      'Stream integration.registered, integration.health_changed, integration.key_rotated events.',
    group: 'events',
  },
  'provider:read': {
    title: 'Read providers',
    description:
      'List installed agent providers (claude, codex, gemini, shell) and their catalog metadata.',
    group: 'misc',
  },
  'memory:read': {
    title: 'Read memory',
    description:
      'Search, list, and read stored memories. Used by the memory MCP for live cross-session recall. Does not grant delete/re-embed (admin only).',
    group: 'misc',
  },
  'memory:write': {
    title: 'Write memory',
    description:
      'Store new memories. Used by the memory MCP so agents can persist durable facts. Does not grant delete/re-embed (admin only).',
    group: 'misc',
  },
}

export const SCOPE_INFO: ScopeInfo[] = ALL_SCOPES.map((id) => {
  const meta = RAW[id]
  if (meta) return { id, ...meta }
  // Unknown scope from a future opendray version — keep it in the
  // picker so the operator can still toggle it, just without prose.
  return {
    id,
    title: id,
    description: 'No description available for this scope yet.',
    group: 'misc' as ScopeGroup,
  }
})

export const SCOPE_GROUPS: { id: ScopeGroup; label: string; blurb: string }[] = [
  {
    id: 'sessions',
    label: 'Sessions',
    blurb: 'Drive PTY-backed agent sessions: list, spawn, send input.',
  },
  {
    id: 'channels',
    label: 'Channels',
    blurb: 'Push to and receive from external chat platforms.',
  },
  {
    id: 'events',
    label: 'Event subscriptions',
    blurb:
      'Live-tail topics from the gateway event bus over WebSocket. Each subscription is a separate scope.',
  },
  {
    id: 'misc',
    label: 'Other',
    blurb: 'Catalog reads + anything that doesn\'t belong above.',
  },
]
