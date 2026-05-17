import type { ControllerState } from "../protocol";

export type InputDevice = {
  updateState(player: number, state: ControllerState): void;
  release(player: number): void;
};
