import { buttonNames, type ControllerState } from "../protocol";
import type { InputDevice } from "./device";

export class FakeInputDevice implements InputDevice {
  private readonly lastLogAt = new Map<number, number>();

  updateState(player: number, state: ControllerState): void {
    const now = Date.now();
    const lastLogAt = this.lastLogAt.get(player) ?? 0;

    if (now - lastLogAt < 1000) return;

    this.lastLogAt.set(player, now);

    const pressed = buttonNames.filter((name) => state.buttons[name].pressed);
    console.log(
      `P${player} left=(${state.axes.leftX.toFixed(2)}, ${state.axes.leftY.toFixed(2)}) right=(${state.axes.rightX.toFixed(2)}, ${state.axes.rightY.toFixed(2)}) buttons=${pressed.join(",") || "none"}`,
    );
  }

  release(player: number): void {
    this.lastLogAt.delete(player);
    console.log(`P${player} released`);
  }
}
