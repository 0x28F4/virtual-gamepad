FROM node:26-bookworm-slim AS builder
WORKDIR /app/frontend
RUN npm install -g pnpm@10.33.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY frontend ./
RUN pnpm build

FROM ghcr.io/astral-sh/uv:python3.14-bookworm-slim AS backend
WORKDIR /app/backend
RUN apt-get update \
  && apt-get install -y --no-install-recommends build-essential linux-libc-dev \
  && rm -rf /var/lib/apt/lists/*
COPY backend/pyproject.toml backend/uv.lock ./
RUN uv sync --locked --no-dev
COPY backend ./
COPY --from=builder /app/frontend/dist ./public
ENV HOST=0.0.0.0 \
    PORT=8788 \
    INPUT_DRIVER=fake
EXPOSE 8788
CMD ["uv", "run", "python", "main.py"]
