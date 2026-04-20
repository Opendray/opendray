#!/usr/bin/env node
// pg-browser — v1 host-form sidecar. Speaks JSON-RPC 2.0 over stdio
// with LSP Content-Length framing (see plugin/host/jsonrpc.go).
//
// Configuration flow:
//
//   - The platform writes user-entered config into plugin_kv (non-
//     secret fields) and plugin_secret (secret fields) at install
//     time + every "Configure" save. Keys are prefixed with
//     "__config." so they don't collide with the plugin's own
//     runtime storage.
//
//   - This sidecar reads those values on startup via outbound JSON-
//     RPC calls to storage/get and secret/get. When the user saves
//     new config, the gateway kills the sidecar; the supervisor
//     respawns it on the next invoke and we re-read everything fresh.
//
//   - If any required field is missing (user skipped configure or
//     PUT-cleared it) every method returns a friendly error pointing
//     to the Configure action.
//
// stdout is reserved for the RPC channel; diagnostics go to stderr.

'use strict';

const { stdin, stdout, stderr } = process;

// ─── JSON-RPC framing ────────────────────────────────────────────────

const pending = new Map();
let nextId = 1;

function send(msg) {
  const body = Buffer.from(JSON.stringify(msg), 'utf8');
  stdout.write(`Content-Length: ${body.length}\r\n\r\n`);
  stdout.write(body);
}

function log(...args) {
  stderr.write(args.join(' ') + '\n');
}

function startReader(onMessage) {
  let buf = Buffer.alloc(0);
  stdin.on('data', (chunk) => {
    buf = Buffer.concat([buf, chunk]);
    for (;;) {
      const headerEnd = buf.indexOf('\r\n\r\n');
      if (headerEnd < 0) return;
      const header = buf.slice(0, headerEnd).toString('utf8');
      const m = header.match(/Content-Length:\s*(\d+)/i);
      if (!m) { buf = buf.slice(headerEnd + 4); continue; }
      const len = parseInt(m[1], 10);
      if (buf.length < headerEnd + 4 + len) return;
      const body = buf.slice(headerEnd + 4, headerEnd + 4 + len).toString('utf8');
      buf = buf.slice(headerEnd + 4 + len);
      try { onMessage(JSON.parse(body)); }
      catch (e) { log(`pg-browser: parse error: ${e}`); }
    }
  });
  stdin.on('end', () => process.exit(0));
}

// call makes an outbound JSON-RPC request and returns a Promise that
// resolves with result / rejects with {code,message}. The Go host side
// dispatches to the right namespace based on the "ns/method" prefix.
function call(method, params) {
  return new Promise((resolve, reject) => {
    const id = String(nextId++);
    pending.set(id, { resolve, reject });
    send({ jsonrpc: '2.0', id, method, params });
  });
}

// ─── Config reader ───────────────────────────────────────────────────
// Both storage.get and secret.get return null when the key is absent.
// The platform's config PUT stores non-secret values as JSON strings
// (e.g. "\"5432\"" for the port), so we always get back a string here;
// the caller does any int/bool coercion.

const CONFIG_PREFIX = '__config.';

async function readConfig() {
  const [host, port, user, db, sslMode, password] = await Promise.all([
    call('storage/get', [CONFIG_PREFIX + 'host']),
    call('storage/get', [CONFIG_PREFIX + 'port']),
    call('storage/get', [CONFIG_PREFIX + 'user']),
    call('storage/get', [CONFIG_PREFIX + 'database']),
    call('storage/get', [CONFIG_PREFIX + 'sslMode']),
    call('secret/get',  [CONFIG_PREFIX + 'password']),
  ]);
  return {
    host: host || '',
    port: port ? parseInt(port, 10) : 5432,
    user: user || '',
    database: db || 'postgres',
    sslMode: sslMode || 'disable',
    password: password || '',
  };
}

function assertConfigured(cfg) {
  const missing = [];
  for (const k of ['host', 'user', 'password']) {
    if (!cfg[k]) missing.push(k);
  }
  if (missing.length) {
    throw new Error(
      `pg-browser is not configured — ${missing.join(', ')} missing. ` +
      `Open Plugin → pg-browser → Configure to set connection details.`,
    );
  }
}

// ─── pg client (lazy singleton) ──────────────────────────────────────

let pg;
try {
  pg = require('pg');
} catch (err) {
  log('pg-browser: FATAL — node-postgres (pg) not bundled with plugin:', err.message);
}

let client = null;

