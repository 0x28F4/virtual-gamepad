# Deployment

Copy this directory to the target host (for example with `scp -r deploy root@HOST:/root/deploy`).

Copy `.env.example` to `.env` and edit only the values you need to override.
Set `PUBLIC_HOST` to the DNS name or public IP clients use to reach the host.
Set the browser playback URL in `configs/config`.

## Host setup

Install Docker/Docker Compose (for example on Ubuntu):

```bash
apt update
apt install -y docker.io docker-compose-plugin
```

## Compose stack

Create `.env` and start the stack with Docker Compose:

```bash
cp .env.example .env
docker compose up -d
```

The compose file starts:

- `virtual-gamepad`, with `/dev/uinput` access and configurable port `8788` binding.
- `mediamtx`, with RTMP ingest, WebRTC playback, LL-HLS playback, and RTSP playback.
- `cloudflared`, only when `COMPOSE_PROFILES=cloudflare` is set.

The backend always generates a room token on startup and logs the tokenized join
URL:

```bash
docker compose logs virtual-gamepad
```

## Stream playback

The controller UI embeds the MediaMTX WebRTC player through the pluggable stream
config. For the standard MediaMTX path, set `stream.playbackUrl` in
`configs/config` to `http://<PUBLIC_HOST>:8889/live`.

The MediaMTX path publishes to `rtmp://<PUBLIC_HOST>:1935/live` and plays back
through WebRTC at `http://<PUBLIC_HOST>:8889/live`. If the controller UI is
exposed over HTTPS through Cloudflare Tunnel, the stream URL also needs to be
HTTPS or the browser will block it as mixed content.

MediaMTX listens on these host ports:

```text
1935/tcp  RTMP ingest from OBS or FFmpeg
8889/tcp  WebRTC HTTP signaling and player
8189/udp  WebRTC media
8189/tcp  WebRTC media TCP fallback
8888/tcp  LL-HLS playback
8554/tcp  RTSP playback
9997/tcp  MediaMTX API, bound to 127.0.0.1 by Compose
```

If the server IP changes and you do not use DNS, update `PUBLIC_HOST` in `.env`
and recreate the stack. Compose passes it to MediaMTX as
`MTX_WEBRTCADDITIONALHOSTS`, which sets the advertised WebRTC host while keeping
`configs/mediamtx.yml` host-agnostic.

The MediaMTX API is enabled for local session inspection. Because stream auth is
disabled for this experiment, the Compose file binds the API to localhost only.

## Cloudflare Tunnel

Cloudflare Tunnel is optional. If you enable it, create a Cloudflare Tunnel and
route a public hostname to this service URL:

```text
http://virtual-gamepad:8788
```

Set these values in `.env`:

```bash
COMPOSE_PROFILES=cloudflare
CONTROLLER_BIND=127.0.0.1
CLOUDFLARE_TUNNEL_TOKEN=your-token
```

The `cloudflared` sidecar shares Docker Compose's default network with the controller container, so it can reach the gateway by the `virtual-gamepad` service name. The controller port is only bound to `127.0.0.1` on the host; public traffic should enter through Cloudflare Tunnel.

### Controller and stream on one hostname

For a single hostname such as `play.example.com`, configure these tunnel routes
in this order:

```text
play.example.com /live*  -> http://mediamtx:8889
play.example.com         -> http://virtual-gamepad:8788
```

Then set the controller stream config to:

```json
{
  "stream": {
    "enabled": true,
    "playbackUrl": "https://play.example.com/live",
    "label": "Game Stream"
  }
}
```

Keep `PUBLIC_HOST` set to the same public hostname or IP clients use to reach
MediaMTX.

If you do not enable Cloudflare Tunnel, use:

```bash
COMPOSE_PROFILES=
CONTROLLER_BIND=0.0.0.0
```

In that mode, expose `8788/tcp` through your host firewall or put your own
reverse proxy in front of it.

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
