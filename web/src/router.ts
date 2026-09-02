import { createRouter, createWebHashHistory } from "vue-router";
import AdminView from "./AdminView.vue";
import ExploreView from "./ExploreView.vue";
import ListsView from "./ListsView.vue";
import MediaDetailView from "./MediaDetailView.vue";
import NotFoundView from "./NotFoundView.vue";
import RecommendationsView from "./RecommendationsView.vue";
import SettingsView from "./SettingsView.vue";

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/search",
      redirect: (route) => ({ path: "/", query: route.query }),
    },
    {
      path: "/home",
      redirect: (route) => ({ path: "/", query: route.query }),
    },
    {
      path: "/explore",
      redirect: (route) => ({ path: "/", query: route.query }),
    },
    { path: "/", component: ExploreView },
    {
      path: "/library",
      redirect: (route) => ({
        path: "/",
        query: { ...route.query, downloaded: "1" },
      }),
    },
    { path: "/lists", component: ListsView },
    { path: "/lists/:id(\\d+)", component: ListsView },
    { path: "/recommendations", component: RecommendationsView },
    { path: "/settings", component: SettingsView },
    { path: "/admin", component: AdminView },
    { path: "/movie/:id(\\d+)", component: MediaDetailView },
    { path: "/tv/:id(\\d+)", component: MediaDetailView },
    { path: "/collection/:id(\\d+)", component: MediaDetailView },
    { path: "/:pathMatch(.*)*", component: NotFoundView },
  ],
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition;
    if (to.path === from.path && to.path === "/") return false;
    return { top: 0 };
  },
});
