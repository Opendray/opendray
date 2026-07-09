// Database tool API client — per-project external database connections,
// schema introspection, table data browsing, row CRUD and a SQL console.
// Wraps the /api/v1/dbtool/* endpoints via the shared api<T>() helper.
import { api } from './api'

export type DBDriver = 'postgres' | 'mysql' | 'mariadb' | 'sqlite'

// Drivers offered in the connection form, in display order.
export const DB_DRIVERS: readonly DBDriver[] = [
  'postgres',
  'mysql',
  'mariadb',
  'sqlite',
]

// Well-known default port per driver (0 for the file-based SQLite).
export const DB_DEFAULT_PORTS: Record<DBDriver, number> = {
  postgres: 5432,
  mysql: 3306,
  mariadb: 3306,
  sqlite: 0,
}

// SQLite is a file-path connection (db_name is the path); the others use
// host/port/username/password/ssl.
export function driverUsesServer(driver: DBDriver): boolean {
  return driver !== 'sqlite'
}

export interface DBConnection {
  id: string
  cwd: string
  name: string
  driver: DBDriver
  host: string
  port: number
  db_name: string
  username: string
  ssl_mode: string
  read_only: boolean
  options: Record<string, unknown>
  created_at: string
  updated_at: string
  has_password: boolean
}

export interface DBConnectionInput {
  cwd: string
  name: string
  driver?: DBDriver
  host: string
  port: number
  db_name: string
  username: string
  password?: string
  ssl_mode?: string
  read_only?: boolean
  options?: Record<string, unknown>
}

export interface DBConnectionPatch {
  name?: string
  host?: string
  port?: number
  db_name?: string
  username?: string
  password?: string
  ssl_mode?: string
  read_only?: boolean
  options?: Record<string, unknown>
}

export interface DBPingResult {
  ok: boolean
  server_version?: string
  is_superuser: boolean
  latency_ms: number
  error?: string
}

export interface DBSchema {
  name: string
}

export interface DBTable {
  name: string
  kind: 'table' | 'view' | 'foreign'
  row_estimate: number
}

export interface DBColumn {
  name: string
  data_type: string
  nullable: boolean
  default?: string
  position: number
}

export interface DBIndex {
  name: string
  definition: string
  unique: boolean
  primary: boolean
}

export interface DBForeignKey {
  name: string
  columns: string[]
  ref_schema: string
  ref_table: string
  ref_columns: string[]
}

export interface DBTableMeta {
  schema: string
  table: string
  columns: DBColumn[]
  primary_key: string[]
  indexes: DBIndex[]
  foreign_keys: DBForeignKey[]
}

export interface DBFilter {
  column: string
  op: string
  value?: unknown
}

export interface DBSort {
  column: string
  desc?: boolean
}

export interface DBTableDataReq {
  schema: string
  table: string
  limit?: number
  offset?: number
  sort?: DBSort[]
  filters?: DBFilter[]
}

export interface DBColumnMeta {
  name: string
  type: string
}

export interface DBResultSet {
  columns: DBColumnMeta[]
  rows: unknown[][]
  rows_affected: number
  command?: string
  truncated: boolean
  duration_ms: number
}

// Whitelisted filter operators, mirroring the backend's filterOps.
export const DB_FILTER_OPS = [
  '=',
  '!=',
  '<',
  '>',
  '<=',
  '>=',
  'LIKE',
  'ILIKE',
  'NOT LIKE',
  'NOT ILIKE',
  'IS NULL',
  'IS NOT NULL',
] as const

// Operators that don't take a value (the value input is hidden).
export const DB_VALUELESS_OPS = new Set(['IS NULL', 'IS NOT NULL'])

// Filter operators available for a given driver. ILIKE is PostgreSQL-only;
// MySQL/SQLite reject it, so they get the shared subset.
export function dbFilterOps(driver: DBDriver): readonly string[] {
  if (driver === 'postgres') return DB_FILTER_OPS
  return DB_FILTER_OPS.filter((op) => op !== 'ILIKE' && op !== 'NOT ILIKE')
}

