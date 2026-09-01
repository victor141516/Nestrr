import { describe, expect, it } from "vitest";
import { isPlarrEvent } from "./playerProtocol";

const validEvent = {
  source: "plarr",
  version: 1,
  eventId: "event-1",
  sessionId: "session-1",
  type: "heartbeat",
  mediaPath: "film.mkv",
  positionSeconds: 12,
  durationSeconds: 120,
  paused: false,
  occurredAt: "2026-08-31T00:00:00.000Z",
};

describe("isPlarrEvent", () => {
  it("accepts a complete v1 event", () => {
    expect(isPlarrEvent(validEvent)).toBe(true);
  });

  it("rejects unknown events and non-finite playback positions", () => {
    expect(isPlarrEvent({ ...validEvent, type: "execute" })).toBe(false);
    expect(isPlarrEvent({ ...validEvent, positionSeconds: Number.NaN })).toBe(
      false,
    );
  });
});
