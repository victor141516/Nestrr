import { nextTick, onBeforeUnmount, onMounted, watch, type Ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  directionalTarget,
  remoteKey,
  televisionEnabled,
  type Direction,
} from "./television";
import "./television.css";

const selector =
  'a[href],button,input:not([type="hidden"]),select,textarea,summary,[tabindex="0"]';

function candidates(root: ParentNode = document): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(selector)).filter(
    (el) => {
      const rect = el.getBoundingClientRect();
      return (
        rect.width > 0 &&
        rect.height > 0 &&
        getComputedStyle(el).visibility !== "hidden" &&
        !el.matches(':disabled,[aria-disabled="true"]') &&
        !el.closest('[inert],[aria-hidden="true"]')
      );
    },
  );
}

function focus(element?: HTMLElement) {
  element?.focus({ preventScroll: true });
  element?.scrollIntoView({ block: "nearest", inline: "nearest" });
}

export function useTelevisionNavigation(options: {
  player: Ref<unknown>;
  closePlayer: () => void;
  ready: Ref<boolean>;
  pageLoading: Ref<boolean>;
}) {
  const enabled = televisionEnabled();
  const route = useRoute();
  const router = useRouter();
  let observer: MutationObserver | undefined;
  let pendingFocus = false;
  let disposed = false;
  let automaticFallback: HTMLElement | undefined;
  const remembered = new Map<string, string>();

  function ensureFocus() {
    if (pendingFocus || disposed) return;
    pendingFocus = true;
    void nextTick(() => {
      pendingFocus = false;
      if (
        disposed ||
        options.player.value ||
        !options.ready.value ||
        options.pageLoading.value
      )
        return;
      const all = candidates();
      if (
        all.includes(document.activeElement as HTMLElement) &&
        document.activeElement !== automaticFallback
      )
        return;
      const href = remembered.get(route.path);
      const preferred =
        all.find((el) => href && el.getAttribute("href") === href) ||
        all.find((el) =>
          el.matches(".auth-card input,[data-tv-play],.media-card"),
        );
      automaticFallback = preferred ? undefined : all[0];
      focus(preferred || automaticFallback);
    });
  }

  function remember(event: FocusEvent) {
    const el = event.target as HTMLElement;
    const href = el.getAttribute?.("href");
    if (href && el.matches(".media-card")) remembered.set(route.path, href);
  }

  function keydown(event: KeyboardEvent) {
    automaticFallback = undefined;
    if (
      event.defaultPrevented ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey
    )
      return;
    const key = remoteKey(event);
    const active = document.activeElement as HTMLElement | null;
    const dialogs = Array.from(
      document.querySelectorAll<HTMLElement>(
        '[role="dialog"],[role="alertdialog"]',
      ),
    );
    const dialog = dialogs
      .filter((el) => el.getBoundingClientRect().width > 0)
      .pop();
    if (key === "Back" || key === "Escape") {
      // Let the TV dismiss its keyboard before leaving the login/search field.
      if (active?.matches("input,textarea") || active?.isContentEditable) {
        if (key === "Escape") active.blur();
        return;
      }
      if (key === "Escape" && dialog) return;
      event.preventDefault();
      if (event.repeat) return;
      if (dialog) {
        (active || dialog).dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", bubbles: true }),
        );
      } else if (options.player.value) options.closePlayer();
      else if (window.history.state?.back) router.back();
      else if (route.path !== "/")
        void router.replace("/?downloaded=1&type=Movies");
      else window.close();
      return;
    }
    if (options.player.value) return;
    if (!["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(key))
      return;
    if (
      active?.isContentEditable ||
      active?.matches("textarea,select") ||
      (active?.matches("input") &&
        (key === "ArrowLeft" || key === "ArrowRight"))
    )
      return;
    const all = candidates(dialog || document);
    const target =
      active && all.includes(active)
        ? directionalTarget(
            active.getBoundingClientRect(),
            all
              .filter((el) => el !== active)
              .map((element) => ({
                element,
                rect: element.getBoundingClientRect(),
              })),
            key as Direction,
          )
        : all[0];
    event.preventDefault();
    focus(target);
  }

  onMounted(() => {
    if (!enabled) return;
    document.documentElement.classList.add("television");
    window.addEventListener("keydown", keydown);
    document.addEventListener("focusin", remember);
    observer = new MutationObserver(ensureFocus);
    observer.observe(document.getElementById("app") || document.body, {
      childList: true,
      subtree: true,
    });
    ensureFocus();
  });
  watch(
    [() => route.fullPath, options.ready, options.pageLoading, options.player],
    () => {
      if (enabled) ensureFocus();
    },
    { flush: "post" },
  );
  onBeforeUnmount(() => {
    disposed = true;
    observer?.disconnect();
    window.removeEventListener("keydown", keydown);
    document.removeEventListener("focusin", remember);
    if (enabled) document.documentElement.classList.remove("television");
  });
  return { television: enabled };
}
