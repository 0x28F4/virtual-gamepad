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

Run the gateway:

```bash
sudo modprobe uinput
sudo uv run python src/main.py
```

Configuration:

```text
HOST=0.0.0.0
PORT=8788
ROOM_TOKEN=dev
MAX_PLAYERS=4
LOG_FILE=backend/logs/gateway.log
PUBLIC_DIR=backend/public
```

The gateway always fans controller input out to Linux uinput, the terminal input
visualizer, and the log file. Logs are written to `backend/logs/gateway.log` by
default.

Start the backend before the emulator so the virtual controllers are detected.
