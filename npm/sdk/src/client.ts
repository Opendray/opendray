import type {
  Channel,
  Integration,
  IntegrationCreated,
  IntegrationRegistration,
  IntegrationUpdate,
  Provider,
  Session,
  SessionBuffer,
  SessionCreateRequest,
  SessionInputRequest,
  SessionResizeRequest,
} from "./types.js";

export interface ClientOptions {
  /** Base URL of the gateway, e.g. `https://opendray.example.com`. */
  baseUrl: string;
  /** Admin bearer token or integration API key (`odk_live_…`). */
  token: string;
  /** Optional custom fetch (e.g. for proxying or instrumentation). */
  fetch?: typeof fetch;
  /** Default request timeout in ms. 30s if omitted. 0 disables. */
  timeoutMs?: number;
}

export class OpendrayError extends Error {
  readonly status: number;
  readonly body: unknown;
  constructor(status: number, body: unknown, message: string) {
    super(message);
    this.name = "OpendrayError";
    this.status = status;
    this.body = body;
  }
}

export class Client {
  readonly baseUrl: string;
  readonly token: string;
  private readonly fetchImpl: typeof fetch;
  private readonly timeoutMs: number;

  constructor(opts: ClientOptions) {
    if (!opts.baseUrl) throw new Error("Client: baseUrl is required");
    if (!opts.token) throw new Error("Client: token is required");
    let trimmed = opts.baseUrl;
    while (trimmed.endsWith("/")) trimmed = trimmed.slice(0, -1);
    this.baseUrl = trimmed;
    this.token = opts.token;
    this.fetchImpl = opts.fetch ?? globalThis.fetch;
    if (!this.fetchImpl) {
      throw new Error(
        "Client: no fetch available — pass `fetch` in options on platforms older than Node 18.",
      );
    }
    this.timeoutMs = opts.timeoutMs ?? 30_000;
  }

  // ── low level ──────────────────────────────────────────────────────

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const url = `${this.baseUrl}/api/v1${path}`;
    const headers: Record<string, string> = {
      Authorization: `Bearer ${this.token}`,
      Accept: "application/json",
    };
    let payload: BodyInit | undefined;
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }

    const ctrl = this.timeoutMs > 0 ? new AbortController() : null;
    const timer = ctrl
      ? setTimeout(() => ctrl.abort(new Error("request timed out")), this.timeoutMs)
      : null;

    const init: RequestInit = { method, headers };
    if (payload !== undefined) init.body = payload;
    if (ctrl) init.signal = ctrl.signal;

    try {
      const res = await this.fetchImpl(url, init);
      const text = await res.text();
      let parsed: unknown = undefined;
      if (text.length > 0) {
        try {
          parsed = JSON.parse(text);
        } catch {
          parsed = text;
        }
      }
      if (!res.ok) {
        const msg =
          (parsed && typeof parsed === "object" && "error" in parsed
            ? String((parsed as { error: unknown }).error)
            : undefined) ?? `${method} ${path} -> ${res.status}`;
        throw new OpendrayError(res.status, parsed, msg);
      }
      return parsed as T;
    } finally {
      if (timer) clearTimeout(timer);
    }
  }

  // ── integrations (admin-only) ──────────────────────────────────────

  registerIntegration(input: IntegrationRegistration): Promise<IntegrationCreated> {
    return this.request("POST", "/integrations", input);
  }

  listIntegrations(): Promise<Integration[]> {
    return this.request("GET", "/integrations");
  }

  getIntegration(id: string): Promise<Integration> {
    return this.request("GET", `/integrations/${encodeURIComponent(id)}`);
  }

  updateIntegration(id: string, patch: IntegrationUpdate): Promise<Integration> {
    return this.request("PATCH", `/integrations/${encodeURIComponent(id)}`, patch);
  }

  deleteIntegration(id: string): Promise<void> {
    return this.request("DELETE", `/integrations/${encodeURIComponent(id)}`);
  }

  rotateIntegrationKey(id: string): Promise<IntegrationCreated> {
    return this.request("POST", `/integrations/${encodeURIComponent(id)}/rotate-key`);
  }

  // ── sessions (dual-auth) ───────────────────────────────────────────

  createSession(input: SessionCreateRequest): Promise<Session> {
    return this.request("POST", "/sessions", input);
  }

  listSessions(): Promise<Session[]> {
    return this.request("GET", "/sessions");
  }

  getSession(id: string): Promise<Session> {
    return this.request("GET", `/sessions/${encodeURIComponent(id)}`);
  }

  deleteSession(id: string): Promise<void> {
    return this.request("DELETE", `/sessions/${encodeURIComponent(id)}`);
  }

  sendInput(id: string, input: SessionInputRequest): Promise<void> {
    return this.request("POST", `/sessions/${encodeURIComponent(id)}/input`, input);
  }

  resizeSession(id: string, size: SessionResizeRequest): Promise<void> {
    return this.request("POST", `/sessions/${encodeURIComponent(id)}/resize`, size);
  }

  getSessionBuffer(id: string): Promise<SessionBuffer> {
    return this.request("GET", `/sessions/${encodeURIComponent(id)}/buffer`);
  }

  // ── providers / channels (dual-auth) ───────────────────────────────

  listProviders(): Promise<Provider[]> {
    return this.request("GET", "/providers");
  }

  setProviderConfig(id: string, config: Record<string, unknown>): Promise<Provider> {
    return this.request(
      "PATCH",
      `/providers/${encodeURIComponent(id)}/config`,
      config,
    );
  }

  listChannels(): Promise<Channel[]> {
    return this.request("GET", "/channels");
  }

  // ── ws helpers (URLs only — open via subscribeEvents / streamSession) ─

  /**
   * Compose the WS URL for the events stream. Used by `subscribeEvents`,
   * exposed publicly for callers that want to manage the socket lifecycle
   * themselves.
   */
  eventsUrl(topics: string[]): string {
    const u = new URL(`${this.baseUrl}/api/v1/integrations/_events`);
    u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
    u.searchParams.set("token", this.token);
    if (topics.length > 0) u.searchParams.set("topics", topics.join(","));
    return u.toString();
  }

  sessionStreamUrl(id: string): string {
    const u = new URL(
      `${this.baseUrl}/api/v1/sessions/${encodeURIComponent(id)}/stream`,
    );
    u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
    u.searchParams.set("token", this.token);
    return u.toString();
  }
}
