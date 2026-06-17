#!/usr/bin/env python3
"""
visayabai ↔ opendray bridge adapter (reference implementation).

This is the thin relay that connects the visayabai bot to opendray as a
`kind=bridge` channel. opendray does all the session routing, reply
detection, and notification policy; this adapter only:

  1. opens a WebSocket to /api/v1/channels/bridge/ws and registers,
  2. forwards each visayabai user message to opendray as a `message` frame,
  3. renders opendray's outbound frames (send / typing / card / buttons)
     back to the visayabai user.

Wire protocol (verified against internal/channel/bridge):
  - connect:  wss://HOST/api/v1/channels/bridge/ws?token=<BRIDGE_TOKEN>
  - 1st frame (adapter→opendray):
        {"type":"register","platform":"visayabai",
         "capabilities":["text","typing"], "metadata":{}}
    opendray replies {"type":"register_ack","ok":true}
  - inbound (adapter→opendray), a user said something:
        {"type":"message","conversation_id":"<stable per-user/chat id>",
         "user_id":"...","user_name":"...","text":"hello"}
    button taps: {"type":"card_action","action":"<id>","conversation_id":...}
  - outbound (opendray→adapter), render to the user — only the frame
    types matching capabilities you declared arrive:
        {"type":"send","conversation_id":...,"text":...}
        {"type":"start_typing",...} / {"type":"stop_typing",...}
        {"type":"send_card",...,"card":{...}}        (declare "card")
        {"type":"send_buttons",...,"text":...,"buttons":[[...]]} ("buttons")
        {"type":"send_image"/"send_file",...}        ("image"/"file")
  - heartbeats: opendray sends WS ping frames; the `websockets` lib
    auto-pongs. Adapter-level {"type":"ping"} → {"type":"pong"} also ok.

`conversation_id` is THE key idea: use a stable id per visayabai user
(or per chat). opendray binds each conversation_id to its own session,
so N users get N independent Claude sessions automatically.

Run:
  pip install websockets
  OPENDRAY_HOST=opendray.example.com:8770 \
  BRIDGE_TOKEN=... \
  python visayabai_bridge_adapter.py
"""

import asyncio
import json
import os
import signal
from typing import Any

import websockets  # pip install websockets

OPENDRAY_HOST = os.environ["OPENDRAY_HOST"]          # host:port, no scheme
BRIDGE_TOKEN = os.environ["BRIDGE_TOKEN"]            # from the bridge channel config
USE_TLS = os.environ.get("OPENDRAY_TLS", "1") != "0"  # wss by default

# Capabilities this adapter can actually render. Start minimal; add
# "card"/"buttons"/"image"/"file" once visayabai can display them — then
# opendray's control keyboard + provider picker render as rich UI instead
# of falling back to plain text.
CAPABILITIES = ["text", "typing"]

_scheme = "wss" if USE_TLS else "ws"
WS_URL = f"{_scheme}://{OPENDRAY_HOST}/api/v1/channels/bridge/ws?token={BRIDGE_TOKEN}"


# ── visayabai side (STUBS — wire these to your real bot) ──────────────
async def send_to_visayabai_user(conversation_id: str, text: str) -> None:
    """Render an opendray reply to the visayabai user identified by
    conversation_id. Replace with your bot's real send call."""
    print(f"[→ visayabai:{conversation_id}] {text}")


async def set_typing(conversation_id: str, on: bool) -> None:
    """Show/hide a typing indicator, if visayabai supports one."""
    print(f"[→ visayabai:{conversation_id}] typing={on}")


