-- Remember each session's last client terminal size so a (re)spawn starts
-- the PTY at it instead of the fixed 80x24 floor. A full-screen TUI (grok
-- especially) then comes up already matching the last window, and only
-- needs the browser's fit to confirm — removing the visible "tiny then
-- jump" and stale-size render on open/restart. NULL = never sized; the
-- manager falls back to the default floor. SMALLINT holds the sane range
-- (bounded to <=500x300 by the manager before use).
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS term_cols SMALLINT,
    ADD COLUMN IF NOT EXISTS term_rows SMALLINT;
