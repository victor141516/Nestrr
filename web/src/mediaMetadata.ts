import type { RouteLocationRaw } from "vue-router";

export type MediaMetadataItem = {
  id: number | string;
  label: string;
  to?: RouteLocationRaw;
};

export type TMDBNamedItem = {
  id?: number;
  name?: string;
};

export type TMDBCountry = {
  iso_3166_1?: string;
  name?: string;
};

function exploreType(kind: string): "Movies" | "Shows" {
  return kind === "tv" ? "Shows" : "Movies";
}

function validNamedItems(items: TMDBNamedItem[] | null | undefined) {
  const seen = new Set<number>();
  return (items || []).flatMap((item) => {
    const id = Number(item.id);
    const name = String(item.name || "").trim();
    if (!Number.isInteger(id) || id <= 0 || !name || seen.has(id)) return [];
    seen.add(id);
    return [{ id, name }];
  });
}

export function genreExploreRoute(
  kind: string,
  genreIDs: number[],
): RouteLocationRaw {
  return {
    path: "/",
    query: {
      type: exploreType(kind),
      includeGenres: [...new Set(genreIDs)].join(","),
    },
  };
}

export function keywordExploreRoute(
  kind: string,
  keywordID: number,
): RouteLocationRaw {
  return {
    path: "/",
    query: {
      type: exploreType(kind),
      includeKeywords: String(keywordID),
    },
  };
}

export function companyExploreRoute(
  kind: string,
  company: { id: number; name: string },
): RouteLocationRaw {
  return {
    path: "/",
    query: {
      type: exploreType(kind),
      includeCompanies: String(company.id),
      companyNames: JSON.stringify([company.name]),
    },
  };
}

export function countryExploreRoute(
  kind: string,
  countryCode: string,
): RouteLocationRaw {
  return {
    path: "/",
    query: {
      type: exploreType(kind),
      countries: countryCode,
    },
  };
}

export function genreMetadataItems(
  kind: string,
  genres: TMDBNamedItem[] | null | undefined,
): MediaMetadataItem[] {
  return validNamedItems(genres).map((genre) => ({
    id: genre.id,
    label: genre.name,
    to: genreExploreRoute(kind, [genre.id]),
  }));
}

export function keywordMetadataItems(
  kind: string,
  keywords: TMDBNamedItem[] | null | undefined,
): MediaMetadataItem[] {
  return validNamedItems(keywords).map((keyword) => ({
    id: keyword.id,
    label: keyword.name,
    to: keywordExploreRoute(kind, keyword.id),
  }));
}

export function companyMetadataItems(
  kind: string,
  companies: TMDBNamedItem[] | null | undefined,
): MediaMetadataItem[] {
  return validNamedItems(companies).map((company) => ({
    id: company.id,
    label: company.name,
    to: companyExploreRoute(kind, company),
  }));
}

export function countryMetadataItems(
  kind: string,
  countries: TMDBCountry[] | null | undefined,
): MediaMetadataItem[] {
  const seen = new Set<string>();
  return (countries || []).flatMap((country) => {
    const id = String(country.iso_3166_1 || "")
      .trim()
      .toUpperCase();
    const label = String(country.name || "").trim();
    if (!id || !label || seen.has(id)) return [];
    seen.add(id);
    return [{ id, label, to: countryExploreRoute(kind, id) }];
  });
}
