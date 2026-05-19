Browser controller input gateway.

The backend serves both:

```text
/       static frontend from backend/public
/ws     controller WebSocket
```

Build the frontend into the backend public directory:

```bash
pnpm build:backend
```

Run in fake mode for frontend development:

```bash
uv run python main.py
```

Run with real Linux virtual gamepads:

```bash
sudo modprobe uinput
sudo INPUT_DRIVER=uinput uv run python main.py
```

Configuration:

```text
HOST=0.0.0.0
PORT=8788
ROOM_TOKEN=dev
MAX_PLAYERS=4
INPUT_DRIVER=fake|uinput
LOG_INPUT=1
LOG_FILE=backend/logs/gateway.log
PUBLIC_DIR=backend/public
```

Fake input starts an interactive terminal visualizer when the backend is attached
to a terminal. It shows all player slots at once with bars for sticks/triggers
and highlighted buttons. Logs are written to `backend/logs/gateway.log` by
default. Set `LOG_INPUT=1` to enable raw input logs.

Start the backend before the emulator so the virtual controllers are detected.
