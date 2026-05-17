from __future__ import annotations

import asyncio
import json
import logging
import os
from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from aiohttp import WSMsgType, web

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


@dataclass(frozen=True)
class Config:
    host: str
    port: int
    room_token: str
    max_players: int
    input_driver: str
    public_dir: Path


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


class InputDevice(ABC):
    @abstractmethod
    def update_state(self, player: int, state: dict[str, Any]) -> None:
        pass

    @abstractmethod
    def release(self, player: int) -> None:
        pass


class FakeInputDevice(InputDevice):
    def __init__(self) -> None:
        self._last_log_at: dict[int, float] = {}

    def update_state(self, player: int, state: dict[str, Any]) -> None:
        now = asyncio.get_running_loop().time()
        if now - self._last_log_at.get(player, 0) < 1:
            return

        self._last_log_at[player] = now
        axes = state["axes"]
        pressed_buttons = [name for name in BUTTON_NAMES if state["buttons"][name]["pressed"]]
        logging.info(
            "P%s left=(%.2f, %.2f) right=(%.2f, %.2f) buttons=%s",
            player,
            axes["leftX"],
            axes["leftY"],
            axes["rightX"],
            axes["rightY"],
            ",".join(pressed_buttons) or "none",
        )

    def release(self, player: int) -> None:
        self._last_log_at.pop(player, None)
        logging.info("P%s released", player)


class UInputDevice(InputDevice):
    def __init__(self, max_players: int) -> None:
        import uinput

        self._uinput = uinput
        self._devices = {
            player: self._create_device(player) for player in range(1, max_players + 1)
        }
        self._last_states: dict[int, dict[Any, int]] = {}

    def update_state(self, player: int, state: dict[str, Any]) -> None:
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
        self.update_state(player, neutral_state())
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
    return isinstance(value, int | float) and value == value


def create_input_device(config: Config) -> InputDevice:
    if config.input_driver == "fake":
        return FakeInputDevice()
    if config.input_driver == "uinput":
        return UInputDevice(config.max_players)
    raise ValueError(f"Unsupported INPUT_DRIVER={config.input_driver!r}")


async def health(_: web.Request) -> web.Response:
    return web.json_response({"ok": True})


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
    input_device: InputDevice = request.app["input_device"]

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
            input_device.update_state(player, input_message["state"])
    finally:
        input_device.release(player)
        players.release(player)
        logging.info("controller disconnected player=%s", player)

    return ws


def load_config() -> Config:
    return Config(
        host=os.environ.get("HOST", "0.0.0.0"),
        port=int(os.environ.get("PORT", "8788")),
        room_token=os.environ.get("ROOM_TOKEN", "dev"),
        max_players=int(os.environ.get("MAX_PLAYERS", "4")),
        input_driver=os.environ.get("INPUT_DRIVER", "fake"),
        public_dir=Path(os.environ.get("PUBLIC_DIR", Path(__file__).with_name("public"))),
    )


def build_app(config: Config) -> web.Application:
    app = web.Application()
    app["config"] = config
    app["players"] = PlayerSlots(config.max_players)
    app["input_device"] = create_input_device(config)
    app.router.add_get("/health", health)
    app.router.add_get("/ws", websocket_handler)
    app.router.add_get("/", static_handler)
    app.router.add_get("/{path:.*}", static_handler)
    return app


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    config = load_config()
    logging.info(
        "starting gateway host=%s port=%s driver=%s max_players=%s public_dir=%s",
        config.host,
        config.port,
        config.input_driver,
        config.max_players,
        config.public_dir,
    )
    web.run_app(build_app(config), host=config.host, port=config.port)


if __name__ == "__main__":
    main()
