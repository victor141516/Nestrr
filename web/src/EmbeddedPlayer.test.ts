import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import EmbeddedPlayer from "./EmbeddedPlayer.vue";

afterEach(cleanup);

const event = {
  source: "plarr",
  version: 1,
  eventId: "event-1",
  sessionId: "session-1",
  type: "play",
  mediaPath: "film.mkv",
  positionSeconds: 1,
  durationSeconds: 10,
  paused: false,
  occurredAt: "2026-08-31T00:00:00.000Z",
};

function mountPlayer() {
  return render(EmbeddedPlayer, {
    props: {
      url: "https://player.example/play",
      origin: "https://player.example",
      labels: {
        upNext: "Up next",
        nextEpisodeIn: "Next episode in",
        playNow: "Play now",
        cancel: "Cancel",
      },
    },
  });
}

describe("EmbeddedPlayer", () => {
  it("replaces the iframe when the media URL changes", async () => {
    const view = mountPlayer();
    const firstFrame = view.container.querySelector("iframe")!;

    await view.rerender({ url: "https://player.example/play-next" });

    const nextFrame = view.container.querySelector("iframe")!;
    expect(nextFrame).not.toBe(firstFrame);
    expect(nextFrame.getAttribute("src")).toBe(
      "https://player.example/play-next",
    );
  });

  it("only emits validated messages from its own frame and origin", () => {
    const view = mountPlayer();
    const frame = view.container.querySelector("iframe")!;

    window.dispatchEvent(
      new MessageEvent("message", {
        data: event,
        origin: "https://attacker.example",
        source: frame.contentWindow,
      }),
    );
    window.dispatchEvent(
      new MessageEvent("message", {
        data: event,
        origin: "https://player.example",
        source: window,
      }),
    );
    window.dispatchEvent(
      new MessageEvent("message", {
        data: event,
        origin: "https://player.example",
        source: frame.contentWindow,
      }),
    );

    expect(view.emitted().playerEvent).toEqual([[event]]);
  });

  it("removes the window listener on unmount", () => {
    const remove = vi.spyOn(window, "removeEventListener");
    const view = mountPlayer();

    view.unmount();

    expect(remove).toHaveBeenCalledWith("message", expect.any(Function));
    remove.mockRestore();
  });

  it("emits the autoplay actions", async () => {
    const view = render(EmbeddedPlayer, {
      props: {
        url: "https://player.example/play",
        origin: "https://player.example",
        autoplaySeconds: 4,
        labels: {
          upNext: "Up next",
          nextEpisodeIn: "Next episode in",
          playNow: "Play now",
          cancel: "Cancel",
        },
      },
    });

    await fireEvent.click(view.getByText("Play now"));
    await fireEvent.click(view.getByText("Cancel"));

    expect(view.emitted().playNext).toHaveLength(1);
    expect(view.emitted().cancelAutoplay).toHaveLength(1);
  });
});
