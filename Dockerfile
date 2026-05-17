FROM node:26-bookworm-slim AS frontend
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.33.0 --activate
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend/package.json ./frontend/package.json
RUN pnpm install --frozen-lockfile
COPY frontend ./frontend
RUN pnpm --dir frontend build

FROM ghcr.io/astral-sh/uv:python3.14-bookworm-slim AS backend
WORKDIR /app/backend
RUN apt-get update \
  && apt-get install -y --no-install-recommends build-essential linux-libc-dev \
  && rm -rf /var/lib/apt/lists/*
COPY backend/pyproject.toml backend/uv.lock ./
RUN uv sync --locked --no-dev
COPY backend ./
COPY --from=frontend /app/frontend/dist ./public
ENV HOST=0.0.0.0 \
    PORT=8788 \
    INPUT_DRIVER=fake
EXPOSE 8788
CMD ["uv", "run", "python", "main.py"]