export async function listConnections(cwd: string): Promise<DBConnection[]> {
  const res = await api<{ connections: DBConnection[] }>(
    `/api/v1/dbtool/connections?cwd=${encodeURIComponent(cwd)}`,
  )
  return res.connections ?? []
}

export async function createConnection(
  input: DBConnectionInput,
): Promise<DBConnection> {
  return api<DBConnection>('/api/v1/dbtool/connections', {
    method: 'POST',
    body: input,
  })
}

export async function updateConnection(
  id: string,
  patch: DBConnectionPatch,
): Promise<DBConnection> {
  return api<DBConnection>(`/api/v1/dbtool/connections/${id}`, {
    method: 'PATCH',
    body: patch,
  })
}

export async function deleteConnection(id: string): Promise<void> {
  await api(`/api/v1/dbtool/connections/${id}`, { method: 'DELETE' })
}

// testConnectionParams tests an unsaved connection payload.
export async function testConnectionParams(
  input: DBConnectionInput,
): Promise<DBPingResult> {
  return api<DBPingResult>('/api/v1/dbtool/connections/test', {
    method: 'POST',
    body: input,
  })
}

// testConnection tests a saved connection by id (stored credentials).
export async function testConnection(id: string): Promise<DBPingResult> {
  return api<DBPingResult>(`/api/v1/dbtool/connections/${id}/test`, {
    method: 'POST',
  })
}

export async function listSchemas(id: string): Promise<DBSchema[]> {
  const res = await api<{ schemas: DBSchema[] }>(
    `/api/v1/dbtool/connections/${id}/schemas`,
  )
  return res.schemas ?? []
}

export async function listTables(
  id: string,
  schema: string,
): Promise<DBTable[]> {
  const res = await api<{ tables: DBTable[] }>(
    `/api/v1/dbtool/connections/${id}/schemas/${encodeURIComponent(schema)}/tables`,
  )
  return res.tables ?? []
}

export async function getTableMeta(
  id: string,
  schema: string,
  table: string,
): Promise<DBTableMeta> {
  return api<DBTableMeta>(
    `/api/v1/dbtool/connections/${id}/schemas/${encodeURIComponent(
      schema,
    )}/tables/${encodeURIComponent(table)}/meta`,
  )
}

export async function getTableData(
  id: string,
  req: DBTableDataReq,
): Promise<DBResultSet> {
  return api<DBResultSet>(`/api/v1/dbtool/connections/${id}/table-data`, {
    method: 'POST',
    body: req,
  })
}

export async function runQuery(
  id: string,
  sql: string,
  maxRows?: number,
): Promise<DBResultSet> {
  return api<DBResultSet>(`/api/v1/dbtool/connections/${id}/query`, {
    method: 'POST',
    body: { sql, max_rows: maxRows },
  })
}

export async function insertRow(
  id: string,
  schema: string,
  table: string,
  values: Record<string, unknown>,
): Promise<DBResultSet> {
  return api<DBResultSet>(`/api/v1/dbtool/connections/${id}/rows/insert`, {
    method: 'POST',
    body: { schema, table, values },
  })
}

export async function updateRow(
  id: string,
  schema: string,
  table: string,
  pk: Record<string, unknown>,
  values: Record<string, unknown>,
): Promise<{ rows_affected: number }> {
  return api<{ rows_affected: number }>(
    `/api/v1/dbtool/connections/${id}/rows/update`,
    { method: 'POST', body: { schema, table, pk, values } },
  )
}

export async function deleteRows(
  id: string,
  schema: string,
  table: string,
  pks: Record<string, unknown>[],
): Promise<{ rows_affected: number }> {
  return api<{ rows_affected: number }>(
    `/api/v1/dbtool/connections/${id}/rows/delete`,
    { method: 'POST', body: { schema, table, pks } },
  )
}
