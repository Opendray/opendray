-- 0024_device_tokens.sql
--
-- Schema for the mobile device-token registry. Used by the future
-- push-notification subsystem to look up which devices to alert on
-- session.idle / session.ended / errors. The schema lands now (B2)
-- so the surface is stable before the producer side ships in
-- phase C; no rows are populated yet.
--
-- See ADR 0015 §6 for the full mobile-protocol design.
CREATE TABLE IF NOT EXISTS device_tokens (
  id          TEXT        PRIMARY KEY,             -- e.g. dev_<base32>
  platform    TEXT        NOT NULL,                -- "ios" | "android" | "web" (push-capable)
  -- The vendor-issued push token (APNs device token or FCM registration ID).
  -- Length is platform-specific; ~64 hex for APNs, ~150+ chars for FCM v1.
  push_token  TEXT        NOT NULL,
  -- The auth principal that registered this device. Today this is the
  -- single-admin username; phase-C work may extend to integration IDs
  -- if integrations want to receive their own pushes.
  principal   TEXT        NOT NULL,
  -- Optional human-readable label set during registration ("Kev's
  -- iPhone 17", "Pixel 9 Pro"). UI surfaces this in the device list
  -- so the operator can revoke individual devices.
  label       TEXT        NOT NULL DEFAULT '',
  -- App build that registered the token. Helps debug "why does this
  -- one device get duplicate notifications" — usually it's an old
  -- build that hasn't called DELETE on its previous token.
  app_version TEXT        NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Updated whenever the same device re-registers (token refresh).
  -- A device that hasn't heartbeated in N days is a candidate for
  -- silent revocation.
  last_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lookup index for "fan out a notification to every device of a
-- principal" — the dominant query pattern.
CREATE INDEX IF NOT EXISTS device_tokens_principal_idx
  ON device_tokens (principal);

-- Uniqueness on the platform+push_token pair so that a device that
-- re-registers (token refresh) UPSERTs into a single row instead of
-- accumulating duplicates.
CREATE UNIQUE INDEX IF NOT EXISTS device_tokens_platform_push_token_uq
  ON device_tokens (platform, push_token);
