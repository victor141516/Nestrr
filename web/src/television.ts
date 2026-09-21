// The hosted webOS launcher opts in outside the hash router, so navigation
// keeps the mode without adding a parameter to every RouterLink.
export function televisionEnabled(search = window.location.search): boolean {
  return new URLSearchParams(search).get("tv") === "1";
}

export function televisionPlayerURL(url: string): string {
  const target = new URL(url);
  const [route, query = ""] = target.hash.split("?");
  const params = new URLSearchParams(query);
  params.set("tv", "1");
  target.hash = `${route}?${params}`;
  return target.href;
}

export type Direction = "ArrowLeft" | "ArrowRight" | "ArrowUp" | "ArrowDown";

export function remoteKey(
  event: Pick<KeyboardEvent, "key" | "keyCode">,
): string {
  if (event.keyCode === 461) return "Back";
  return event.key && event.key !== "Unidentified"
    ? event.key
    : {
        13: "Enter",
        37: "ArrowLeft",
        38: "ArrowUp",
        39: "ArrowRight",
        40: "ArrowDown",
      }[event.keyCode] || "";
}

// Prefer the same row/column, then the closest candidate in that direction.
export function directionalTarget(
  from: DOMRect,
  candidates: { element: HTMLElement; rect: DOMRect }[],
  direction: Direction,
): HTMLElement | undefined {
  const horizontal = direction === "ArrowLeft" || direction === "ArrowRight";
  const sign = direction === "ArrowLeft" || direction === "ArrowUp" ? -1 : 1;
  const axis = (r: DOMRect) =>
    horizontal ? r.left + r.width / 2 : r.top + r.height / 2;
  const cross = (r: DOMRect) =>
    horizontal ? r.top + r.height / 2 : r.left + r.width / 2;
  let best: HTMLElement | undefined;
  let bestScore = Infinity;
  for (const { element, rect } of candidates) {
    const forward = (axis(rect) - axis(from)) * sign;
    if (forward < 1) continue;
    const overlap = horizontal
      ? rect.bottom > from.top && rect.top < from.bottom
      : rect.right > from.left && rect.left < from.right;
    const score =
      forward +
      Math.abs(cross(rect) - cross(from)) * 3 +
      (overlap ? 0 : 10_000);
    if (score < bestScore) {
      bestScore = score;
      best = element;
    }
  }
  return best;
}
