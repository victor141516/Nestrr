// @vitest-environment jsdom
import { defineComponent, shallowRef, type ShallowRef } from "vue";
import { render } from "@testing-library/vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useLiveSearchSchedule } from "./useLiveSearchSchedule";

describe("useLiveSearchSchedule", () => {
  afterEach(() => vi.useRealTimers());

  it("throttles local lookups without postponing them and debounces settlement", async () => {
    vi.useFakeTimers();
    const local = vi.fn();
    const settle = vi.fn();
    let query!: ShallowRef<string>;
    const Harness = defineComponent({
      setup() {
        query = shallowRef("");
        useLiveSearchSchedule({
          query,
          localDelayMs: 30,
          settleDelayMs: 2_000,
          onInput: vi.fn(),
          onLocal: local,
          onSettle: settle,
        });
        return () => null;
      },
    });
    render(Harness);

    query.value = "ma";
    await vi.advanceTimersByTimeAsync(20);
    query.value = "mal";
    await vi.advanceTimersByTimeAsync(10);
    expect(local).toHaveBeenCalledWith("mal");
    expect(settle).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1_990);
    expect(settle).toHaveBeenCalledTimes(1);
    expect(settle).toHaveBeenCalledWith("mal");
  });

  it("cancels timers when its component unmounts", async () => {
    vi.useFakeTimers();
    const local = vi.fn();
    const settle = vi.fn();
    let query!: ShallowRef<string>;
    const Harness = defineComponent({
      setup() {
        query = shallowRef("");
        useLiveSearchSchedule({
          query,
          onInput: vi.fn(),
          onLocal: local,
          onSettle: settle,
        });
        return () => null;
      },
    });
    const view = render(Harness);
    query.value = "dune";
    view.unmount();
    await vi.runAllTimersAsync();
    expect(local).not.toHaveBeenCalled();
    expect(settle).not.toHaveBeenCalled();
  });
});
