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
PUBLIC_DIR=backend/public
```

Start the backend before the emulator so the virtual controllers are detected.
