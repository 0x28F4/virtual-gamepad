import fastifyStatic from "@fastify/static";
import websocket from "@fastify/websocket";
import Fastify from "fastify";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { FakeInputDevice } from "./input/fake";
import { PlayerSlots } from "./players";
import { parseInputMessage, type ServerMessage } from "./protocol";

const port = Number.parseInt(process.env.PORT ?? "8788", 10);
const roomToken = process.env.ROOM_TOKEN ?? "dev";
const maxPlayers = Number.parseInt(process.env.MAX_PLAYERS ?? "4", 10);

const app = Fastify({ logger: true });
const players = new PlayerSlots(maxPlayers);
const inputDevice = new FakeInputDevice();

await app.register(websocket);

const distPath = join(process.cwd(), "dist");
if (existsSync(distPath)) {
  await app.register(fastifyStatic, {
    root: distPath,
    prefix: "/",
  });
}

app.get("/health", async () => ({ ok: true }));

app.get("/ws", { websocket: true }, (socket, request) => {
  const url = new URL(request.url, `http://${request.headers.host ?? "localhost"}`);
  const token = url.searchParams.get("token");

  if (token !== roomToken) {
    send(socket, { type: "error", message: "Invalid room token" });
    socket.close(1008, "Invalid room token");
    return;
  }

  const player = players.assign();
  if (player === null) {
    send(socket, { type: "error", message: "No player slots available" });
    socket.close(1013, "No player slots available");
    return;
  }

  let latestSeq = -1;
  app.log.info({ player }, "controller connected");
  send(socket, { type: "hello", player });

  socket.on("message", (raw: { toString(): string }) => {
    let parsed: unknown;

    try {
      parsed = JSON.parse(raw.toString());
    } catch {
      return;
    }

    const message = parseInputMessage(parsed);
    if (!message || message.seq <= latestSeq) return;

    latestSeq = message.seq;
    inputDevice.updateState(player, message.state);
  });

  socket.on("close", () => {
    inputDevice.release(player);
    players.release(player);
    app.log.info({ player }, "controller disconnected");
  });
});

await app.listen({ host: "0.0.0.0", port });

function send(socket: { send(data: string): void }, message: ServerMessage): void {
  socket.send(JSON.stringify(message));
}
