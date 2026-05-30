from __future__ import annotations

import asyncio
import json
import logging
import logging.config
import math
import os
import secrets
import sys
import urllib.error
import urllib.parse
import urllib.request
from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from aiohttp import ClientError, ClientSession, ClientTimeout, WSMsgType, web

BACKEND_DIR = Path(__file__).resolve().parent.parent

BUTTON_NAMES = (
    "a",
    "b",
    "x",
    "y",
    "leftBumper",
    "rightBumper",
    "leftTrigger",
    "rightTrigger",
    "back",
    "start",
    "leftStick",
    "rightStick",
    "dpadUp",
    "dpadDown",
    "dpadLeft",
    "dpadRight",
    "home",
)

AXIS_NAMES = ("leftX", "leftY", "rightX", "rightY")
SERVER_SHUTDOWN_TIMEOUT = 1
MEDIAMTX_SYNC_ATTEMPTS = 10
MEDIAMTX_SYNC_DELAY = 0.5


@dataclass(frozen=True)
class GameStreamConfig:
    enabled: bool
    playback_url: str
    label: str


@dataclass(frozen=True)
class Config:
    host: str
    port: int
    room_token: str
    max_players: int
    log_file: Path
    public_dir: Path
    game_stream: GameStreamConfig
    public_host: str


class PlayerSlots:
    def __init__(self, max_players: int) -> None:
        self._max_players = max_players
        self._occupied: set[int] = set()

    def assign(self) -> int | None:
        for player in range(1, self._max_players + 1):
            if player not in self._occupied:
                self._occupied.add(player)
                return player
        return None

    def release(self, player: int) -> None:
        self._occupied.discard(player)


class DeviceController(ABC):
    @abstractmethod
    def update_state(self, player: int, seq: int | float, state: dict[str, Any]) -> None:
        pass

    @abstractmethod
    def release(self, player: int) -> None:
        pass


class TuiDeviceController(DeviceController):
    def __init__(self, max_players: int, join_text: str) -> None:
        self._tui = None
        from gamepad_tui import GamepadTui

        tui = GamepadTui(max_players, join_text)
        if tui.start():
            self._tui = tui
            logging.info("controller TUI started")
        else:
            logging.info("controller TUI disabled because terminal is not interactive")

    def update_state(self, player: int, seq: int | float, state: dict[str, Any]) -> None:
        if self._tui:
            self._tui.update_state(player, state)

    def release(self, player: int) -> None:
        if self._tui:
            self._tui.release(player)
        logging.info("P%s released", player)

    def close(self) -> None:
        if self._tui:
            self._tui.stop()


