#!/usr/bin/env node
// pg-browser — v1 host-form sidecar. Speaks JSON-RPC 2.0 over stdio
// with LSP Content-Length framing (see plugin/host/jsonrpc.go).
//
// Connection parameters come from the PG_* env vars declared in
// manifest.host.env. The host platform's supervisor filters env
// aggressively (PATH/HOME/USER/LANG/TMPDIR + manifest's own env);
// arbitrary host env doesn't leak into the sidecar.
//
// Queries are read-only by convention. listDatabases / listSchemas /
// listTables hit pg_catalog views; sampleQuery runs a single
// SELECT COUNT(*). Nothing here writes, so the plugin is safe to
// expose even with a privileged role (though a dedicated read-only
// role is still recommended).

'use strict';

const { stdin, stdout, stderr } = process;

// ─── JSON-RPC framing ────────────────────────────────────────────────
// LSP-style Content-Length framing. The host supervisor uses the same
// format in both directions, so we only need one parser + one writer.

function writeMessage(msg) {
  const body = Buffer.from(JSON.stringify(msg), 'utf8');
  stdout.write(`Content-Length: ${body.length}\r\n\r\n`);
  stdout.write(body);
}

function log(...args) {
  // stderr is drained by the supervisor at Info level — stdout is
  // reserved for the JSON-RPC channel.
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
      const match = /Content-Length:\s*(\d+)/i.exec(header);
      if (!match) {
        log('pg-browser: malformed header; dropping buffer');
        buf = Buffer.alloc(0);
        return;
      }
      const bodyLen = parseInt(match[1], 10);
      const bodyStart = headerEnd + 4;
      if (buf.length < bodyStart + bodyLen) return;
      const body = buf.slice(bodyStart, bodyStart + bodyLen).toString('utf8');
      buf = buf.slice(bodyStart + bodyLen);
      try {
        onMessage(JSON.parse(body));
      } catch (err) {
        log('pg-browser: JSON parse error:', err.message);
      }
    }
  });
  stdin.on('end', () => process.exit(0));
}

// ─── pg client (lazy singleton) ──────────────────────────────────────
// One long-lived Client per sidecar lifetime. idleShutdownMinutes in
// the manifest kills the whole process when there's no traffic, so
// connection leaks are bounded by that timer.

let pg;
try {
  pg = require('pg');
} catch (err) {
  log('pg-browser: FATAL — node-postgres (pg) not bundled with plugin:', err.message);
  // Don't exit — let method calls surface the error so the user sees
  // it in the command result instead of a silent crash.
}

let client = null;
let clientError = null;

async function getClient() {
  if (client) return client;
  if (!pg) {
    throw new Error('node-postgres not installed in plugin bundle');
  }
  const { Client } = pg;
  const cfg = {
    host: process.env.PG_HOST,
    port: parseInt(process.env.PG_PORT || '5432', 10),
    user: process.env.PG_USER,
    password: process.env.PG_PASSWORD,
    database: process.env.PG_DATABASE || 'postgres',
    ssl: process.env.PG_SSLMODE && process.env.PG_SSLMODE !== 'disable'
      ? { rejectUnauthorized: process.env.PG_SSLMODE === 'verify-full' }
      : false,
    connectionTimeoutMillis: 8000,
    statement_timeout: 15000,
  };
  for (const key of ['host', 'user', 'password']) {
    if (!cfg[key] || cfg[key] === '__REPLACE_ME__') {
      throw new Error(
        `PG_${key.toUpperCase()} is not configured — the plugin operator ` +
        `must set manifest.host.env.PG_${key.toUpperCase()} before publishing.`,
      );
    }
  }
  const c = new Client(cfg);
  await c.connect();
  client = c;
  log(`pg-browser: connected to ${cfg.user}@${cfg.host}:${cfg.port}/${cfg.database}`);
  return client;
}

// ─── Method implementations ──────────────────────────────────────────

async function methodInfo() {
  const cfg = {
    host: process.env.PG_HOST,
    port: parseInt(process.env.PG_PORT || '5432', 10),
    user: process.env.PG_USER,
    database: process.env.PG_DATABASE,
    sslmode: process.env.PG_SSLMODE || 'disable',
  };
  let status = 'disconnected';
  let serverVersion = null;
  try {
    const c = await getClient();
    const { rows } = await c.query('SELECT version() AS v');
    serverVersion = rows[0] && rows[0].v;
    status = 'connected';
  } catch (err) {
    status = `error: ${err.message}`;
  }
  return { status, config: cfg, serverVersion };
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
    writeMessage({
      jsonrpc: '2.0',
      id,
      error: { code: -32601, message: `unknown method: ${method}` },
    });
    return;
  }
  try {
    const result = await fn(params || {});
    writeMessage({ jsonrpc: '2.0', id, result });
  } catch (err) {
    log(`pg-browser: method ${method} failed:`, err.message);
    writeMessage({
      jsonrpc: '2.0',
      id,
      error: { code: -32000, message: err.message || String(err) },
    });
  }
}

startReader((msg) => {
  // Only requests (those with an id) get replies. Notifications
  // without an id are silently dropped — the platform doesn't push
  // any today, but keeping the branch makes the wire contract explicit.
  if (msg && typeof msg.id !== 'undefined') {
    handleRequest(msg);
  }
});

process.on('SIGTERM', () => {
  log('pg-browser: SIGTERM — closing pg client');
  if (client) client.end().catch(() => {});
  process.exit(0);
});

log('pg-browser: ready');
