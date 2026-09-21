<script setup lang="ts">
import { provide } from "vue";
import AppDialog from "./AppDialog.vue";
import AppShellHeader from "./AppShellHeader.vue";
import AppIcon from "./AppIcon.vue";
import AuthView from "./AuthView.vue";
import EmbeddedPlayer from "./EmbeddedPlayer.vue";
import TorrentSearchModal from "./TorrentSearchModal.vue";
import { useTelevisionNavigation } from "./useTelevisionNavigation";
import {
  nestrrApplicationKey,
  useNestrrApplication,
} from "./useNestrrApplication";

const application = useNestrrApplication();
const { television } = useTelevisionNavigation(application);
provide(nestrrApplicationKey, application);
const {
  adminAction,
  authenticate,
  autoplay,
  busy,
  cancelAutoplay,
  categories,
  clearMetadata,
  confirmAdminAction,
  confirmMetadata,
  error,
  flash,
  library,
  listEditor,
  loadRoute,
  logout,
  navigateEpisode,
  needsBootstrap,
  onPlayerMessage,
  onTorrentAccepted,
  pageLoading,
  pathOnly,
  player,
  ready,
  saveListEditor,
  session,
  stopImpersonation,
  t,
  toast,
  torrent,
} = application;
</script>

<template>
  <div v-if="!ready" class="boot-screen" aria-busy="true">
    <div class="brand brand-large">
      <span><AppIcon name="sparkles" /></span>Nestrr
    </div>
    <span class="spinner" />
    <p>{{ t("checking") }}</p>
  </div>

  <AuthView
    v-else-if="!session.user"
    :needs-bootstrap="needsBootstrap"
    :busy="busy"
    :error="error"
    :translate="t"
    @authenticate="authenticate"
  />

  <template v-else>
    <EmbeddedPlayer
      v-if="player"
      :url="player.url"
      :origin="player.origin"
      :television="television"
      :autoplay-seconds="autoplay?.seconds"
      :labels="{
        upNext: t('upNext'),
        nextEpisodeIn: t('nextEpisodeIn'),
        playNow: t('playNow'),
        cancel: t('cancel'),
      }"
      @player-event="onPlayerMessage"
      @play-next="
        cancelAutoplay();
        navigateEpisode('next');
      "
      @cancel-autoplay="cancelAutoplay"
    />

    <div v-show="!player" class="app-shell">
      <AppShellHeader
        :user="session.user"
        :impersonating="session.impersonating"
        :path="pathOnly"
        :translate="t"
        @logout="logout"
        @stop-impersonation="stopImpersonation"
      />

      <p v-if="error && pathOnly !== '/'" class="global-error" role="alert">
        {{ error }} <button @click="loadRoute">{{ t("retry") }}</button>
      </p>
      <div v-if="pageLoading" class="page-loader" aria-busy="true">
        <span class="spinner" /><span>{{ t("loading") }}</span>
      </div>

      <main v-else class="main-view">
        <RouterView />
      </main>

      <nav class="mobile-nav" aria-label="Primary">
        <RouterLink :class="{ active: pathOnly === '/' }" to="/"
          ><AppIcon name="compass" /><span>{{ t("explore") }}</span></RouterLink
        ><RouterLink
          :class="{ active: pathOnly.startsWith('/lists') }"
          to="/lists"
          ><AppIcon name="list" /><span>{{ t("lists") }}</span></RouterLink
        ><RouterLink
          :class="{ active: pathOnly === '/recommendations' }"
          to="/recommendations"
          ><AppIcon name="sparkles" /><span>{{
            t("recommendations")
          }}</span></RouterLink
        >
      </nav>

      <AppDialog
        v-if="listEditor"
        panel-class="confirm-dialog list-editor-dialog"
        labelledby="list-editor-title"
        @close="listEditor = null"
      >
        <form class="dialog-form" @submit.prevent="saveListEditor">
          <button
            type="button"
            class="close"
            :aria-label="t('close')"
            @click="listEditor = null"
          >
            <AppIcon name="x" />
          </button>
          <span
            :class="['dialog-icon', { danger: listEditor.mode === 'delete' }]"
            ><AppIcon :name="listEditor.mode === 'delete' ? 'trash' : 'list'"
          /></span>
          <h2 id="list-editor-title">
            {{
              listEditor.mode === "create"
                ? t("newList")
                : listEditor.mode === "rename"
                  ? t("renameList")
                  : t("deleteList")
            }}
          </h2>
          <p v-if="listEditor.mode === 'delete'">
            {{ t("deleteListHelp") }} <b>{{ listEditor.list?.name }}</b>
          </p>
          <label v-else
            >{{ t("listName") }}
            <input
              v-model="listEditor.name"
              maxlength="60"
              required
              autofocus
            />
          </label>
          <div class="dialog-actions">
            <button type="button" @click="listEditor = null">
              {{ t("cancel") }}
            </button>
            <button
              :class="listEditor.mode === 'delete' ? 'danger' : 'primary'"
              :disabled="
                listEditor.mode !== 'delete' && !listEditor.name.trim()
              "
            >
              {{ listEditor.mode === "delete" ? t("delete") : t("save") }}
            </button>
          </div>
        </form>
      </AppDialog>

      <TorrentSearchModal
        v-if="torrent"
        :item="torrent.item"
        :kind="torrent.kind"
        :categories="categories"
        :translate="t"
        @close="torrent = null"
        @accepted="onTorrentAccepted"
        @notice="flash"
      />

      <AppDialog
        v-if="confirmMetadata"
        panel-class="confirm-dialog"
        labelledby="confirm-metadata-title"
        describedby="confirm-metadata-description"
        role="alertdialog"
        @close="confirmMetadata = false"
      >
        <button class="close" @click="confirmMetadata = false">
          <AppIcon name="x" /></button
        ><span class="dialog-icon danger"><AppIcon name="alert" /></span>
        <h2 id="confirm-metadata-title">
          {{ t("confirmMetadataDelete") }}
        </h2>
        <p id="confirm-metadata-description">
          {{ t("confirmMetadataHelp") }}
        </p>
        <div class="dialog-actions">
          <button data-dialog-initial-focus @click="confirmMetadata = false">
            {{ t("cancel") }}</button
          ><button class="danger-fill" @click="clearMetadata">
            <AppIcon name="trash" />{{ t("clear") }}
          </button>
        </div>
      </AppDialog>

      <AppDialog
        v-if="adminAction"
        panel-class="confirm-dialog"
        labelledby="admin-action-title"
        @close="adminAction = null"
      >
        <button class="close" @click="adminAction = null">
          <AppIcon name="x" /></button
        ><span
          :class="[
            'dialog-icon',
            adminAction.type === 'delete' ? 'danger' : '',
          ]"
          ><AppIcon
            :name="
              adminAction.type === 'rename'
                ? 'pencil'
                : adminAction.type === 'reset'
                  ? 'refresh'
                  : 'alert'
            "
        /></span>
        <h2 id="admin-action-title">
          {{
            adminAction.type === "rename"
              ? t("rename")
              : adminAction.type === "reset"
                ? t("resetPassword")
                : t("delete")
          }}
          · {{ adminAction.user.username }}
        </h2>
        <label v-if="adminAction.type !== 'delete'"
          >{{
            adminAction.type === "rename"
              ? t("username")
              : t("temporaryPassword")
          }}<input
            v-model="adminAction.value"
            :type="adminAction.type === 'reset' ? 'password' : 'text'"
            :minlength="adminAction.type === 'reset' ? 10 : 2"
        /></label>
        <p v-if="adminAction.type === 'reset'" class="inline-message warning">
          <AppIcon name="alert" />{{ t("sessionsRevoked") }}
        </p>
        <p v-if="adminAction.type === 'delete'">{{ t("deleteUserHelp") }}</p>
        <div class="dialog-actions">
          <button data-dialog-initial-focus @click="adminAction = null">
            {{ t("cancel") }}</button
          ><button
            :class="adminAction.type === 'delete' ? 'danger-fill' : 'primary'"
            :disabled="
              busy ||
              (adminAction.type === 'reset' && adminAction.value.length < 10)
            "
            @click="confirmAdminAction"
          >
            {{ t("confirm") }}
          </button>
        </div>
      </AppDialog>

      <div v-if="toast" class="toast" role="status">{{ toast }}</div>
    </div>
  </template>
</template>
