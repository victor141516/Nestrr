import { onScopeDispose } from "vue";

export interface LatestRequestController {
  begin: () => number;
  invalidate: () => void;
  isLatest: (request: number) => boolean;
}

/**
 * Coordinates async work whose results are only valid for the most recent
 * invocation. Disposing the owning Vue scope invalidates every pending task.
 */
export function useLatestRequest(): LatestRequestController {
  let revision = 0;
  let active = true;

  function begin(): number {
    revision += 1;
    return revision;
  }

  function invalidate(): void {
    revision += 1;
  }

  function isLatest(request: number): boolean {
    return active && request === revision;
  }

  onScopeDispose(() => {
    active = false;
    invalidate();
  });

  return { begin, invalidate, isLatest };
}
