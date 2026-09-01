import { computed, onBeforeUnmount, readonly, shallowRef } from "vue";
import { streamAPI } from "./api";
import type { MediaListItem } from "./mediaLists";

export type BulkImportStatus =
  | "queued"
  | "database_search"
  | "database_match"
  | "database_miss"
  | "tmdb_search"
  | "tmdb_match"
  | "saved"
  | "not_found"
  | "error"
  | "complete";

export type BulkImportEvent = {
  index?: number;
  input?: string;
  status: BulkImportStatus;
  source?: "database" | "tmdb";
  added?: boolean;
  item?: MediaListItem;
  error?: string;
};

export type BulkImportEntry = Required<
  Pick<BulkImportEvent, "index" | "input">
> &
  Omit<BulkImportEvent, "index" | "input">;

type UseBulkListImportOptions = {
  listId: () => number;
  onImported?: (event: BulkImportEvent) => void;
  onFinished?: () => void | Promise<void>;
};

function inputLines(value: string) {
  return value
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

export function useBulkListImport(options: UseBulkListImportOptions) {
  const text = shallowRef("");
  const entries = shallowRef<BulkImportEntry[]>([]);
  const running = shallowRef(false);
  const completed = shallowRef(false);
  const error = shallowRef("");
  let controller: AbortController | null = null;

  const lines = computed(() => inputLines(text.value));
  const canSubmit = computed(
    () => !running.value && lines.value.length > 0 && lines.value.length <= 500,
  );
  const unresolved = computed(() =>
    entries.value.filter(
      (entry) => entry.status === "not_found" || entry.status === "error",
    ),
  );

  function applyEvent(event: BulkImportEvent) {
    if (event.status === "complete") {
      completed.value = true;
      return;
    }
    const index = event.index ?? 0;
    entries.value = entries.value.map((entry) =>
      entry.index === index
        ? {
            ...entry,
            ...event,
            index,
            input: event.input ?? entry.input,
          }
        : entry,
    );
    if (event.status === "saved" && event.item) options.onImported?.(event);
  }

  async function submit() {
    if (!canSubmit.value) return;
    controller?.abort();
    controller = new AbortController();
    const submittedText = text.value;
    entries.value = lines.value.map((input, index) => ({
      index,
      input,
      status: "queued",
    }));
    running.value = true;
    completed.value = false;
    error.value = "";
    try {
      await streamAPI<BulkImportEvent>(
        `/lists/${options.listId()}/bulk-import`,
        {
          method: "POST",
          body: JSON.stringify({ titles: submittedText }),
          signal: controller.signal,
        },
        applyEvent,
      );
      completed.value = true;
      await options.onFinished?.();
    } catch (cause) {
      if ((cause as DOMException)?.name !== "AbortError")
        error.value =
          cause instanceof Error ? cause.message : "The import failed";
    } finally {
      running.value = false;
      controller = null;
    }
  }

  function cancel() {
    controller?.abort();
  }

  function reset() {
    cancel();
    text.value = "";
    entries.value = [];
    completed.value = false;
    error.value = "";
  }

  onBeforeUnmount(cancel);

  return {
    canSubmit,
    completed: readonly(completed),
    entries: readonly(entries),
    error: readonly(error),
    running: readonly(running),
    text,
    unresolved,
    reset,
    submit,
  };
}
