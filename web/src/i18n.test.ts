import { describe, expect, it } from "vitest";
import { tr } from "./i18n";

describe("interface language fallback", () => {
  it("uses Spanish for Spanish locales", () =>
    expect(tr("es-ES", "home")).toBe("Inicio"));
  it("falls back to English for unsupported locales", () =>
    expect(tr("pt-PT", "home")).toBe("Home"));
  it("uses the qBittorrent category name for shows in English", () =>
    expect(tr("en-US", "shows")).toBe("Shows"));
});
