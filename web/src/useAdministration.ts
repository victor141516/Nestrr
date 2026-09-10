import { onBeforeUnmount, ref, type Ref } from "vue";
import type { Router } from "vue-router";
import { acceptTokens, api, session, type User } from "./api";
import type { TranslationKey } from "./i18n";

type Options = {
  router: Router;
  t: (key: TranslationKey) => string;
  flash: (message: string) => void;
  loadRoute: () => Promise<void>;
  go: (path: string) => void;
  busy: Ref<boolean>;
};

type WorkProgress = {
  total: number;
  done: number;
  found?: number;
  added?: number;
  failed: number;
};

export type RecommendationRunStatus = {
  id?: number;
  state: "idle" | "running" | "completed" | "failed";
  phase: string;
  startedAt?: string;
  updatedAt?: string;
  finishedAt?: string;
  usersScanned: number;
  signals: {
    genres: number;
    keywords: number;
    actors: number;
    directors: number;
    companies: number;
  };
  discovery: WorkProgress;
  hydration: WorkProgress;
  catalogCount: number;
  pending: number;
  lastError?: string;
};

export function useAdministration(options: Options) {
  const { router, t, flash, loadRoute, go, busy } = options;
  const users = ref<User[]>([]);
  const adminState = ref<any>(null);
  const recommendationRun = ref<RecommendationRunStatus | null>(null);
  const adminAction = ref<{
    type: "rename" | "reset" | "delete";
    user: User;
    value: string;
  } | null>(null);
  const confirmMetadata = ref(false);
  let recommendationPoll: number | null = null;

  function stopRecommendationPolling(): void {
    if (recommendationPoll === null) return;
    window.clearTimeout(recommendationPoll);
    recommendationPoll = null;
  }

  function scheduleRecommendationPolling(): void {
    stopRecommendationPolling();
    if (recommendationRun.value?.state !== "running") return;
    recommendationPoll = window.setTimeout(async () => {
      recommendationPoll = null;
      await loadRecommendationStatus();
    }, 1000);
  }

  async function loadRecommendationStatus(): Promise<void> {
    recommendationRun.value = await api<RecommendationRunStatus>(
      "/admin/recommendations/catalog",
    );
    scheduleRecommendationPolling();
  }

  async function loadAdmin() {
    const [r, state, recommendations] = await Promise.all([
      api<any>("/admin/users"),
      api<any>("/admin/state"),
      api<RecommendationRunStatus>("/admin/recommendations/catalog"),
    ]);
    users.value = r.users;
    adminState.value = state;
    recommendationRun.value = recommendations;
    scheduleRecommendationPolling();
  }

  async function runRecommendationCatalog(): Promise<void> {
    recommendationRun.value = await api<RecommendationRunStatus>(
      "/admin/recommendations/catalog/run",
      { method: "POST" },
    );
    scheduleRecommendationPolling();
  }

  async function clearMetadata() {
    await api("/admin/metadata", { method: "DELETE" });
    confirmMetadata.value = false;
    await loadAdmin();
    flash(t("clear"));
  }
  async function addUser(ev: Event) {
    const f = new FormData(ev.target as HTMLFormElement);
    await api("/admin/users", {
      method: "POST",
      body: JSON.stringify({
        username: f.get("username"),
        password: f.get("password"),
        language: f.get("language"),
      }),
    });
    (ev.target as HTMLFormElement).reset();
    await loadAdmin();
  }
  async function impersonate(id: number) {
    const p = await api<any>(`/admin/users/${id}/impersonate`, {
      method: "POST",
    });
    acceptTokens(p);
    go("/");
    await loadRoute();
  }
  async function stopImpersonation() {
    const p = await api<any>("/admin/impersonation/stop", { method: "POST" });
    acceptTokens(p);
    go("/");
    await loadRoute();
  }
  function openAdminAction(type: "rename" | "reset" | "delete", user: User) {
    adminAction.value = {
      type,
      user,
      value: type === "rename" ? user.username : "",
    };
  }
  async function confirmAdminAction() {
    const action = adminAction.value;
    if (!action) return;
    busy.value = true;
    try {
      if (action.type === "delete")
        await api(`/admin/users/${action.user.id}`, { method: "DELETE" });
      if (
        action.type === "rename" &&
        action.value.trim() &&
        action.value.trim() !== action.user.username
      )
        await api(`/admin/users/${action.user.id}`, {
          method: "PATCH",
          body: JSON.stringify({ username: action.value.trim() }),
        });
      if (action.type === "reset" && action.value.length >= 10) {
        await api(`/admin/users/${action.user.id}/password`, {
          method: "PUT",
          body: JSON.stringify({ password: action.value }),
        });
        flash("Password reset and sessions revoked");
      }
      adminAction.value = null;
      await loadAdmin();
    } finally {
      busy.value = false;
    }
  }

  onBeforeUnmount(stopRecommendationPolling);

  return {
    addUser,
    adminAction,
    adminState,
    clearMetadata,
    confirmAdminAction,
    confirmMetadata,
    impersonate,
    loadAdmin,
    openAdminAction,
    recommendationRun,
    runRecommendationCatalog,
    stopImpersonation,
    users,
  };
}
