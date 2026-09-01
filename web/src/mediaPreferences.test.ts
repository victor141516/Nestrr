import { describe, expect, it } from "vitest";
import { preferredAudioLanguage } from "./mediaPreferences";

describe("preferredAudioLanguage", () => {
  it("uses the configured user language before a track preference exists", () => {
    expect(preferredAudioLanguage(undefined, "es-ES")).toBe("es");
  });

  it("keeps an explicitly selected track language", () => {
    expect(preferredAudioLanguage("eng", "es-ES")).toBe("eng");
  });
});
