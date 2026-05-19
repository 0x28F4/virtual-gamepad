from __future__ import annotations

import math

import main


def test_load_config_defaults_to_backend_paths(monkeypatch) -> None:
    monkeypatch.delenv("LOG_FILE", raising=False)
    monkeypatch.delenv("PUBLIC_DIR", raising=False)

    config = main.load_config()

    assert config.max_players == 4
    assert config.log_file == main.BACKEND_DIR / "logs" / "gateway.log"
    assert config.public_dir == main.BACKEND_DIR / "public"


def test_is_number_rejects_non_finite_values() -> None:
    assert main.is_number(0)
    assert main.is_number(1.25)
    assert not main.is_number(math.nan)
    assert not main.is_number(math.inf)
    assert not main.is_number("1")


def test_parse_input_message_accepts_standard_input() -> None:
    message = {"type": "input", "seq": 1, "state": main.neutral_state()}

    assert main.parse_input_message(message) == message


def test_parse_input_message_rejects_invalid_numbers() -> None:
    message = {"type": "input", "seq": 1, "state": main.neutral_state()}
    message["state"]["axes"]["leftX"] = math.inf

    assert main.parse_input_message(message) is None
