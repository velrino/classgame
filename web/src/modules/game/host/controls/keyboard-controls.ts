import { shouldBlockGameKeyEvent } from "@/modules/game/host/controls/block-game-keys";

export type MovementInput = {
  forward: number;
  right: number;
};

const sprintKeys = new Set(["ShiftLeft", "ShiftRight"]);

export function createKeyboardControls(target: Window) {
  const pressed = new Set<string>();
  let interactRequested = false;

  function handleKeyDown(event: KeyboardEvent) {
    if (sprintKeys.has(event.code)) {
      pressed.add(event.code);
      return;
    }

    if (!shouldBlockGameKeyEvent(event)) {
      return;
    }

    event.preventDefault();

    if (event.code === "Enter" || event.code === "Space") {
      if (!event.repeat) {
        interactRequested = true;
      }

      return;
    }

    pressed.add(event.code);
  }

  function handleKeyUp(event: KeyboardEvent) {
    pressed.delete(event.code);
  }

  target.addEventListener("keydown", handleKeyDown);
  target.addEventListener("keyup", handleKeyUp);

  return {
    consumeInteract() {
      const requested = interactRequested;
      interactRequested = false;
      return requested;
    },
    dispose() {
      target.removeEventListener("keydown", handleKeyDown);
      target.removeEventListener("keyup", handleKeyUp);
    },
    getMovement(): MovementInput {
      return {
        forward:
          Number(pressed.has("KeyW") || pressed.has("ArrowUp")) -
          Number(pressed.has("KeyS") || pressed.has("ArrowDown")),
        right:
          Number(pressed.has("KeyD") || pressed.has("ArrowRight")) -
          Number(pressed.has("KeyA") || pressed.has("ArrowLeft")),
      };
    },
    isSprinting() {
      return pressed.has("ShiftLeft") || pressed.has("ShiftRight");
    },
  };
}
