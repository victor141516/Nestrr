export type CatalogMediaType = "movie" | "tv";

export function mediaTypeFromQuery(value: unknown): CatalogMediaType {
  if (Array.isArray(value)) value = value[0];
  if (typeof value !== "string") return "movie";
  const normalized = value.trim().toLowerCase();
  return normalized === "shows" || normalized === "show" || normalized === "tv"
    ? "tv"
    : "movie";
}

export function mediaTypeQueryValue(
  value: CatalogMediaType,
): "Movies" | "Shows" {
  return value === "tv" ? "Shows" : "Movies";
}
