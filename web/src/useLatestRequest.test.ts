import { defineComponent } from "vue";
import { cleanup, render } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import {
  useLatestRequest,
  type LatestRequestController,
} from "./useLatestRequest";

afterEach(cleanup);

describe("useLatestRequest", () => {
  it("only accepts the newest request", () => {
    let requests: LatestRequestController | undefined;
    render(
      defineComponent({
        setup() {
          requests = useLatestRequest();
          return () => null;
        },
      }),
    );

    const first = requests!.begin();
    const second = requests!.begin();

    expect(requests!.isLatest(first)).toBe(false);
    expect(requests!.isLatest(second)).toBe(true);
  });

  it("invalidates pending requests when the component unmounts", () => {
    let requests: LatestRequestController | undefined;
    const view = render(
      defineComponent({
        setup() {
          requests = useLatestRequest();
          return () => null;
        },
      }),
    );
    const pending = requests!.begin();

    view.unmount();

    expect(requests!.isLatest(pending)).toBe(false);
  });
});
