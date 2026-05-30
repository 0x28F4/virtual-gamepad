import {
  axisNames,
  buttonNames,
  type ButtonName,
  type ClientConfig,
  type ControllerState,
  type ServerMessage,
} from "./protocol";
import "./style.css";

const streamSection = requireElement("stream-section");
const streamFrame = requireElement<HTMLIFrameElement>("stream-frame");
const streamStatus = requireElement("stream-status");
const streamSource = requireElement("stream-source");
const tokenInput = requireElement<HTMLInputElement>("token");
const connectButton = requireElement<HTMLButtonElement>("connect");
const connectionStatus = requireElement("connection-status");
const playerElement = requireElement("player");
const gatewayElement = requireElement("gateway");
const gamepadStatus = requireElement("gamepad-status");
const gamepadName = requireElement("gamepad-name");
const packetsElement = requireElement("packets");
const stateElement = requireElement("state");

let socket: WebSocket | null = null;
let assignedPlayer: number | null = null;
let selectedGamepadIndex: number | null = null;
let lastSentState: ControllerState | null = null;
let seq = 0;
let sentPackets = 0;

const shouldAutoConnect = applyRoomTokenFromUrl();
loadClientConfig();

connectButton.addEventListener("click", () => {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close(1000, "User disconnected");
    return;
  }

  connect();
});

window.addEventListener("gamepadconnected", (event) => {
  if (isVirtualOutputGamepad(event.gamepad)) return;

  selectedGamepadIndex = event.gamepad.index;
  gamepadStatus.textContent = "Connected";
  gamepadName.textContent = event.gamepad.id;
});

window.addEventListener("gamepaddisconnected", (event) => {
  if (selectedGamepadIndex === event.gamepad.index) {
    selectedGamepadIndex = null;
    gamepadStatus.textContent = "Press any gamepad button";
    gamepadName.textContent = "None";
  }
});

window.addEventListener("pagehide", () => {
  socket?.close(1000, "Page hidden");
});

if (shouldAutoConnect) {
  connect();
}

requestAnimationFrame(tick);

function applyRoomTokenFromUrl(): boolean {
  const token = new URLSearchParams(window.location.search).get("token");
  if (token) {
    tokenInput.value = token;
    return true;
  }

  return false;
}

async function loadClientConfig(): Promise<void> {
  try {
    const response = await fetch("/config", { headers: { accept: "application/json" } });
    if (!response.ok) throw new Error(`Config request failed: ${response.status}`);

    const config = (await response.json()) as ClientConfig;
    renderStream(config.stream);
  } catch (error) {
    streamStatus.textContent = "Unavailable";
    streamSource.textContent = error instanceof Error ? error.message : "Config request failed";
    streamFrame.hidden = true;
  }
}

function renderStream(stream: ClientConfig["stream"]): void {
  if (!stream.enabled || !stream.playbackUrl) {
    streamStatus.textContent = "Not configured";
    streamSource.textContent = "No stream playback URL";
    streamFrame.removeAttribute("src");
    streamFrame.hidden = true;
    return;
  }

  streamSection.hidden = false;
  streamStatus.textContent = stream.label;
  streamSource.textContent = stream.playbackUrl;
  streamFrame.src = stream.playbackUrl;
  streamFrame.hidden = false;
}

function connect(): void {
  socket?.close();

  const url = new URL("/ws", window.location.href);
  url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("token", tokenInput.value.trim() || "dev");
  gatewayElement.textContent = url.toString();

  socket = new WebSocket(url);
  assignedPlayer = null;
  lastSentState = null;
  connectionStatus.textContent = "Connecting";
  connectButton.textContent = "Disconnect";

  socket.addEventListener("open", () => {
    connectionStatus.textContent = "Connected";
  });

  socket.addEventListener("message", (event) => {
    const message = parseServerMessage(event.data);
    if (!message) return;

    if (message.type === "hello") {
      assignedPlayer = message.player;
      playerElement.textContent = `Player ${message.player}`;
      return;
    }

    connectionStatus.textContent = message.message;
  });

  socket.addEventListener("close", () => {
    assignedPlayer = null;
    connectionStatus.textContent = "Disconnected";
    playerElement.textContent = "Unassigned";
    connectButton.textContent = "Connect";
  });
}

