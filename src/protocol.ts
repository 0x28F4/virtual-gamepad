export const buttonNames = [
  "a",
  "b",
  "x",
  "y",
  "leftBumper",
  "rightBumper",
  "leftTrigger",
  "rightTrigger",
  "back",
  "start",
  "leftStick",
  "rightStick",
  "dpadUp",
  "dpadDown",
  "dpadLeft",
  "dpadRight",
  "home",
] as const;

export const axisNames = ["leftX", "leftY", "rightX", "rightY"] as const;

export type ButtonName = (typeof buttonNames)[number];
export type AxisName = (typeof axisNames)[number];

export type ButtonState = {
  pressed: boolean;
  value: number;
};

export type ControllerState = {
  mapping: "standard";
  buttons: Record<ButtonName, ButtonState>;
  axes: Record<AxisName, number>;
};

export type InputMessage = {
  type: "input";
  seq: number;
  state: ControllerState;
};

export type ServerMessage =
  | { type: "hello"; player: number }
  | { type: "error"; message: string };

export function parseInputMessage(value: unknown): InputMessage | null {
  if (!isRecord(value) || value.type !== "input") return null;
  if (typeof value.seq !== "number" || !Number.isFinite(value.seq)) return null;

  const state = value.state;
  if (!isRecord(state) || !isRecord(state.buttons) || !isRecord(state.axes)) return null;
  if (state.mapping !== "standard") return null;

  for (const name of buttonNames) {
    const button = state.buttons[name];
    if (!isRecord(button)) return null;
    if (typeof button.pressed !== "boolean") return null;
    if (typeof button.value !== "number" || !Number.isFinite(button.value)) return null;
  }

  for (const name of axisNames) {
    if (typeof state.axes[name] !== "number" || !Number.isFinite(state.axes[name])) return null;
  }

  return value as InputMessage;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