class UInputDeviceController(DeviceController):
    def __init__(self, max_players: int) -> None:
        import uinput

        self._uinput = uinput
        self._devices = {
            player: self._create_device(player) for player in range(1, max_players + 1)
        }
        self._last_states: dict[int, dict[Any, int]] = {}

    def update_state(self, player: int, seq: int | float, state: dict[str, Any]) -> None:
        events = self._events_from_state(state)
        last_state = self._last_states.get(player, {})
        changed = [(event, value) for event, value in events.items() if last_state.get(event) != value]
        if not changed:
            return

        device = self._devices[player]
        for index, (event, value) in enumerate(changed):
            device.emit(event, value, syn=index == len(changed) - 1)

        self._last_states[player] = events

    def release(self, player: int) -> None:
        self.update_state(player, -1, neutral_state())
        self._last_states.pop(player, None)
        logging.info("P%s released", player)

    def _create_device(self, player: int) -> Any:
        uinput = self._uinput
        events = (
            uinput.BTN_SOUTH,
            uinput.BTN_EAST,
            uinput.BTN_WEST,
            uinput.BTN_NORTH,
            uinput.BTN_TL,
            uinput.BTN_TR,
            uinput.BTN_TL2,
            uinput.BTN_TR2,
            uinput.BTN_SELECT,
            uinput.BTN_START,
            uinput.BTN_THUMBL,
            uinput.BTN_THUMBR,
            uinput.BTN_MODE,
            uinput.ABS_X + (-32768, 32767, 0, 0),
            uinput.ABS_Y + (-32768, 32767, 0, 0),
            uinput.ABS_RX + (-32768, 32767, 0, 0),
            uinput.ABS_RY + (-32768, 32767, 0, 0),
            uinput.ABS_Z + (0, 255, 0, 0),
            uinput.ABS_RZ + (0, 255, 0, 0),
            uinput.ABS_HAT0X + (-1, 1, 0, 0),
            uinput.ABS_HAT0Y + (-1, 1, 0, 0),
        )
        return uinput.Device(events, name=f"Browser Gamepad P{player}")

    def _events_from_state(self, state: dict[str, Any]) -> dict[Any, int]:
        uinput = self._uinput
        buttons = state["buttons"]
        axes = state["axes"]
        return {
            uinput.BTN_SOUTH: pressed(buttons["a"]),
            uinput.BTN_EAST: pressed(buttons["b"]),
            uinput.BTN_WEST: pressed(buttons["x"]),
            uinput.BTN_NORTH: pressed(buttons["y"]),
            uinput.BTN_TL: pressed(buttons["leftBumper"]),
            uinput.BTN_TR: pressed(buttons["rightBumper"]),
            uinput.BTN_TL2: pressed(buttons["leftTrigger"]),
            uinput.BTN_TR2: pressed(buttons["rightTrigger"]),
            uinput.BTN_SELECT: pressed(buttons["back"]),
            uinput.BTN_START: pressed(buttons["start"]),
            uinput.BTN_THUMBL: pressed(buttons["leftStick"]),
            uinput.BTN_THUMBR: pressed(buttons["rightStick"]),
            uinput.BTN_MODE: pressed(buttons["home"]),
            uinput.ABS_X: axis_value(axes["leftX"]),
            uinput.ABS_Y: axis_value(axes["leftY"]),
            uinput.ABS_RX: axis_value(axes["rightX"]),
            uinput.ABS_RY: axis_value(axes["rightY"]),
            uinput.ABS_Z: trigger_value(buttons["leftTrigger"]),
            uinput.ABS_RZ: trigger_value(buttons["rightTrigger"]),
            uinput.ABS_HAT0X: hat_value(buttons["dpadLeft"], buttons["dpadRight"]),
            uinput.ABS_HAT0Y: hat_value(buttons["dpadUp"], buttons["dpadDown"]),
        }


class LogDeviceController(DeviceController):
    def update_state(self, player: int, seq: int | float, state: dict[str, Any]) -> None:
        logging.info(
            "input received player=%s seq=%s %s",
            player,
            seq,
            summarize_state(state),
        )

    def release(self, player: int) -> None:
        pass


class MultiplexDeviceController(DeviceController):
    def __init__(self, controllers: list[DeviceController]) -> None:
        self._controllers = controllers

    def update_state(self, player: int, seq: int | float, state: dict[str, Any]) -> None:
        for controller in self._controllers:
            controller.update_state(player, seq, state)

    def release(self, player: int) -> None:
        for controller in self._controllers:
            controller.release(player)

    def close(self) -> None:
        for controller in self._controllers:
            close = getattr(controller, "close", None)
            if close:
                close()


def pressed(button: dict[str, Any]) -> int:
    return 1 if button["pressed"] else 0


def axis_value(value: float) -> int:
    clamped = max(-1, min(1, value))
    return round(clamped * (32767 if clamped >= 0 else 32768))


def trigger_value(button: dict[str, Any]) -> int:
    return round(max(0, min(1, button["value"])) * 255)


def hat_value(negative: dict[str, Any], positive: dict[str, Any]) -> int:
    return pressed(positive) - pressed(negative)


def neutral_state() -> dict[str, Any]:
    return {
        "mapping": "standard",
        "buttons": {name: {"pressed": False, "value": 0} for name in BUTTON_NAMES},
        "axes": {name: 0 for name in AXIS_NAMES},
    }


def summarize_state(state: dict[str, Any]) -> str:
    axes = state["axes"]
    pressed_buttons = [name for name in BUTTON_NAMES if state["buttons"][name]["pressed"]]
    return (
        f"left=({axes['leftX']:.2f},{axes['leftY']:.2f}) "
        f"right=({axes['rightX']:.2f},{axes['rightY']:.2f}) "
        f"buttons={','.join(pressed_buttons) or 'none'}"
    )


