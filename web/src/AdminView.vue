<script setup lang="ts">
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AppIcon from "./AppIcon.vue";

const {
  addUser,
  adminState,
  artwork,
  healthyIntegrations,
  impersonate,
  integrationCount,
  integrationLabel,
  integrations,
  openAdminAction,
  t,
  users,
} = useNestrrApplicationContext();
</script>

<template>
  <section class="content page admin-page">
    <h1>{{ t("admin") }}</h1>
    <form class="panel admin-add" @submit.prevent="addUser">
      <h2>{{ t("addUser") }}</h2>
      <label
        >{{ t("username")
        }}<input name="username" required minlength="2" maxlength="40" /></label
      ><label
        >{{ t("temporaryPassword")
        }}<input
          name="password"
          required
          type="password"
          minlength="10" /></label
      ><label
        >{{ t("language")
        }}<select name="language">
          <option value="en-US">English</option>
          <option value="es-ES">Español</option>
        </select></label
      ><button class="primary">
        <AppIcon name="plus" />{{ t("addUser") }}
      </button>
    </form>
    <section class="panel users-panel">
      <h2>{{ t("users") }}</h2>
      <article v-for="u in users" :key="u.id" class="user-row">
        <div class="avatar large">{{ u.username[0]?.toUpperCase() }}</div>
        <div class="user-copy">
          <b>{{ u.username }}</b
          ><small
            >{{ u.isAdmin ? t("administrator") : t("member") }} ·
            {{ u.language }}</small
          >
        </div>
        <span v-if="u.isAdmin" class="admin-badge">ADMIN</span>
        <div class="user-actions">
          <button @click="openAdminAction('rename', u)">
            <AppIcon name="pencil" /><span>{{ t("rename") }}</span></button
          ><button @click="openAdminAction('reset', u)">
            <AppIcon name="refresh" /><span>{{
              t("resetPassword")
            }}</span></button
          ><button v-if="!u.isAdmin" @click="impersonate(u.id)">
            <AppIcon name="eye" /><span>{{ t("viewAsUser") }}</span></button
          ><button
            v-if="!u.isAdmin"
            class="danger"
            @click="openAdminAction('delete', u)"
          >
            <AppIcon name="trash" /><span>{{ t("delete") }}</span>
          </button>
        </div>
      </article>
    </section>
    <section v-if="adminState" class="panel server-panel">
      <div class="section-heading">
        <div>
          <h2>{{ t("serverStatus") }}</h2>
          <p>
            {{ healthyIntegrations }}/{{ integrationCount }}
            {{ t("servicesHealthy") }}
          </p>
        </div>
      </div>
      <div class="server-services">
        <div v-for="(v, k) in integrations" :key="k">
          <span
            :class="[
              'service-icon',
              v.healthy ? 'healthy' : v.configured ? 'degraded' : 'off',
            ]"
            ><AppIcon :name="v.healthy ? 'check' : 'alert'" :size="15" /></span
          ><b>{{ integrationLabel(String(k)) }}</b
          ><small>{{ v.healthy ? t("healthy") : v.message }}</small>
        </div>
      </div>
      <div class="server-facts">
        <p>
          <b>TMDB</b>
          {{
            adminState.config.tmdbConfigured
              ? t("configured")
              : t("notConfigured")
          }}
        </p>
        <p>
          <b>{{ t("stored") }}</b> {{ adminState.counts.metadata }} metadata ·
          {{ adminState.counts.image_assets }} artwork ·
          {{ adminState.counts.progress }} progress
        </p>
      </div>
    </section>
  </section>
</template>
