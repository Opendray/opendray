#!/usr/bin/env python3
"""
visayabai Telegram ↔ opendray bridge — complete, runnable.

Runs a Telegram bot and relays it to opendray as a `kind=bridge`
channel. opendray owns session routing, reply detection, the control
keyboard, /new provider selection, account switching, and voice;
this process is the thin relay between Telegram and opendray.

Two concurrent loops:
  • bridge loop  — WS to /api/v1/channels/bridge/ws, register, then
                   render opendray's outbound frames into Telegram.
  • telegram loop — long-poll getUpdates; forward user messages and
                    inline-button taps into opendray.

Routing: conversation_id = Telegram chat id (stable per chat), so each
chat gets its own opendray session automatically.

Setup:
  pip install aiohttp websockets
  export TELEGRAM_BOT_TOKEN=123456:ABC...        # @BotFather token
  export OPENDRAY_HOST=opendray.example.com:8770  # host:port, no scheme
  export BRIDGE_TOKEN=...                          # bridge channel token
  export OPENDRAY_TLS=1                            # 0 for ws:// (LAN/dev)
  python visayabai_telegram_bridge.py

opendray side (one-time): Channels → add kind=bridge, set name
"visayabai" + the same BRIDGE_TOKEN, bind/allow a session.
"""

import asyncio
import json
import os
import signal
from typing import Any

import aiohttp           # pip install aiohttp
import websockets        # pip install websockets

TELEGRAM_BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
OPENDRAY_HOST = os.environ["OPENDRAY_HOST"]          # host:port, no scheme
BRIDGE_TOKEN = os.environ["BRIDGE_TOKEN"]
USE_TLS = os.environ.get("OPENDRAY_TLS", "1") != "0"

# Declare what we can render. buttons → opendray sends the control
# keyboard (Stop/Restart/Switch) and the /new provider picker as real
# inline keyboards instead of plain text.
CAPABILITIES = ["text", "typing", "buttons"]

_scheme = "wss" if USE_TLS else "ws"
WS_URL = f"{_scheme}://{OPENDRAY_HOST}/api/v1/channels/bridge/ws?token={BRIDGE_TOKEN}"
TG_API = f"https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}"

# Telegram callback_data is capped at 64 bytes. opendray button values
# (e.g. "nav:/sessions/<id>", control actions) normally fit; we guard
# anyway and keep a map from a short id back to the full value when a
# value is too long.
_cb_overflow: dict[str, str] = {}
_cb_seq = 0


class Telegram:
    """Thin Telegram Bot API client (long-poll + sends)."""

    def __init__(self, session: aiohttp.ClientSession) -> None:
        self._s = session
        self._offset = 0

    async def _call(self, method: str, **params: Any) -> dict[str, Any]:
        async with self._s.post(f"{TG_API}/{method}", json=params) as r:
            return await r.json()

    async def send_text(
        self, chat_id: str, text: str, buttons: list[list[dict]] | None = None
    ) -> None:
        if not text.strip() and not buttons:
            return
        params: dict[str, Any] = {"chat_id": chat_id, "text": text or "…"}
        if buttons:
            params["reply_markup"] = {"inline_keyboard": buttons}
        await self._call("sendMessage", **params)

    async def typing(self, chat_id: str) -> None:
        # Telegram typing auto-expires after ~5s; opendray sends a fresh
        # start_typing before each reply, so we just re-assert it.
        await self._call("sendChatAction", chat_id=chat_id, action="typing")

    async def answer_callback(self, callback_id: str) -> None:
        await self._call("answerCallbackQuery", callback_query_id=callback_id)

    async def poll_forever(self, on_message, on_callback) -> None:
        # allowed_updates keeps the stream to what we handle.
        while True:
            try:
                resp = await self._call(
                    "getUpdates",
                    offset=self._offset,
                    timeout=30,
                    allowed_updates=["message", "callback_query"],
                )
            except Exception as e:  # noqa: BLE001
                print(f"[tg] getUpdates error: {e!r}")
                await asyncio.sleep(3)
                continue
            for upd in resp.get("result", []):
                self._offset = upd["update_id"] + 1
                if "message" in upd and "text" in upd["message"]:
                    m = upd["message"]
                    await on_message(
                        chat_id=str(m["chat"]["id"]),
                        text=m["text"],
                        user_id=str(m["from"].get("id", "")),
                        user_name=m["from"].get("first_name", "")
                        or m["from"].get("username", ""),
                    )
                elif "callback_query" in upd:
                    cq = upd["callback_query"]
                    data = cq.get("data", "")
                    action = _cb_overflow.get(data, data)
                    await on_callback(
                        chat_id=str(cq["message"]["chat"]["id"]),
                        action=action,
                    )
                    await self.answer_callback(cq["id"])


