// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/vue";
import { defineComponent, h } from "vue";
import { createMemoryHistory, createRouter, type Router } from "vue-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { clearSession } from "./api";
import { useNestrrApplication } from "./useNestrrApplication";

afterEach(() => {
  cleanup();
  clearSession();
  vi.restoreAllMocks();
});

function testRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:pathMatch(.*)*", component: { render: () => null } }],
  });
}

describe("useNestrrApplication bootstrap", () => {
  it("leaves the boot screen and exposes the setup error when setup fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("", { status: 503 })),
    );
    const router = testRouter();
    await router.push("/");
    await router.isReady();

    render(
      defineComponent({
        setup() {
          const application = useNestrrApplication();
          return () =>
            h(
              "p",
              application.ready.value ? application.error.value : "booting",
            );
        },
      }),
      { global: { plugins: [router] } },
    );

    expect(await screen.findByText("HTTP 503")).toBeTruthy();
  });
});