async function getClient() {
  if (client) return client;
  if (!pg) throw new Error('node-postgres not installed in plugin bundle');
  const cfg = await readConfig();
  assertConfigured(cfg);
  const { Client } = pg;
  const c = new Client({
    host: cfg.host,
    port: cfg.port,
    user: cfg.user,
    password: cfg.password,
    database: cfg.database,
    ssl: cfg.sslMode && cfg.sslMode !== 'disable'
      ? { rejectUnauthorized: cfg.sslMode === 'verify-full' }
      : false,
    connectionTimeoutMillis: 8000,
    statement_timeout: 15000,
  });
  await c.connect();
  client = c;
  log(`pg-browser: connected to ${cfg.user}@${cfg.host}:${cfg.port}/${cfg.database}`);
  return client;
}

// ─── Method implementations ──────────────────────────────────────────

async function methodInfo() {
  let status = 'disconnected';
  let serverVersion = null;
  let config = null;
  try {
    const cfg = await readConfig();
    config = {
      host: cfg.host || '(unset)',
      port: cfg.port,
      user: cfg.user || '(unset)',
      database: cfg.database,
      sslMode: cfg.sslMode,
      passwordSet: Boolean(cfg.password),
    };
    const c = await getClient();
    const { rows } = await c.query('SELECT version() AS v');
    serverVersion = rows[0] && rows[0].v;
    status = 'connected';
  } catch (err) {
    status = `error: ${err.message}`;
  }
  return { status, config, serverVersion };
}

async function methodListDatabases() {
  const c = await getClient();
  const { rows } = await c.query(`
    SELECT datname AS name,
           pg_size_pretty(pg_database_size(datname)) AS size
    FROM pg_database
    WHERE datistemplate = false
    ORDER BY datname
  `);
  return { count: rows.length, databases: rows };
}

async function methodListSchemas() {
  const c = await getClient();
  const { rows } = await c.query(`
    SELECT nspname AS name,
           pg_catalog.pg_get_userbyid(nspowner) AS owner
    FROM pg_namespace
    WHERE nspname NOT LIKE 'pg_%'
      AND nspname != 'information_schema'
    ORDER BY nspname
  `);
  return { count: rows.length, schemas: rows };
}

async function methodListTables() {
  const c = await getClient();
  const { rows } = await c.query(`
    SELECT table_schema AS schema,
           table_name AS name,
           table_type AS type
    FROM information_schema.tables
    WHERE table_schema = 'public'
    ORDER BY table_name
  `);
  return { count: rows.length, tables: rows };
}

async function methodSampleQuery() {
  const c = await getClient();
  const sql = `
    SELECT schemaname AS schema,
           COUNT(*) AS tables
    FROM pg_catalog.pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    GROUP BY schemaname
    ORDER BY schemaname
  `;
  const { rows } = await c.query(sql);
  return { sql: sql.trim(), rowCount: rows.length, rows };
}

const methods = {
  info: methodInfo,
  listDatabases: methodListDatabases,
  listSchemas: methodListSchemas,
  listTables: methodListTables,
  sampleQuery: methodSampleQuery,
};

// ─── RPC dispatch ────────────────────────────────────────────────────

async function handleRequest(msg) {
  const { id, method, params } = msg;
  const fn = methods[method];
  if (typeof fn !== 'function') {
    send({
      jsonrpc: '2.0', id,
      error: { code: -32601, message: `unknown method: ${method}` },
    });
    return;
  }
  try {
    const result = await fn(params || {});
    send({ jsonrpc: '2.0', id, result });
  } catch (err) {
    log(`pg-browser: method ${method} failed:`, err.message);
    send({
      jsonrpc: '2.0', id,
      error: { code: -32000, message: err.message || String(err) },
    });
  }
}

startReader((msg) => {
  // Inbound request: method present, id present → dispatch.
  if (msg && typeof msg.method === 'string' && typeof msg.id !== 'undefined') {
    handleRequest(msg);
    return;
  }
  // Inbound response to an outbound call we made.
  if (msg && typeof msg.id !== 'undefined'
    && (msg.result !== undefined || msg.error)) {
    const p = pending.get(msg.id);
    if (!p) return;
    pending.delete(msg.id);
    if (msg.error) {
      p.reject(new Error(`${msg.error.code}: ${msg.error.message}`));
    } else {
      p.resolve(msg.result);
    }
  }
});

process.on('SIGTERM', () => {
  log('pg-browser: SIGTERM — closing pg client');
  if (client) client.end().catch(() => {});
  process.exit(0);
});

log('pg-browser: ready');