def parse_input_message(value: Any) -> dict[str, Any] | None:
    if not isinstance(value, dict) or value.get("type") != "input":
        return None

    seq = value.get("seq")
    state = value.get("state")
    if not isinstance(seq, int | float):
        return None
    if not isinstance(state, dict) or state.get("mapping") != "standard":
        return None

    buttons = state.get("buttons")
    axes = state.get("axes")
    if not isinstance(buttons, dict) or not isinstance(axes, dict):
        return None

    for name in BUTTON_NAMES:
        button = buttons.get(name)
        if not isinstance(button, dict):
            return None
        if not isinstance(button.get("pressed"), bool):
            return None
        if not is_number(button.get("value")):
            return None

    for name in AXIS_NAMES:
        if not is_number(axes.get(name)):
            return None

    return value


def is_number(value: Any) -> bool:
    return isinstance(value, int | float) and math.isfinite(value)


def create_device_controller(config: Config) -> DeviceController:
    return MultiplexDeviceController(
        [
            UInputDeviceController(config.max_players),
            TuiDeviceController(config.max_players, join_text(config)),
            LogDeviceController(),
        ]
    )


async def health(_: web.Request) -> web.Response:
    return web.json_response({"ok": True})


async def client_config(request: web.Request) -> web.Response:
    config: Config = request.app["config"]
    stream = config.game_stream
    return web.json_response(
        {
            "stream": {
                "enabled": stream.enabled,
                "playbackUrl": stream.playback_url,
                "label": stream.label,
            }
        }
    )


async def static_handler(request: web.Request) -> web.StreamResponse:
    config: Config = request.app["config"]
    public_dir = config.public_dir
    index_path = public_dir / "index.html"
    requested_path = request.match_info.get("path", "")

    if not index_path.exists():
        raise web.HTTPNotFound(text="Frontend has not been built into backend/public")

    if requested_path:
        candidate = (public_dir / requested_path).resolve()
        if public_dir.resolve() in (candidate, *candidate.parents) and candidate.is_file():
            return web.FileResponse(candidate)

    return web.FileResponse(index_path)


async def websocket_handler(request: web.Request) -> web.WebSocketResponse:
    config: Config = request.app["config"]
    players: PlayerSlots = request.app["players"]
    device_controller: DeviceController = request.app["device_controller"]

    ws = web.WebSocketResponse()
    await ws.prepare(request)

    if request.query.get("token") != config.room_token:
        await ws.send_json({"type": "error", "message": "Invalid room token"})
        await ws.close(code=1008, message=b"Invalid room token")
        return ws

    player = players.assign()
    if player is None:
        await ws.send_json({"type": "error", "message": "No player slots available"})
        await ws.close(code=1013, message=b"No player slots available")
        return ws

    latest_seq = -1
    logging.info("controller connected player=%s", player)
    await ws.send_json({"type": "hello", "player": player})

    try:
        async for message in ws:
            if message.type != WSMsgType.TEXT:
                continue

            try:
                parsed = json.loads(message.data)
            except json.JSONDecodeError:
                continue

            input_message = parse_input_message(parsed)
            if input_message is None or input_message["seq"] <= latest_seq:
                continue

            latest_seq = input_message["seq"]
            device_controller.update_state(player, input_message["seq"], input_message["state"])
    finally:
        device_controller.release(player)
        players.release(player)
        logging.info("controller disconnected player=%s", player)

    return ws


def load_config() -> Config:
    public_host = os.environ.get("PUBLIC_HOST", "").strip()
    if not public_host:
        public_host = discover_public_host()

    room_token = secrets.token_urlsafe(18)

    port = int(os.environ.get("PORT", "8788"))
    stream_url = os.environ.get("GAME_STREAM_URL", "").strip()
    if not stream_url and public_host:
        stream_url = f"http://{public_host}:8889/live"

    return Config(
        host=os.environ.get("HOST", "0.0.0.0"),
        port=port,
        room_token=room_token,
        max_players=int(os.environ.get("MAX_PLAYERS", "4")),
        log_file=Path(os.environ.get("LOG_FILE", BACKEND_DIR / "logs" / "gateway.log")),
        public_dir=Path(os.environ.get("PUBLIC_DIR", BACKEND_DIR / "public")),
        game_stream=GameStreamConfig(
            enabled=bool(stream_url),
            playback_url=stream_url,
            label="Game Stream",
        ),
        public_host=public_host,
    )


def truthy(value: str) -> bool:
    return value.strip().lower() in {"1", "true", "yes", "on"}


