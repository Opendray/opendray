-- Git credentials are scoped per host + owner, not per host.
--
-- One row per hostname assumed one identity per forge. That holds until
-- you touch a personal repo and an org repo on the same host: a
-- fine-grained GitHub token is granted per repository, so a token that
-- reaches github.com/<you>/... generally cannot reach
-- github.com/<org>/... — and there was no way to store both. GitHub's
-- reply in that case is a 403 reading "Write access to repository not
-- granted" even for a fetch, which sends people looking for a bug in
-- the gateway instead of at their token's repository list.
--
-- owner = '' is the host-wide fallback, which is exactly what every
-- existing row becomes. Nothing to migrate, nothing to re-enter: a
-- deployment that never needs a second identity behaves as before.

ALTER TABLE git_hosts
  ADD COLUMN IF NOT EXISTS owner TEXT NOT NULL DEFAULT '';

-- Replace the host-unique constraint with (host, owner). Dropping by the
-- generated name is safe here: it is what CREATE TABLE ... UNIQUE
-- produced, and IF EXISTS keeps a re-run harmless.
ALTER TABLE git_hosts DROP CONSTRAINT IF EXISTS git_hosts_host_key;

-- Owners are case-insensitive on every forge we support, so the
-- uniqueness has to be too, or github.com/Opendray and
-- github.com/opendray become two rows that resolve unpredictably.
CREATE UNIQUE INDEX IF NOT EXISTS git_hosts_host_owner_key
  ON git_hosts (host, lower(owner));
