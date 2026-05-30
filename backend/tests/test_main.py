from __future__ import annotations

import logging
import math

import main


def test_load_config_defaults_to_backend_paths(monkeypatch) -> None:
    monkeypatch.delenv("LOG_FILE", raising=False)
    monkeypatch.delenv("PUBLIC_DIR", raising=False)
    monkeypatch.delenv("GAME_STREAM_URL", raising=False)
    monkeypatch.delenv("PUBLIC_HOST", raising=False)
    monkeypatch.setattr(main, "discover_public_host", lambda: "")

    config = main.load_config()

    assert config.max_players == 4
    assert config.log_file == main.BACKEND_DIR / "logs" / "gateway.log"
    assert config.public_dir == main.BACKEND_DIR / "public"
    assert not config.game_stream.enabled
    assert config.room_token
    assert config.public_host == ""


def test_load_config_accepts_pluggable_game_stream(monkeypatch) -> None:
    monkeypatch.setenv("GAME_STREAM_URL", "http://play.example.com:8889/live")

    config = main.load_config()

    assert config.game_stream.enabled
    assert config.game_stream.playback_url == "http://play.example.com:8889/live"
    assert config.game_stream.label == "Game Stream"


def test_load_config_derives_stream_from_public_host(monkeypatch) -> None:
    monkeypatch.delenv("GAME_STREAM_URL", raising=False)
    monkeypatch.setenv("PUBLIC_HOST", "play.example.com")

    config = main.load_config()

    assert config.room_token
    assert config.public_host == "play.example.com"
    assert config.game_stream.playback_url == "http://play.example.com:8889/live"


def test_load_config_discovers_public_host(monkeypatch) -> None:
    monkeypatch.delenv("PUBLIC_HOST", raising=False)
    monkeypatch.delenv("GAME_STREAM_URL", raising=False)
    monkeypatch.setattr(main, "discover_public_host", lambda: "203.0.113.10")

    config = main.load_config()

    assert config.public_host == "203.0.113.10"
    assert config.game_stream.playback_url == "http://203.0.113.10:8889/live"


def test_join_text_uses_public_host(monkeypatch) -> None:
    monkeypatch.setenv("PUBLIC_HOST", "play.example.com")
    config = main.load_config()

    assert main.join_text(config) == f"Join: http://play.example.com:8788?token={config.room_token}"


def test_configure_logging_always_adds_console_handler(monkeypatch, tmp_path) -> None:
    monkeypatch.delenv("PUBLIC_HOST", raising=False)
    monkeypatch.setattr(main, "discover_public_host", lambda: "")
    config = main.load_config()
    config = main.Config(
        host=config.host,
        port=config.port,
        room_token=config.room_token,
        max_players=config.max_players,
        log_file=tmp_path / "gateway.log",
        public_dir=config.public_dir,
        game_stream=config.game_stream,
        public_host=config.public_host,
    )

    main.configure_logging(config)

    root = logging.getLogger()
    assert any(isinstance(handler, logging.StreamHandler) for handler in root.handlers)


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
