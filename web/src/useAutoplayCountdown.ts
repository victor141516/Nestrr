import { onScopeDispose, readonly, shallowRef } from "vue";

export interface AutoplayCountdown {
  seconds: number;
}

export function useAutoplayCountdown() {
  const countdown = shallowRef<AutoplayCountdown | null>(null);
  let timer: number | undefined;

  function cancel(): void {
    if (timer !== undefined) window.clearInterval(timer);
    timer = undefined;
    countdown.value = null;
  }

  function schedule(onComplete: () => void, seconds = 5): void {
    cancel();
    countdown.value = { seconds };
    timer = window.setInterval(() => {
      const remaining = (countdown.value?.seconds ?? 1) - 1;
      if (remaining <= 0) {
        cancel();
        onComplete();
        return;
      }
      countdown.value = { seconds: remaining };
    }, 1000);
  }

  onScopeDispose(cancel);

  return {
    countdown: readonly(countdown),
    schedule,
    cancel,
  };
}
