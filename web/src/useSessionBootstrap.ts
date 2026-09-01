import { onBeforeUnmount, onMounted, ref, shallowRef } from "vue";
import type { Router } from "vue-router";
import {
  acceptTokens,
  api,
  clearSession,
  refresh,
  session,
  startSessionRefresh,
} from "./api";

interface SessionBootstrapOptions {
  router: Router;
  loadRoute: () => Promise<void>;
}

export function useSessionBootstrap(options: SessionBootstrapOptions) {
  const ready = ref(false);
  const needsBootstrap = ref(false);
  const rottenTomatoesEnabled = shallowRef(false);
  const error = ref("");
  const busy = ref(false);
  let stopSessionRefresh: undefined | (() => void);

  async function initialize() {
    try {
      const response = await fetch("/api/v1/setup");
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const setup = await response.json();
      needsBootstrap.value = setup.needsBootstrap;
      rottenTomatoesEnabled.value = Boolean(setup.rottenTomatoesEnabled);
      if (!needsBootstrap.value) await refresh();
      if (session.user) await options.loadRoute();
    } catch (reason) {
      error.value =
        reason instanceof Error
          ? reason.message
          : "The request could not be completed.";
    } finally {
      ready.value = true;
    }
  }

  async function authenticate(mode: "login" | "bootstrap", event: Event) {
    busy.value = true;
    error.value = "";
    const form = new FormData(event.target as HTMLFormElement);
    try {
      const payload = await api(`/auth/${mode}`, {
        method: "POST",
        body: JSON.stringify({
          username: form.get("username"),
          password: form.get("password"),
          language: navigator.language,
        }),
      });
      acceptTokens(payload);
      needsBootstrap.value = false;
      await options.loadRoute();
    } catch (reason) {
      error.value =
        reason instanceof Error
          ? reason.message
          : "The request could not be completed.";
    } finally {
      busy.value = false;
    }
  }

  async function logout() {
    await api("/auth/logout", { method: "POST" }).catch(() => {});
    clearSession();
    await options.router.push("/");
  }

  onMounted(() => {
    stopSessionRefresh = startSessionRefresh();
    void initialize();
  });

  onBeforeUnmount(() => stopSessionRefresh?.());

  return {
    authenticate,
    busy,
    error,
    logout,
    needsBootstrap,
    ready,
    rottenTomatoesEnabled,
  };
}
