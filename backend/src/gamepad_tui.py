from __future__ import annotations

import atexit
import curses
import sys
import threading
import time
from typing import Any


BUTTON_GROUPS = (
    ("face", ("a", "b", "x", "y")),
    ("shoulder", ("leftBumper", "rightBumper", "leftStick", "rightStick")),
    ("menu", ("back", "start", "home")),
    ("dpad", ("dpadUp", "dpadDown", "dpadLeft", "dpadRight")),
)

BUTTON_LABELS = {
    "a": "A",
    "b": "B",
    "x": "X",
    "y": "Y",
    "leftBumper": "LB",
    "rightBumper": "RB",
    "leftStick": "LS",
    "rightStick": "RS",
    "back": "BACK",
    "start": "START",
    "home": "HOME",
    "dpadUp": "UP",
    "dpadDown": "DOWN",
    "dpadLeft": "LEFT",
    "dpadRight": "RIGHT",
}


class GamepadTui:
    def __init__(self, max_players: int) -> None:
        self._max_players = max_players
        self._states: dict[int, dict[str, Any] | None] = {
            player: None for player in range(1, max_players + 1)
        }
        self._packets: dict[int, int] = {player: 0 for player in range(1, max_players + 1)}
        self._last_seen: dict[int, float] = {player: 0 for player in range(1, max_players + 1)}
        self._lock = threading.Lock()
        self._running = threading.Event()
        self._thread: threading.Thread | None = None

    def start(self) -> bool:
        if not sys.stdin.isatty() or not sys.stdout.isatty():
            return False

        self._running.set()
        self._thread = threading.Thread(target=self._run, name="gamepad-tui", daemon=True)
        self._thread.start()
        atexit.register(self.stop)
        return True

    def stop(self) -> None:
        self._running.clear()
        if self._thread and self._thread is not threading.current_thread():
            self._thread.join(timeout=0.2)

    def update_state(self, player: int, state: dict[str, Any]) -> None:
        with self._lock:
            self._states[player] = state
            self._packets[player] += 1
            self._last_seen[player] = time.monotonic()

    def release(self, player: int) -> None:
        with self._lock:
            self._states[player] = None

    def _run(self) -> None:
        try:
            curses.wrapper(self._loop)
        finally:
            self._running.clear()

    def _loop(self, stdscr: curses.window) -> None:
        curses.curs_set(0)
        curses.use_default_colors()
        curses.init_pair(1, curses.COLOR_GREEN, -1)
        curses.init_pair(2, curses.COLOR_GREEN, -1)
        curses.init_pair(3, curses.COLOR_CYAN, -1)
        curses.init_pair(4, curses.COLOR_YELLOW, -1)
        curses.init_pair(5, curses.COLOR_YELLOW, -1)
        curses.init_pair(6, curses.COLOR_RED, -1)
        curses.init_pair(7, curses.COLOR_WHITE, -1)
        stdscr.nodelay(True)
        stdscr.keypad(True)

        while self._running.is_set():
            self._render(stdscr)
            self._running.wait(1 / 20)

    def _render(self, stdscr: curses.window) -> None:
        height, width = stdscr.getmaxyx()
        stdscr.erase()

        if height < 10 or width < 92:
            self._add(stdscr, 0, 0, "Needs at least 92 columns", curses.color_pair(6))
            stdscr.refresh()
            return

        now = time.monotonic()
        self._add(stdscr, 0, 1, "INPUT", curses.color_pair(1) | curses.A_BOLD)

        player_height = 7
        visible_players = max(1, min(self._max_players, (height - 2) // player_height))
        for index, player in enumerate(range(1, visible_players + 1)):
            self._draw_player(stdscr, 2 + index * player_height, width, player, now)

        if visible_players < self._max_players:
            self._add(
                stdscr,
                height - 1,
                1,
                f"Terminal too short: showing {visible_players}/{self._max_players} players",
                curses.color_pair(4),
            )

        stdscr.refresh()

    def _draw_player(self, stdscr: curses.window, y: int, width: int, player: int, now: float) -> None:
        state, packets, last_seen = self._snapshot(player)
        active = now - last_seen < 1.5
        status = "LIVE" if active else "idle"
        last_text = "never" if last_seen == 0 else f"{now - last_seen:4.1f}s"
        title_attr = curses.color_pair(2) | curses.A_BOLD if active else curses.color_pair(7) | curses.A_BOLD

        self._add(stdscr, y, 1, "-" * (width - 2), curses.A_DIM)
        self._add(stdscr, y + 1, 1, f"P{player}", title_attr)
        self._add(stdscr, y + 1, 5, f"{status:>4}", title_attr if active else curses.A_DIM)
        self._add(stdscr, y + 1, 12, f"pkt {packets:<6} last {last_text:<7}", curses.color_pair(7))

        if state is None:
            self._add(stdscr, y + 3, 5, "waiting for input", curses.color_pair(4))
            return

        self._draw_axis_bar(stdscr, y + 2, 5, "LX", axis(state, "leftX"))
        self._draw_axis_bar(stdscr, y + 2, 38, "LY", axis(state, "leftY"))
        self._draw_trigger_bar(stdscr, y + 2, 71, "LT", value(state, "leftTrigger"))
        self._draw_axis_bar(stdscr, y + 3, 5, "RX", axis(state, "rightX"))
        self._draw_axis_bar(stdscr, y + 3, 38, "RY", axis(state, "rightY"))
        self._draw_trigger_bar(stdscr, y + 3, 71, "RT", value(state, "rightTrigger"))
        self._draw_buttons(stdscr, y + 5, 5, state)

    def _snapshot(self, player: int) -> tuple[dict[str, Any] | None, int, float]:
        with self._lock:
            return self._states[player], self._packets[player], self._last_seen[player]

    def _draw_axis_bar(self, stdscr: curses.window, y: int, x: int, label: str, amount: float) -> None:
        amount = clamp(amount)
        width = 21
        center = width // 2
        marker = center + round(amount * center)
        cells = ["."] * width
        cells[center] = "|"
        cells[marker] = "#"
        attr = curses.color_pair(2) | curses.A_BOLD if abs(amount) >= 0.08 else curses.A_DIM
        self._add(stdscr, y, x, f"{label} ", curses.color_pair(3) | curses.A_BOLD)
        self._add(stdscr, y, x + 3, "[", curses.color_pair(7))
        self._add(stdscr, y, x + 4, "".join(cells), attr)
        self._add(stdscr, y, x + 4 + width, f"] {amount:+.2f}", curses.color_pair(7))

    def _draw_trigger_bar(self, stdscr: curses.window, y: int, x: int, label: str, amount: float) -> None:
        amount = clamp(amount, 0, 1)
        width = 14
        filled = round(amount * width)
        attr = curses.color_pair(5) | curses.A_BOLD if amount > 0 else curses.A_DIM
        self._add(stdscr, y, x, f"{label} ", curses.color_pair(4) | curses.A_BOLD)
        self._add(stdscr, y, x + 3, "[", curses.color_pair(7))
        self._add(stdscr, y, x + 4, "#" * filled + "." * (width - filled), attr)
        self._add(stdscr, y, x + 4 + width, f"] {amount:.2f}", curses.color_pair(7))

    def _draw_buttons(self, stdscr: curses.window, y: int, x: int, state: dict[str, Any]) -> None:
        cursor = x
        for group, names in BUTTON_GROUPS:
            self._add(stdscr, y, cursor, f"{group}:", curses.color_pair(3) | curses.A_BOLD)
            cursor += len(group) + 2
            for name in names:
                label = BUTTON_LABELS[name]
                is_pressed = pressed(state, name)
                attr = curses.color_pair(2) | curses.A_BOLD if is_pressed else curses.A_DIM
                text = f" {label} " if is_pressed else f" {label.lower()} "
                self._add(stdscr, y, cursor, text, attr)
                cursor += len(text)
            cursor += 2

    def _add(self, stdscr: curses.window, y: int, x: int, text: str, attr: int = 0) -> None:
        try:
            stdscr.addstr(y, x, text, attr)
        except curses.error:
            pass


def pressed(state: dict[str, Any], name: str) -> bool:
    return bool(state["buttons"][name]["pressed"])


def value(state: dict[str, Any], name: str) -> float:
    return float(state["buttons"][name]["value"])


def axis(state: dict[str, Any], name: str) -> float:
    return float(state["axes"][name])


def clamp(value: float, minimum: float = -1, maximum: float = 1) -> float:
    return max(minimum, min(maximum, value))
