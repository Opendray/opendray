# Round Table — rollback (experimental feature)

Round Table is an **experimental** feature. If it does not pan out, it is
designed to be removed with zero residue. This file is the manual rollback
(the migration runner has no down-migrations — `//go:embed migrations/*.sql`
would re-run any `.sql` placed in that directory, so the DROP statements live
here, NOT in `internal/store/migrations/`).

## What the feature added

- **Code**: the entire `internal/roundtable/` package + one wiring block in
  `internal/app/app.go` (grep `ROUND TABLE (experimental)`), all on branch
  `feat/round-table`.
- **Schema**: migration `0077_round_tables.sql` → tables `round_tables` and
  `round_table_turns`. Touches no existing table, enum, or CHECK constraint.
- **No new `memory_workers` row / TaskKind**: seat calls reuse
  `worker.TaskCuration` purely as the metrics label via `RunWith`, so no
  existing enum or CHECK was widened.

## Rollback steps

1. **Drop the tables** — run against each live DB (`opendray_v2`):

   ```sql
   DROP TABLE IF EXISTS round_table_turns;
   DROP TABLE IF EXISTS round_tables;
   ```

   The runner tracks migration versions individually, so once the tables are
   gone the recorded `0077` version is inert (a re-applied `0077` would just
   re-create empty tables). If you also want the version record gone:

   ```sql
   DELETE FROM schema_migrations WHERE version = '0077_round_tables';
   ```
   (Adjust the table/column name to this deployment's migration ledger.)

2. **Remove the code** — delete the `feat/round-table` branch (never merged) or,
   if merged, revert the merge commit. The package is self-contained; the only
   edit outside it is the marked wiring block in `app.go`.

3. **Redeploy** the `main` binary.

No other subsystem reads these tables, so nothing else needs reverting.
