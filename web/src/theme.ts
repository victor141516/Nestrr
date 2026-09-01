export type ThemePreference = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";

export const THEME_STORAGE_KEY = "nestrr-theme";
export const DEFAULT_ACCENT_COLOR = "#ff624f";

export const ACCENT_COLORS = [
  { value: DEFAULT_ACCENT_COLOR, label: "accentCoral" },
  { value: "#f59e0b", label: "accentAmber" },
  { value: "#84cc16", label: "accentLime" },
  { value: "#14b8a6", label: "accentTeal" },
  { value: "#38bdf8", label: "accentSky" },
  { value: "#818cf8", label: "accentIndigo" },
  { value: "#a78bfa", label: "accentViolet" },
  { value: "#f472b6", label: "accentPink" },
] as const;

export function normalizeAccentColor(value?: string | null) {
  const normalized = String(value || "").toLowerCase();
  return ACCENT_COLORS.some((option) => option.value === normalized)
    ? normalized
    : DEFAULT_ACCENT_COLOR;
}

export function readThemePreference(
  storage: Pick<Storage, "getItem"> = window.localStorage,
): ThemePreference {
  const value = storage.getItem(THEME_STORAGE_KEY);
  return value === "light" || value === "dark" ? value : "system";
}

export function resolveTheme(
  preference: ThemePreference,
  prefersDark: boolean,
): ResolvedTheme {
  return preference === "system"
    ? prefersDark
      ? "dark"
      : "light"
    : preference;
}

function systemPrefersDark() {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? true;
}

export function applyThemePreference(
  preference: ThemePreference,
  prefersDark = systemPrefersDark(),
) {
  const resolved = resolveTheme(preference, prefersDark);
  document.documentElement.dataset.theme = resolved;
  document.documentElement.dataset.themePreference = preference;
  document.documentElement.style.colorScheme = resolved;
  return resolved;
}

export function saveThemePreference(preference: ThemePreference) {
  window.localStorage.setItem(THEME_STORAGE_KEY, preference);
  return applyThemePreference(preference);
}

export function initializeTheme() {
  return applyThemePreference(readThemePreference());
}

export function listenForSystemTheme() {
  const media = window.matchMedia?.("(prefers-color-scheme: dark)");
  if (!media) return () => {};
  const update = () => {
    const preference = readThemePreference();
    if (preference === "system")
      applyThemePreference(preference, media.matches);
  };
  media.addEventListener("change", update);
  return () => media.removeEventListener("change", update);
}

export function applyAccentColor(value?: string | null) {
  const color = normalizeAccentColor(value);
  document.documentElement.style.setProperty("--accent", color);
  return color;
}
