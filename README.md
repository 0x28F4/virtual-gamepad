# virtual-gamepad

Browser-based controller sharing with a Python uinput gateway and Vite frontend.
The browser sends gamepad input through WebSockets, and the same page can embed a
low-latency game stream from a pluggable playback source.

## Game stream

The gateway exposes runtime frontend config at `/config`. Configure the current
stream source with environment variables. For the standard MediaMTX path, the
backend derives `GAME_STREAM_URL` from `PUBLIC_HOST`; if `PUBLIC_HOST` is empty,
it tries to discover the host through AWS checkip:

```text
PUBLIC_HOST=example.com
```

Override `GAME_STREAM_URL` only when playback is served from a different host,
path, or protocol. If no playback URL can be derived, the controller UI still
works and the stream panel reports that no stream is configured.

The current experiment uses:

```text
OBS or FFmpeg -> RTMP -> MediaMTX -> WebRTC viewer
```

In deployment, MediaMTX runs in the same Docker Compose project as the controller
gateway. Its config is mounted from `deploy/configs/mediamtx.yml`.

For a deployed MediaMTX server:

```text
RTMP ingest:     rtmp://<PUBLIC_HOST>:1935/live
WebRTC playback: http://<PUBLIC_HOST>:8889/live
LL-HLS fallback: http://<PUBLIC_HOST>:8888/live
```

MediaMTX keeps a static mounted config. Its Control API is enabled internally on
`http://mediamtx:9997`; on startup, the backend patches the global
`webrtcAdditionalHosts` setting from `PUBLIC_HOST`.
The backend always generates a room token and logs the tokenized player join URL
on startup.

WebRTC is the preferred path for interactive play because the observed latency
was sub-second. LL-HLS can remain a compatibility fallback, but it is too latent
for responsive gameplay.

If the controller UI is served over HTTPS, browsers will block an `http://`
stream iframe as mixed content. In that deployment, expose the stream over HTTPS
too, for example through the same public routing layer or an HTTPS-capable stream
frontend.

Container images are published from `main` to GitHub Container Registry:

```text
ghcr.io/0x28f4/virtual-gamepad:latest
```