# ── opendray side ────────────────────────────────────────────────────
class BridgeAdapter:
    def __init__(self) -> None:
        self._conn: websockets.WebSocketClientProtocol | None = None

    async def run_forever(self) -> None:
        """Connect with exponential backoff; reconnect on drop."""
        backoff = 1
        while True:
            try:
                async with websockets.connect(
                    WS_URL, max_size=256 * 1024, ping_interval=None
                ) as conn:
                    self._conn = conn
                    await self._register(conn)
                    backoff = 1  # reset after a clean register
                    await self._read_loop(conn)
            except Exception as e:  # noqa: BLE001 — reconnect on anything
                print(f"[bridge] connection error: {e!r}")
            finally:
                self._conn = None
            print(f"[bridge] reconnecting in {backoff}s")
            await asyncio.sleep(backoff)
            backoff = min(backoff * 2, 30)

    async def _register(self, conn) -> None:
        await conn.send(
            json.dumps(
                {
                    "type": "register",
                    "platform": "visayabai",
                    "capabilities": CAPABILITIES,
                    "metadata": {"adapter": "visayabai-bridge", "version": "0.1.0"},
                }
            )
        )
        ack = json.loads(await conn.recv())
        if ack.get("type") != "register_ack" or not ack.get("ok"):
            raise RuntimeError(f"register rejected: {ack!r}")
        print("[bridge] registered with opendray")

    async def _read_loop(self, conn) -> None:
        async for raw in conn:
            try:
                frame = json.loads(raw)
            except json.JSONDecodeError:
                continue
            await self._handle_outbound(frame)

    async def _handle_outbound(self, frame: dict[str, Any]) -> None:
        """opendray → visayabai. One handler per outbound frame type."""
        t = frame.get("type")
        conv = frame.get("conversation_id") or ""
        if t == "send":
            await send_to_visayabai_user(conv, frame.get("text", ""))
        elif t == "send_buttons":
            # Render text; flatten button labels as a fallback if visayabai
            # can't show inline keyboards yet.
            text = frame.get("text", "")
            labels = [
                b.get("label", "")
                for row in frame.get("buttons", [])
                for b in row
            ]
            if labels:
                text += "\n\n[" + " | ".join(labels) + "]"
            await send_to_visayabai_user(conv, text)
        elif t == "send_card":
            card = frame.get("card") or {}
            await send_to_visayabai_user(
                conv, card.get("text") or card.get("title") or json.dumps(card)
            )
        elif t == "start_typing":
            await set_typing(conv, True)
        elif t == "stop_typing":
            await set_typing(conv, False)
        elif t == "pong":
            pass
        else:
            print(f"[bridge] unhandled outbound frame: {t}")

    # ── call this from your visayabai message handler ────────────────
    async def forward_user_message(
        self, conversation_id: str, text: str, user_id: str = "", user_name: str = ""
    ) -> None:
        """visayabai → opendray. Call when a user sends a message."""
        if self._conn is None:
            print("[bridge] not connected; dropping message")
            return
        await self._conn.send(
            json.dumps(
                {
                    "type": "message",
                    "conversation_id": conversation_id,  # stable per user/chat
                    "user_id": user_id,
                    "user_name": user_name,
                    "text": text,
                }
            )
        )

    async def forward_button_press(self, conversation_id: str, action: str) -> None:
        """visayabai → opendray. Call when a user taps an inline button."""
        if self._conn is None:
            return
        await self._conn.send(
            json.dumps(
                {
                    "type": "card_action",
                    "conversation_id": conversation_id,
                    "action": action,
                }
            )
        )


async def main() -> None:
    adapter = BridgeAdapter()

    # DEMO: feed one fake user message 3s after connect so you can see the
    # round-trip. Delete this and call adapter.forward_user_message(...)
    # from your real visayabai handler instead.
    async def _demo() -> None:
        await asyncio.sleep(3)
        await adapter.forward_user_message(
            conversation_id="visayabai-user-123",
            text="hello from visayabai — what can you do?",
            user_name="Navid",
        )

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)

    runner = asyncio.create_task(adapter.run_forever())
    demo = asyncio.create_task(_demo())
    await stop.wait()
    runner.cancel()
    demo.cancel()


if __name__ == "__main__":
    asyncio.run(main())
