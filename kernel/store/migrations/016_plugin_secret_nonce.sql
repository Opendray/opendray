-- Add the AES-GCM nonce column to plugin_secret. AES-GCM requires a
-- unique 12-byte nonce per encryption; we store it alongside the
-- ciphertext because deriving it from (plugin, key) would leak row
-- equality if the plaintext ever repeats.
--
-- Default is the empty bytea so any pre-M3 rows (there are none — the
-- secret namespace didn't ship in M2) are identifiable by a zero-length
-- nonce and can be re-keyed in place on first read.
--
-- M3 T6. Encryption lands in T13.
ALTER TABLE plugin_secret
    ADD COLUMN IF NOT EXISTS nonce BYTEA NOT NULL DEFAULT ''::bytea;
