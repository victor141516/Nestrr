import { fireEvent, render, waitFor } from "@testing-library/vue";
import { defineComponent, shallowRef } from "vue";
import { describe, expect, it } from "vitest";
import AppDialog from "./AppDialog.vue";

const DialogHarness = defineComponent({
  components: { AppDialog },
  setup() {
    const open = shallowRef(false);
    return { open };
  },
  template: `
    <button @click="open = true">Open</button>
    <AppDialog
      v-if="open"
      panel-class="test-dialog"
      labelledby="test-title"
      @close="open = false"
    >
      <h2 id="test-title">Dialog</h2>
      <button data-dialog-initial-focus>First</button>
      <button>Last</button>
    </AppDialog>
  `,
});

describe("AppDialog", () => {
  it("contains focus, closes with Escape and restores the opener", async () => {
    const view = render(DialogHarness);
    const opener = view.getByRole("button", { name: "Open" });

    opener.focus();
    await fireEvent.click(opener);
    const dialog = view.getByRole("dialog");
    const first = view.getByRole("button", { name: "First" });
    const last = view.getByRole("button", { name: "Last" });

    await waitFor(() => expect(document.activeElement).toBe(first));
    expect(document.body.style.overflow).toBe("hidden");

    last.focus();
    await fireEvent.keyDown(last, { key: "Tab" });
    expect(document.activeElement).toBe(first);

    await fireEvent.keyDown(dialog, { key: "Escape" });
    await waitFor(() => expect(view.queryByRole("dialog")).toBeNull());
    expect(document.body.style.overflow).toBe("");
    await waitFor(() => expect(document.activeElement).toBe(opener));
  });
});
