<script setup lang="ts">
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AppIcon from "./AppIcon.vue";
import type { ThemePreference } from "./theme";

const {
  ACCENT_COLORS,
  accentColor,
  accentSaving,
  library,
  saveSettings,
  selectAccentColor,
  selectTheme,
  session,
  t,
  themePreference,
  title,
} = useNestrrApplicationContext();
</script>

<template>
  <section class="content page narrow">
    <h1>{{ t("settings") }}</h1>
    <div class="panel settings-card appearance-card">
      <div class="panel-title">
        <span><AppIcon name="palette" /></span>
        <h2>{{ t("appearance") }}</h2>
      </div>
      <div class="appearance-field">
        <span class="field-label">{{ t("theme") }}</span>
        <div class="theme-options" role="group" :aria-label="t('theme')">
          <button
            v-for="option in [
              { value: 'system', label: 'systemTheme', icon: 'monitor' },
              { value: 'light', label: 'lightTheme', icon: 'sun' },
              { value: 'dark', label: 'darkTheme', icon: 'moon' },
            ]"
            :key="option.value"
            type="button"
            :class="{ active: themePreference === option.value }"
            :aria-pressed="themePreference === option.value"
            @click="selectTheme(option.value as ThemePreference)"
          >
            <AppIcon :name="option.icon" />{{ t(option.label) }}
          </button>
        </div>
      </div>
      <fieldset class="accent-fieldset">
        <legend>{{ t("accentColor") }}</legend>
        <div class="accent-options">
          <button
            v-for="option in ACCENT_COLORS"
            :key="option.value"
            type="button"
            :class="{
              active: accentColor === option.value,
            }"
            :aria-label="t(option.label)"
            :title="t(option.label)"
            :aria-pressed="accentColor === option.value"
            :disabled="accentSaving"
            @click="selectAccentColor(option.value)"
          >
            <i :style="{ backgroundColor: option.value }" />
            <AppIcon
              v-if="accentColor === option.value"
              name="check"
              :size="16"
            />
          </button>
        </div>
      </fieldset>
    </div>
    <form class="panel settings-card" @submit.prevent="saveSettings">
      <div class="panel-title">
        <span><AppIcon name="globe" /></span>
        <h2>{{ t("languageContent") }}</h2>
      </div>
      <label
        >{{ t("language")
        }}<select name="language" :value="session.user?.language">
          <option value="en-US">English (US)</option>
          <option value="es-ES">Español (España)</option>
          <option value="pt-PT">Português</option>
          <option value="fr-FR">Français</option>
        </select></label
      >
      <button class="primary">
        <AppIcon name="check" />{{ t("saveChanges") }}
      </button>
    </form>
  </section>
</template>
