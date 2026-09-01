<script setup lang="ts">
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";

const props = defineProps<{
  needsBootstrap: boolean;
  busy: boolean;
  error: string;
  translate: (key: TranslationKey) => string;
}>();

const emit = defineEmits<{
  authenticate: [mode: "login" | "bootstrap", event: Event];
}>();
</script>

<template>
  <main class="auth-shell">
    <form
      class="auth-card"
      :aria-busy="busy"
      @submit.prevent="
        emit('authenticate', needsBootstrap ? 'bootstrap' : 'login', $event)
      "
    >
      <div class="brand brand-large">
        <span><AppIcon name="sparkles" /></span>Nestrr
      </div>
      <h2>
        {{ translate(needsBootstrap ? "createAdmin" : "signIn") }}
      </h2>
      <p v-if="needsBootstrap" class="notice warning">
        <AppIcon name="alert" />{{ translate("bootstrapWarning") }}
      </p>
      <label
        >{{ translate("username")
        }}<input
          name="username"
          autocomplete="username"
          required
          minlength="2"
          maxlength="40" /></label
      ><label
        >{{ translate("password")
        }}<input
          name="password"
          type="password"
          :autocomplete="needsBootstrap ? 'new-password' : 'current-password'"
          required
          minlength="10"
      /></label>
      <p v-if="error" class="inline-message error" role="alert">
        ! {{ error }}
      </p>
      <button class="primary full" :disabled="busy">
        <span v-if="busy" class="mini-spinner" />{{
          translate(
            busy ? "working" : needsBootstrap ? "createAdmin" : "signIn",
          )
        }}
      </button>
    </form>
  </main>
</template>
