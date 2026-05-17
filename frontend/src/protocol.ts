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