def discover_public_host() -> str:
    try:
        with urllib.request.urlopen("https://checkip.amazonaws.com", timeout=5) as response:
            return response.read().decode("utf-8").strip()
    except (OSError, urllib.error.URLError) as exc:
        logging.info("public host auto-discovery failed: %s", exc)
        return ""


def add_query_param(url: str, name: str, value: str) -> str:
    parsed = urllib.parse.urlsplit(url)
    params = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
    params.append((name, value))
    return urllib.parse.urlunsplit(parsed._replace(query=urllib.parse.urlencode(params)))


def join_text(config: Config) -> str:
    if config.public_host:
        return "Join: " + add_query_param(
            f"http://{config.public_host}:{config.port}",
            "token",
            config.room_token,
        )

    return f"Join query: ?token={urllib.parse.quote(config.room_token)}"


def configure_logging(config: Config) -> None:
    config.log_file.parent.mkdir(parents=True, exist_ok=True)
    handlers: dict[str, dict[str, Any]] = {
        "file": {
            "class": "logging.handlers.RotatingFileHandler",
            "filename": str(config.log_file),
            "maxBytes": 1_000_000,
            "backupCount": 3,
            "formatter": "default",
            "encoding": "utf-8",
        }
    }
    handlers["console"] = {
        "class": "logging.StreamHandler",
        "stream": "ext://sys.stdout",
        "formatter": "default",
    }
    root_handlers = ["file", "console"]

    logging.config.dictConfig(
        {
            "version": 1,
            "disable_existing_loggers": False,
            "formatters": {
                "default": {
                    "format": "%(asctime)s %(levelname)s %(name)s %(message)s",
                    "datefmt": "%Y-%m-%d %H:%M:%S",
                }
            },
            "handlers": handlers,
            "root": {
                "level": "DEBUG",
                "handlers": root_handlers,
            },
        }
    )


async def cleanup_device_controller(app: web.Application) -> None:
    close = getattr(app["device_controller"], "close", None)
    if close:
        close()


async def sync_mediamtx_config(app: web.Application) -> None:
    config: Config = app["config"]
    if not config.public_host:
        return

    url = "http://mediamtx:9997/v3/config/global/patch"
    payload = {"webrtcAdditionalHosts": [config.public_host]}
    timeout = ClientTimeout(total=2)

    for attempt in range(1, MEDIAMTX_SYNC_ATTEMPTS + 1):
        try:
            async with ClientSession(timeout=timeout) as session:
                async with session.patch(url, json=payload) as response:
                    if response.status == 200:
                        logging.info(
                            "synced MediaMTX webrtcAdditionalHosts=%s",
                            config.public_host,
                        )
                        return

                    text = await response.text()
                    logging.warning(
                        "MediaMTX config sync failed status=%s body=%s",
                        response.status,
                        text,
                    )
        except ClientError as exc:
            logging.info(
                "MediaMTX config sync attempt %s/%s failed: %s",
                attempt,
                MEDIAMTX_SYNC_ATTEMPTS,
                exc,
            )

        await asyncio.sleep(MEDIAMTX_SYNC_DELAY)

    logging.warning("MediaMTX config sync did not complete")


def build_app(config: Config) -> web.Application:
    app = web.Application()
    app["config"] = config
    app["players"] = PlayerSlots(config.max_players)
    app["device_controller"] = create_device_controller(config)
    app.router.add_get("/health", health)
    app.router.add_get("/config", client_config)
    app.router.add_get("/ws", websocket_handler)
    app.router.add_get("/", static_handler)
    app.router.add_get("/{path:.*}", static_handler)
    app.on_startup.append(sync_mediamtx_config)
    app.on_cleanup.append(cleanup_device_controller)
    return app


def main() -> None:
    config = load_config()
    configure_logging(config)
    logging.info(
        "starting gateway host=%s port=%s max_players=%s log_file=%s public_dir=%s public_host=%s stream_url=%s",
        config.host,
        config.port,
        config.max_players,
        config.log_file,
        config.public_dir,
        config.public_host or "unknown",
        config.game_stream.playback_url or "none",
    )
    logging.info("generated room token=%s", config.room_token)
    logging.info("%s", join_text(config))

    try:
        web.run_app(
            build_app(config),
            host=config.host,
            port=config.port,
            handler_cancellation=True,
            shutdown_timeout=SERVER_SHUTDOWN_TIMEOUT,
        )
    except KeyboardInterrupt:
        logging.info("shutdown requested")


if __name__ == "__main__":
    main()
