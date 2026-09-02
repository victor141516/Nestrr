import { api } from "./api";
import type { PersonOption } from "./movieFilters";

type PersonResponse = { results?: PersonOption[] };

export function mergePeople(local: PersonOption[], remote: PersonOption[]) {
  const results = new Map<number, PersonOption>();
  for (const person of [...local, ...remote])
    if (!results.has(person.id)) results.set(person.id, person);
  return [...results.values()];
}

export async function searchPeople(
  query: string,
  phase: "local" | "remote",
  signal?: AbortSignal,
): Promise<PersonOption[]> {
  const params = new URLSearchParams({ q: query, phase, limit: "12" });
  const response = await api<PersonResponse>(`/people/search?${params}`, {
    signal,
  });
  return response.results || [];
}

export async function resolvePeople(
  ids: number[],
  signal?: AbortSignal,
): Promise<PersonOption[]> {
  const unique = [
    ...new Set(ids.filter((id) => Number.isInteger(id) && id > 0)),
  ];
  if (!unique.length) return [];
  const response = await api<PersonResponse>(
    `/people/search?ids=${unique.join(",")}`,
    { signal },
  );
  return response.results || [];
}