class Bridge:
    """opendray bridge-channel WS client."""

    def __init__(self, telegram: Telegram) -> None:
        self._tg = telegram
        self._conn: websockets.WebSocketClientProtocol | None = None

    # ── opendray → Telegram ──────────────────────────────────────────
    async def _handle_outbound(self, frame: dict[str, Any]) -> None:
        t = frame.get("type")
        chat_id = frame.get("conversation_id") or ""
        if not chat_id:
            return
        if t == "send":
            await self._tg.send_text(chat_id, frame.get("text", ""))
        elif t == "send_buttons":
            await self._tg.send_text(
                chat_id,
                frame.get("text", ""),
                _to_inline_keyboard(frame.get("buttons", [])),
            )
        elif t == "start_typing":
            await self._tg.typing(chat_id)
        elif t in ("stop_typing", "pong"):
            pass  # Telegram typing self-expires; nothing to do
        else:
            print(f"[bridge] unhandled outbound: {t}")

    # ── Telegram → opendray ──────────────────────────────────────────
    async def forward_message(
        self, chat_id: str, text: str, user_id: str = "", user_name: str = ""
    ) -> None:
        await self._send(
            {
                "type": "message",
                "conversation_id": chat_id,
                "user_id": user_id,
                "user_name": user_name,
                "text": text,
            }
        )

    async def forward_callback(self, chat_id: str, action: str) -> None:
        await self._send(
            {"type": "card_action", "conversation_id": chat_id, "action": action}
        )

    async def _send(self, frame: dict[str, Any]) -> None:
        if self._conn is None:
            print("[bridge] not connected; dropping", frame.get("type"))
            return
        await self._conn.send(json.dumps(frame))

    async def run_forever(self) -> None:
        backoff = 1
        while True:
            try:
                async with websockets.connect(
                    WS_URL, max_size=256 * 1024, ping_interval=None
                ) as conn:
                    self._conn = conn
                    await conn.send(
                        json.dumps(
                            {
                                "type": "register",
                                "platform": "visayabai",
                                "capabilities": CAPABILITIES,
                                "metadata": {"adapter": "visayabai-telegram", "version": "0.1.0"},
                            }
                        )
                    )
                    ack = json.loads(await conn.recv())
                    if ack.get("type") != "register_ack" or not ack.get("ok"):
                        raise RuntimeError(f"register rejected: {ack!r}")
                    print("[bridge] registered with opendray")
                    backoff = 1
                    async for raw in conn:
                        try:
                            await self._handle_outbound(json.loads(raw))
                        except json.JSONDecodeError:
                            continue
            except Exception as e:  # noqa: BLE001
                print(f"[bridge] error: {e!r}")
            finally:
                self._conn = None
            print(f"[bridge] reconnecting in {backoff}s")
            await asyncio.sleep(backoff)
            backoff = min(backoff * 2, 30)


def _to_inline_keyboard(rows: list[list[dict]]) -> list[list[dict]]:
    """Map opendray button rows ([[{text,value,style}]]) to Telegram
    inline_keyboard. callback_data is the button value, with a 64-byte
    overflow fallback."""
    global _cb_seq
    out: list[list[dict]] = []
    for row in rows:
        tg_row: list[dict] = []
        for b in row:
            value = str(b.get("value", ""))
            if len(value.encode()) > 64:
                _cb_seq += 1
                key = f"#{_cb_seq}"
                _cb_overflow[key] = value
                value = key
            tg_row.append({"text": b.get("text", "·"), "callback_data": value})
        if tg_row:
            out.append(tg_row)
    return out


async def main() -> None:
    async with aiohttp.ClientSession() as session:
        tg = Telegram(session)
        bridge = Bridge(tg)

        stop = asyncio.Event()
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, stop.set)

        tasks = [
            asyncio.create_task(bridge.run_forever()),
            asyncio.create_task(
                tg.poll_forever(
                    on_message=bridge.forward_message,
                    on_callback=bridge.forward_callback,
                )
            ),
        ]
        await stop.wait()
        for t in tasks:
            t.cancel()


if __name__ == "__main__":
    asyncio.run(main())
