import { api } from "./api";
import type { FilterOption } from "./movieFilters";

type CompanyResponse = { results?: FilterOption[] };

export function mergeCompanies(
  local: FilterOption[],
  remote: FilterOption[],
): FilterOption[] {
  const results = new Map<number, FilterOption>();
  for (const company of [...local, ...remote])
    if (!results.has(company.id)) results.set(company.id, company);
  return [...results.values()];
}

export async function searchCompanies(
  query: string,
  phase: "local" | "remote",
  signal?: AbortSignal,
): Promise<FilterOption[]> {
  const params = new URLSearchParams({ q: query, phase, limit: "12" });
  const response = await api<CompanyResponse>(`/companies/search?${params}`, {
    signal,
  });
  return response.results || [];
}

export async function resolveCompanies(
  ids: number[],
  signal?: AbortSignal,
): Promise<FilterOption[]> {
  const unique = [
    ...new Set(ids.filter((id) => Number.isInteger(id) && id > 0)),
  ];
  if (!unique.length) return [];
  const response = await api<CompanyResponse>(
    `/companies/search?ids=${unique.join(",")}`,
    { signal },
  );
  return response.results || [];
}
