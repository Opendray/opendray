import type { Client } from "./client.js";
import type { EventFrame } from "./types.js";

export interface SubscribeOptions {
  /** Topic patterns. Use `session.*` for prefix wildcards. */
  topics: string[];
  /** AbortSignal closes the socket and ends the iterator. */
  signal?: AbortSignal;
  /**
   * Custom WebSocket implementation. Defaults to globalThis.WebSocket.
   * On Node <22 you'll need to pass `ws`'s default export.
   */
  WebSocket?: typeof globalThis.WebSocket;
  /** Reconnect on unexpected close. Defaults to true. */
  reconnect?: boolean;
  /** Initial backoff in ms (doubles up to 30s). Defaults to 1000. */
  reconnectBaseMs?: number;
}

/**
 * Returns an async iterable that yields one EventFrame per server message.
 * Reconnects automatically on transient drops; an AbortSignal ends the
 * stream cleanly.
 *
 * ```ts
 * for await (const frame of subscribeEvents(client, { topics: ["session.*"] })) {
 *   console.log(frame.topic, frame.data);
 * }
 * ```
 */
export async function* subscribeEvents<T = unknown>(
  client: Client,
  opts: SubscribeOptions,
): AsyncGenerator<EventFrame<T>, void, void> {
  const WS = opts.WebSocket ?? globalThis.WebSocket;
  if (!WS) {
    throw new Error(
      "subscribeEvents: no WebSocket available. On Node <22 import 'ws' and pass it via `WebSocket`.",
    );
  }

  const url = client.eventsUrl(opts.topics);
  const reconnect = opts.reconnect ?? true;
  const baseDelay = opts.reconnectBaseMs ?? 1000;
  let attempt = 0;

  while (true) {
    if (opts.signal?.aborted) return;
    const queue: EventFrame<T>[] = [];
    let resolve: (() => void) | null = null;
    let closed = false;
    let closeReason: Error | null = null;

    const ws = new WS(url);
    ws.addEventListener("message", (ev: MessageEvent) => {
      try {
        const frame = JSON.parse(String(ev.data)) as EventFrame<T>;
        queue.push(frame);
        if (resolve) {
          const r = resolve;
          resolve = null;
          r();
        }
      } catch (err) {
        closeReason = err instanceof Error ? err : new Error(String(err));
        try {
          ws.close();
        } catch {
          /* ignore */
        }
      }
    });
    ws.addEventListener("close", () => {
      closed = true;
      if (resolve) {
        const r = resolve;
        resolve = null;
        r();
      }
    });
    ws.addEventListener("error", (ev: Event) => {
      closeReason = new Error(`websocket error: ${(ev as ErrorEvent).message ?? "unknown"}`);
    });

    const onAbort = () => {
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    };
    opts.signal?.addEventListener("abort", onAbort, { once: true });

    try {
      while (!closed) {
        while (queue.length > 0) {
          const frame = queue.shift()!;
          yield frame;
          attempt = 0; // reset backoff on successful delivery
        }
        if (closed) break;
        await new Promise<void>((r) => {
          resolve = r;
        });
      }
    } finally {
      opts.signal?.removeEventListener("abort", onAbort);
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    }

    if (opts.signal?.aborted) return;
    if (closeReason) throw closeReason;
    if (!reconnect) return;

    attempt += 1;
    const delay = Math.min(30_000, baseDelay * 2 ** (attempt - 1));
    await new Promise((r) => setTimeout(r, delay));
  }
}
