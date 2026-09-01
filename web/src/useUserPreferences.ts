import { onBeforeUnmount, onMounted, ref, watch, type Ref } from "vue";
import { api, session, type User } from "./api";
import type { TranslationKey } from "./i18n";
import {
  DEFAULT_ACCENT_COLOR,
  applyAccentColor,
  listenForSystemTheme,
  readThemePreference,
  saveThemePreference,
  type ThemePreference,
} from "./theme";

type Options = {
  t: (key: TranslationKey) => string;
  flash: (message: string) => void;
  loadRoute: () => Promise<void>;
  error: Ref<string>;
};

export function useUserPreferences(options: Options) {
  const { t, flash, loadRoute, error } = options;
  const themePreference = ref<ThemePreference>(readThemePreference());
  const accentSaving = ref(false);
  const accentColor = ref(DEFAULT_ACCENT_COLOR);
  let stopSystemTheme: undefined | (() => void);

  async function saveSettings(event: Event) {
    const form = new FormData(event.target as HTMLFormElement);
    const user = await api<User>("/me/settings", {
      method: "PATCH",
      body: JSON.stringify({ language: form.get("language") }),
    });
    session.user = user;
    flash(t("save"));
    await loadRoute();
  }

  function selectTheme(preference: ThemePreference) {
    themePreference.value = preference;
    saveThemePreference(preference);
  }

  async function selectAccentColor(color: string) {
    if (accentSaving.value || session.user?.accentColor === color) return;
    const previous = session.user?.accentColor || DEFAULT_ACCENT_COLOR;
    accentColor.value = color;
    applyAccentColor(color);
    accentSaving.value = true;
    try {
      session.user = await api<User>("/me/settings", {
        method: "PATCH",
        body: JSON.stringify({ accentColor: color }),
      });
      flash(t("save"));
    } catch (reason: any) {
      accentColor.value = previous;
      applyAccentColor(previous);
      error.value = reason.message;
    } finally {
      accentSaving.value = false;
    }
  }

  watch(
    () => session.user?.accentColor,
    (value) => {
      accentColor.value = applyAccentColor(value || DEFAULT_ACCENT_COLOR);
    },
    { immediate: true },
  );
  onMounted(() => {
    stopSystemTheme = listenForSystemTheme();
  });
  onBeforeUnmount(() => stopSystemTheme?.());

  return {
    accentColor,
    accentSaving,
    saveSettings,
    selectAccentColor,
    selectTheme,
    themePreference,
  };
}
