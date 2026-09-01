import { afterEach, describe, expect, it, vi } from "vitest";
import { clearSession, streamAPI } from "./api";

afterEach(() => {
  clearSession();
  vi.unstubAllGlobals();
});

describe("streamAPI", () => {
  it("decodes newline-delimited JSON across network chunks", async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode('{"status":"database_'));
        controller.enqueue(
          encoder.encode('search"}\n{"status":"saved","added":true}\n'),
        );
        controller.close();
      },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(body, { status: 200 })),
    );
    const events: unknown[] = [];

    await streamAPI("/lists/7/bulk-import", { method: "POST" }, (event) =>
      events.push(event),
    );

    expect(events).toEqual([
      { status: "database_search" },
      { status: "saved", added: true },
    ]);
  });
});
