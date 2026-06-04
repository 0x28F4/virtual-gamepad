FROM node:26-bookworm-slim AS builder
WORKDIR /app/frontend
RUN npm install -g pnpm@10.33.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY frontend ./
RUN pnpm build

FROM golang:1.26-bookworm AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 go build -o /out/virtual-gamepad ./cmd/virtual-gamepad

FROM debian:bookworm-slim AS runtime
WORKDIR /app
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
COPY --from=backend /out/virtual-gamepad /usr/local/bin/virtual-gamepad
COPY --from=builder /app/frontend/dist /app/public
ENV HOST=0.0.0.0 \
    PORT=8788 \
    PUBLIC_DIR=/app/public \
    TERM=xterm-256color
EXPOSE 8788
CMD ["virtual-gamepad"]
