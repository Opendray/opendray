import type { Client } from "./client.js";

export interface SessionStreamHandle {
  /** Send raw bytes to the session's PTY stdin. */
  send(data: string): void;
  /** Close the stream. */
  close(): void;
  /** Resolves when the socket closes (cleanly or with an error). */
  readonly closed: Promise<void>;
}

export interface SessionStreamOptions {
  /** Called with each chunk of terminal output. */
  onOutput?: (data: string) => void;
  /** Called when the server reports the session has ended. */
  onEnded?: (info: { exitCode?: number | undefined; reason?: string | undefined }) => void;
  /** AbortSignal closes the socket. */
  signal?: AbortSignal;
  /**
   * Custom WebSocket implementation. Defaults to globalThis.WebSocket.
   * On Node <22 pass `ws`'s default export.
   */
  WebSocket?: typeof globalThis.WebSocket;
}

/**
 * Open a bidirectional terminal stream against the given session.
 *
 * The wire format is gateway-specific JSON frames:
 *   { kind: "output", data: "<chunk>" }   (server → client)
 *   { kind: "ended", exit_code?, reason? } (server → client)
 *   { kind: "input",  data: "<chunk>" }    (client → server)
 *
 * The returned handle exposes `send()` and `close()`, and a `closed`
 * promise that resolves when the socket finishes.
 */
export function streamSession(
  client: Client,
  sessionId: string,
  opts: SessionStreamOptions = {},
): SessionStreamHandle {
  const WS = opts.WebSocket ?? globalThis.WebSocket;
  if (!WS) {
    throw new Error(
      "streamSession: no WebSocket available. On Node <22 import 'ws' and pass it via `WebSocket`.",
    );
  }

  const ws = new WS(client.sessionStreamUrl(sessionId));
  let resolveClosed!: () => void;
  const closed = new Promise<void>((r) => {
    resolveClosed = r;
  });

  ws.addEventListener("message", (ev: MessageEvent) => {
    let frame: { kind?: string; data?: string; exit_code?: number; reason?: string };
    try {
      frame = JSON.parse(String(ev.data));
    } catch {
      return;
    }
    if (frame.kind === "output" && typeof frame.data === "string") {
      opts.onOutput?.(frame.data);
    } else if (frame.kind === "ended") {
      opts.onEnded?.({ exitCode: frame.exit_code, reason: frame.reason });
    }
  });
  ws.addEventListener("close", () => {
    resolveClosed();
  });

  const onAbort = () => {
    try {
      ws.close();
    } catch {
      /* ignore */
    }
  };
  opts.signal?.addEventListener("abort", onAbort, { once: true });

  return {
    send(data: string) {
      if (ws.readyState !== WS.OPEN) {
        throw new Error("streamSession: socket not open");
      }
      ws.send(JSON.stringify({ kind: "input", data }));
    },
    close() {
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    },
    closed,
  };
}
