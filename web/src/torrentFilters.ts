export type TorrentFilterState = {
  minSeeds: number;
  trackers: string[];
  trackersActive?: boolean;
  include: string[];
  exclude: string[];
};

export type TorrentResult = {
  ID?: string | number;
  Title: string;
  Size?: number;
  Seeder?: number;
  Seeders?: number;
  Tracker?: string;
  Link?: string;
  MagnetURI?: string;
};

function terms(values: string[]) {
  return values
    .map((value) => value.trim().toLocaleLowerCase())
    .filter(Boolean);
}

export function filterTorrentResults<T extends TorrentResult>(
  results: T[],
  filters: TorrentFilterState,
) {
  const include = terms(filters.include);
  const exclude = terms(filters.exclude);
  return results
    .filter((result) => {
      const title = String(result.Title || "").toLocaleLowerCase();
      const seeders = Number(result.Seeders ?? result.Seeder ?? 0);
      return (
        seeders >= filters.minSeeds &&
        (!filters.trackersActive ||
          filters.trackers.includes(String(result.Tracker || ""))) &&
        include.every((term) => title.includes(term)) &&
        !exclude.some((term) => title.includes(term))
      );
    })
    .sort(
      (left, right) =>
        Number(right.Seeders ?? right.Seeder ?? 0) -
        Number(left.Seeders ?? left.Seeder ?? 0),
    );
}
