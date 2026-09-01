import { beforeEach, describe, expect, it } from "vitest";
import {
  DEFAULT_ACCENT_COLOR,
  THEME_STORAGE_KEY,
  applyAccentColor,
  applyThemePreference,
  normalizeAccentColor,
  readThemePreference,
  resolveTheme,
  saveThemePreference,
} from "./theme";

describe("theme preferences", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.style.removeProperty("--accent");
  });

  it("uses the device preference until the user chooses an override", () => {
    expect(readThemePreference()).toBe("system");
    expect(resolveTheme("system", true)).toBe("dark");
    expect(resolveTheme("system", false)).toBe("light");
  });

  it("persists an explicit theme only in browser storage", () => {
    saveThemePreference("light");
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("light");
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("applies an allowed account accent and rejects arbitrary colors", () => {
    expect(applyAccentColor("#38BDF8")).toBe("#38bdf8");
    expect(document.documentElement.style.getPropertyValue("--accent")).toBe(
      "#38bdf8",
    );
    expect(normalizeAccentColor("#000000")).toBe(DEFAULT_ACCENT_COLOR);
  });

  it("can apply a system resolution without storing an override", () => {
    applyThemePreference("system", false);
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBeNull();
  });
});
