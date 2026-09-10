import {
  mediaTypeFromQuery,
  mediaTypeQueryValue,
  type CatalogMediaType,
} from "./mediaType";

export type ListSort =
  "addedAt" | "title" | "releaseDate" | "rating" | "popularity" | "runtime";
export type ListDirection = "asc" | "desc";
export type ListRouteQueryValue = string | null | (string | null)[] | undefined;
export type ListRouteQuery = Record<string, ListRouteQueryValue>;
export type ListViewState = {
  mediaType: CatalogMediaType;
  sort: ListSort;
  direction: ListDirection;
};

const listSorts = new Set<ListSort>([
  "addedAt",
  "title",
  "releaseDate",
  "rating",
  "popularity",
  "runtime",
]);

function firstQueryValue(value: unknown): unknown {
  return Array.isArray(value) ? value[0] : value;
}

export function parseListViewQuery(
  query: Record<string, unknown>,
): ListViewState {
  const requestedSort = firstQueryValue(query.sort);
  const requestedDirection = firstQueryValue(query.direction);
  return {
    mediaType: mediaTypeFromQuery(query.type),
    sort:
      typeof requestedSort === "string" &&
      listSorts.has(requestedSort as ListSort)
        ? (requestedSort as ListSort)
        : "addedAt",
    direction: requestedDirection === "asc" ? "asc" : "desc",
  };
}

export function listViewQuery(
  query: ListRouteQuery,
  state: ListViewState,
): ListRouteQuery {
  return {
    ...query,
    type: mediaTypeQueryValue(state.mediaType),
    sort: state.sort,
    direction: state.direction,
  };
}

export function isCanonicalListViewQuery(
  query: Record<string, unknown>,
  state: ListViewState,
): boolean {
  return (
    firstQueryValue(query.type) === mediaTypeQueryValue(state.mediaType) &&
    firstQueryValue(query.sort) === state.sort &&
    firstQueryValue(query.direction) === state.direction
  );
}
