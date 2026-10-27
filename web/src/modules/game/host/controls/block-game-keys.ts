export const blockedGameKeyCodes = new Set([
  "ArrowDown",
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "Enter",
  "KeyA",
  "KeyD",
  "KeyS",
  "KeyW",
  "Space",
]);

export function isInteractiveTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return (
    target.isContentEditable ||
    Boolean(target.closest("button, a[href], [role='button'], [role='dialog']")) ||
    target instanceof HTMLInputElement ||
    target instanceof HTMLSelectElement ||
    target instanceof HTMLTextAreaElement
  );
}

export function shouldBlockGameKeyEvent(event: KeyboardEvent) {
  return (
    !event.ctrlKey &&
    !event.metaKey &&
    !event.altKey &&
    blockedGameKeyCodes.has(event.code) &&
    !isInteractiveTarget(event.target)
  );
}
