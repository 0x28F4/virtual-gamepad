# Deployment

Copy this directory to the target host (for example with `scp -r deploy root@HOST:/root/deploy`).

Copy `.env.example` to `.env` (already git-ignored) and edit the values (`ROOM_TOKEN`, etc.) before running the commands below.

## Host setup

Install Docker/Docker Compose (for example on Ubuntu):

```bash
apt update
apt install -y docker.io docker-compose-plugin
```

## Controller gateway container

Provide environment variables and start the stack with Docker Compose:

```bash
cp .env.example .env
$EDITOR .env
docker compose up -d
```

The compose file pulls `ghcr.io/0x28f4/virtual-gamepad:latest`, runs it with `/dev/uinput` access, binds port `8788` to localhost for debugging, and starts a Cloudflare Tunnel sidecar.

## Cloudflare Tunnel

Create a Cloudflare Tunnel and route a public hostname to this service URL:

```text
http://virtual-gamepad:8788
```

Set the tunnel token in `.env`:

```bash
CLOUDFLARE_TUNNEL_TOKEN=your-token
```

The `cloudflared` sidecar shares Docker Compose's default network with the controller container, so it can reach the gateway by the `virtual-gamepad` service name. The controller port is only bound to `127.0.0.1` on the host; public traffic should enter through Cloudflare Tunnel.

Attach to the terminal input visualizer:

```bash
docker attach virtual-gamepad
```

Detach without stopping the container with `Ctrl-p` then `Ctrl-q`. `Ctrl+C` stops
the gateway.

If the container was already running before this compose file enabled TTY support,
recreate it first:

```bash
docker compose up -d --force-recreate
```
