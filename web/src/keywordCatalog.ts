import { api } from "./api";
import type { KeywordOption } from "./movieFilters";

type KeywordResponse = { results?: KeywordOption[] };

export function mergeKeywordSuggestions(
  prefix: KeywordOption[],
  contains: KeywordOption[],
): KeywordOption[] {
  const results = new Map<number, KeywordOption>();
  for (const keyword of [...prefix, ...contains])
    results.set(keyword.id, keyword);
  return [...results.values()];
}

export async function searchKeywords(
  query: string,
  phase: "prefix" | "contains",
  signal?: AbortSignal,
): Promise<KeywordOption[]> {
  const params = new URLSearchParams({
    q: query,
    phase,
    limit: "12",
  });
  const response = await api<KeywordResponse>(`/keywords/search?${params}`, {
    signal,
  });
  return response.results || [];
}

export async function resolveKeywords(
  ids: number[],
  signal?: AbortSignal,
): Promise<KeywordOption[]> {
  const unique = [
    ...new Set(ids.filter((id) => Number.isInteger(id) && id > 0)),
  ];
  if (!unique.length) return [];
  const response = await api<KeywordResponse>(
    `/keywords/search?ids=${unique.join(",")}`,
    { signal },
  );
  return response.results || [];
}
