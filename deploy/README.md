# Deployment

Copy this directory to the target host (for example with `scp -r deploy root@HOST:/root/deploy`).

Copy `.env.example` to `.env` (already git-ignored), edit the values (`ROOM_TOKEN`, `INPUT_DRIVER`, etc.), and `source` it before running the commands below.

## Host setup

Install Docker/Docker Compose (for example on Ubuntu):

```bash
apt update
apt install -y docker.io docker-compose-plugin
```

## Controller gateway container

Provide environment variables and start the stack with Docker Compose:

```bash
export ROOM_TOKEN=mysecret
export INPUT_DRIVER=uinput
docker compose up -d
```

The compose file pulls `ghcr.io/0x28f4/virtual-gamepad:latest`, runs it with `/dev/uinput` access, and exposes port `8788`.
