import { cleanup, render } from "@testing-library/vue";
import { defineComponent } from "vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useAutoplayCountdown } from "./useAutoplayCountdown";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("useAutoplayCountdown", () => {
  it("completes once and clears its timer", () => {
    vi.useFakeTimers();
    const completed = vi.fn();
    let countdown: ReturnType<typeof useAutoplayCountdown> | undefined;
    render(
      defineComponent({
        setup() {
          countdown = useAutoplayCountdown();
          return () => null;
        },
      }),
    );

    countdown!.schedule(completed, 2);
    vi.advanceTimersByTime(2000);

    expect(completed).toHaveBeenCalledOnce();
    expect(countdown!.countdown.value).toBeNull();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("cancels pending completion when its component unmounts", () => {
    vi.useFakeTimers();
    const completed = vi.fn();
    let countdown: ReturnType<typeof useAutoplayCountdown> | undefined;
    const view = render(
      defineComponent({
        setup() {
          countdown = useAutoplayCountdown();
          return () => null;
        },
      }),
    );
    countdown!.schedule(completed);

    view.unmount();
    vi.runAllTimers();

    expect(completed).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
  });
});
