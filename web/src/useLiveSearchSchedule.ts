import { onBeforeUnmount, watch, type Ref } from "vue";

type UseLiveSearchScheduleOptions = {
  query: Ref<string>;
  localDelayMs?: number;
  settleDelayMs?: number;
  minimumLocalLength?: number;
  onInput: (query: string) => void;
  onLocal: (query: string) => void | Promise<void>;
  onSettle: (query: string) => void | Promise<void>;
};

export function useLiveSearchSchedule(options: UseLiveSearchScheduleOptions) {
  const localDelayMs = options.localDelayMs ?? 80;
  const settleDelayMs = options.settleDelayMs ?? 2_000;
  const minimumLocalLength = options.minimumLocalLength ?? 3;
  let localTimer: number | null = null;
  let settleTimer: number | null = null;
  let syncing = false;

  function cancel() {
    if (localTimer !== null) window.clearTimeout(localTimer);
    if (settleTimer !== null) window.clearTimeout(settleTimer);
    localTimer = null;
    settleTimer = null;
  }

  function schedule(query: string) {
    const trimmed = query.trim();
    options.onInput(trimmed);
    if (settleTimer !== null) window.clearTimeout(settleTimer);
    if (!trimmed) {
      if (localTimer !== null) window.clearTimeout(localTimer);
      localTimer = null;
      settleTimer = window.setTimeout(() => {
        settleTimer = null;
        void options.onSettle("");
      }, settleDelayMs);
      return;
    }
    // This is a fixed-window trailing throttle: later keystrokes update the
    // value read by the callback without restarting the short timer.
    if (localTimer === null) {
      localTimer = window.setTimeout(() => {
        localTimer = null;
        const current = options.query.value.trim();
        if (current.length >= minimumLocalLength) void options.onLocal(current);
      }, localDelayMs);
    }
    settleTimer = window.setTimeout(() => {
      settleTimer = null;
      void options.onSettle(options.query.value.trim());
    }, settleDelayMs);
  }

  watch(
    options.query,
    (value) => {
      if (!syncing) schedule(value);
    },
    { flush: "sync" },
  );

  function syncQuery(value: string) {
    cancel();
    syncing = true;
    options.query.value = value;
    syncing = false;
  }

  async function settle() {
    const current = options.query.value.trim();
    cancel();
    await options.onSettle(current);
  }

  onBeforeUnmount(cancel);

  return { cancel, settle, syncQuery };
}
