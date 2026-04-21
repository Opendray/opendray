#!/usr/bin/env node
// fs-readme — M3 reference host-form sidecar.
//
// Speaks JSON-RPC 2.0 over stdio with LSP Content-Length framing
// (see plugin/host/jsonrpc.go). Implements one inbound method —
// "summarise" — and makes one outbound call — "fs/readFile" — to
// exercise the sidecar -> host -> capability gate path.
//
// stdout is reserved for the RPC channel; all diagnostics go to stderr.

'use strict';

const { stdin, stdout, stderr } = process;
const pending = new Map();
let nextId = 1;

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
      catch (e) { stderr.write(`fs-readme: parse error: ${e}\n`); }
    }
  });
  stdin.on('end', () => process.exit(0));
}

function send(msg) {
  const body = Buffer.from(JSON.stringify(msg), 'utf8');
  stdout.write(`Content-Length: ${body.length}\r\n\r\n`);
  stdout.write(body);
}

function call(method, params) {
  return new Promise((resolve, reject) => {
    const id = String(nextId++);
    pending.set(id, { resolve, reject });
    send({ jsonrpc: '2.0', id, method, params });
  });
}

async function handleRequest(req) {
  try {
    if (req.method === 'summarise') {
      const path = (req.params && req.params.path) || `${process.env.HOME}/README.md`;
      const content = await call('fs/readFile', [path]);
      const text = typeof content === 'string' ? content.slice(0, 400) : '';
      send({ jsonrpc: '2.0', id: req.id, result: { text, path } });
      return;
    }
    send({
      jsonrpc: '2.0', id: req.id,
      error: { code: -32601, message: `unknown method: ${req.method}` },
    });
  } catch (err) {
    send({
      jsonrpc: '2.0', id: req.id,
      error: { code: -32603, message: String(err && err.message ? err.message : err) },
    });
  }
}

startReader((msg) => {
  if (msg.method) {
    handleRequest(msg);
    return;
  }
  if (msg.id !== undefined && (msg.result !== undefined || msg.error)) {
    const p = pending.get(msg.id);
    if (!p) return;
    pending.delete(msg.id);
    if (msg.error) p.reject(new Error(`${msg.error.code}: ${msg.error.message}`));
    else p.resolve(msg.result);
  }
});

stderr.write('fs-readme sidecar ready\n');