function tick(): void {
  const gamepad = getGamepad();

  if (gamepad) {
    const state = normalizeGamepad(gamepad);
    stateElement.textContent = JSON.stringify(state, null, 2);

    if (
      socket?.readyState === WebSocket.OPEN &&
      assignedPlayer !== null &&
      !statesEqual(state, lastSentState)
    ) {
      socket.send(JSON.stringify({ type: "input", seq: seq++, state }));
      lastSentState = state;
      sentPackets += 1;
      packetsElement.textContent = sentPackets.toString();
    }
  } else {
    gamepadStatus.textContent = "Press any physical gamepad button";
    gamepadName.textContent = "None";
  }

  requestAnimationFrame(tick);
}

function getGamepad(): Gamepad | null {
  const gamepads = navigator.getGamepads();

  if (selectedGamepadIndex !== null) {
    const selectedGamepad = gamepads[selectedGamepadIndex] ?? null;
    if (selectedGamepad && !isVirtualOutputGamepad(selectedGamepad)) return selectedGamepad;

    selectedGamepadIndex = null;
  }

  for (const gamepad of gamepads) {
    if (!gamepad) continue;
    if (isVirtualOutputGamepad(gamepad)) continue;

    selectedGamepadIndex = gamepad.index;
    gamepadStatus.textContent = "Connected";
    gamepadName.textContent = gamepad.id;
    return gamepad;
  }

  return null;
}

function isVirtualOutputGamepad(gamepad: Gamepad): boolean {
  return (
    gamepad.id.includes("Browser Gamepad P") ||
    gamepad.id.includes("Vendor: 28de Product: 11ff")
  );
}

function normalizeGamepad(gamepad: Gamepad): ControllerState {
  const button = (index: number) => normalizeButton(gamepad.buttons[index]);
  const axis = (index: number) => applyDeadzone(gamepad.axes[index] ?? 0);

  return {
    mapping: "standard",
    buttons: {
      a: button(0),
      b: button(1),
      x: button(2),
      y: button(3),
      leftBumper: button(4),
      rightBumper: button(5),
      leftTrigger: button(6),
      rightTrigger: button(7),
      back: button(8),
      start: button(9),
      leftStick: button(10),
      rightStick: button(11),
      dpadUp: button(12),
      dpadDown: button(13),
      dpadLeft: button(14),
      dpadRight: button(15),
      home: button(16),
    },
    axes: {
      leftX: axis(0),
      leftY: axis(1),
      rightX: axis(2),
      rightY: axis(3),
    },
  };
}

function normalizeButton(button: GamepadButton | undefined): ControllerState["buttons"][ButtonName] {
  return {
    pressed: button?.pressed ?? false,
    value: quantizeButtonValue(button?.value ?? 0),
  };
}

function applyDeadzone(value: number): number {
  if (Math.abs(value) < 0.08) return 0;

  return Math.round(value * 50) / 50;
}

function quantizeButtonValue(value: number): number {
  return Math.round(value * 100) / 100;
}

function statesEqual(left: ControllerState, right: ControllerState | null): boolean {
  if (!right) return false;

  for (const name of axisNames) {
    if (left.axes[name] !== right.axes[name]) return false;
  }

  for (const name of buttonNames) {
    if (left.buttons[name].pressed !== right.buttons[name].pressed) return false;
    if (left.buttons[name].value !== right.buttons[name].value) return false;
  }

  return true;
}

function parseServerMessage(data: string): ServerMessage | null {
  try {
    const message = JSON.parse(data) as ServerMessage;
    return message.type === "hello" || message.type === "error" ? message : null;
  } catch {
    return null;
  }
}

function requireElement<T extends HTMLElement = HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (!element) throw new Error(`Missing element #${id}`);
  return element as T;
}
