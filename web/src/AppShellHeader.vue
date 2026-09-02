<script setup lang="ts">
import {
  nextTick,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
  watch,
} from "vue";
import { RouterLink } from "vue-router";
import type { User } from "./api";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";

const props = defineProps<{
  user: User;
  impersonating: boolean;
  path: string;
  translate: (key: TranslationKey) => string;
}>();

const emit = defineEmits<{
  logout: [];
  stopImpersonation: [];
}>();

const menuOpen = shallowRef(false);
const accountWrap = useTemplateRef<HTMLElement>("accountWrap");
const avatarButton = useTemplateRef<HTMLButtonElement>("avatarButton");

function closeMenu(returnFocus = false): void {
  if (!menuOpen.value) return;
  menuOpen.value = false;
  if (returnFocus) void nextTick(() => avatarButton.value?.focus());
}

async function toggleMenu(): Promise<void> {
  menuOpen.value = !menuOpen.value;
  if (!menuOpen.value) return;
  await nextTick();
  accountWrap.value?.querySelector<HTMLElement>("[role='menuitem']")?.focus();
}

function handleDocumentPointer(event: PointerEvent): void {
  if (!accountWrap.value?.contains(event.target as Node)) closeMenu();
}

function handleFocusOut(event: FocusEvent): void {
  const nextTarget = event.relatedTarget;
  if (nextTarget instanceof Node && accountWrap.value?.contains(nextTarget)) {
    return;
  }
  closeMenu();
}

watch(
  () => props.path,
  () => closeMenu(),
);

onMounted(() =>
  document.addEventListener("pointerdown", handleDocumentPointer),
);
onBeforeUnmount(() =>
  document.removeEventListener("pointerdown", handleDocumentPointer),
);

function logout(): void {
  closeMenu();
  emit("logout");
}
</script>

<template>
  <div v-if="impersonating" class="impersonation">
    <span
      ><AppIcon name="eye" /> {{ translate("viewingAs") }}
      <b>{{ user.username }}</b></span
    ><button @click="emit('stopImpersonation')">
      <AppIcon name="swap" /> {{ translate("returnAdmin") }}
    </button>
  </div>
  <header class="app-header">
    <RouterLink class="brand small" aria-label="Nestrr" to="/"
      ><span><AppIcon name="sparkles" /></span><b>Nestrr</b></RouterLink
    >
    <nav class="desktop-nav" aria-label="Primary">
      <RouterLink :class="{ active: path === '/' }" to="/"
        ><AppIcon name="compass" />{{ translate("explore") }}</RouterLink
      ><RouterLink :class="{ active: path.startsWith('/lists') }" to="/lists"
        ><AppIcon name="list" />{{ translate("lists") }}</RouterLink
      ><RouterLink
        :class="{ active: path === '/recommendations' }"
        to="/recommendations"
        ><AppIcon name="sparkles" />{{
          translate("recommendations")
        }}</RouterLink
      >
    </nav>
    <div class="header-spacer" />
    <div ref="accountWrap" class="account-wrap" @focusout="handleFocusOut">
      <button
        ref="avatarButton"
        class="avatar"
        aria-controls="account-menu"
        :aria-expanded="menuOpen"
        aria-haspopup="menu"
        @click="toggleMenu"
        @keydown.esc="closeMenu(true)"
      >
        {{ user.username[0]?.toUpperCase() }}
      </button>
      <div
        v-if="menuOpen"
        id="account-menu"
        class="user-menu"
        role="menu"
        @keydown.esc="closeMenu(true)"
      >
        <div class="menu-identity">
          <b>{{ user.username }}</b
          ><small>{{
            user.isAdmin ? translate("administrator") : translate("member")
          }}</small>
        </div>
        <RouterLink role="menuitem" to="/settings" @click="closeMenu()"
          ><AppIcon name="settings" />{{ translate("settings") }}</RouterLink
        ><RouterLink
          v-if="user.isAdmin && !impersonating"
          role="menuitem"
          to="/admin"
          @click="closeMenu()"
          ><AppIcon name="shield" />{{ translate("admin") }}</RouterLink
        ><button role="menuitem" class="danger" @click="logout">
          <AppIcon name="logout" />{{ translate("logout") }}
        </button>
      </div>
    </div>
  </header>
</template>
