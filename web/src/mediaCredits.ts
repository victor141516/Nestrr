export type MovieCreditPerson = {
  id: number;
  name: string;
  profilePath: string | null;
};

export type MovieCastMember = MovieCreditPerson & {
  character: string;
  order: number;
};

export type MovieCredits = {
  directors: MovieCreditPerson[];
  cast: MovieCastMember[];
};

type TMDBCreditPerson = {
  id?: unknown;
  name?: unknown;
  profile_path?: unknown;
  job?: unknown;
  character?: unknown;
  order?: unknown;
};

type TMDBCreditsPayload = {
  cast?: unknown;
  crew?: unknown;
};

function person(value: TMDBCreditPerson): MovieCreditPerson | null {
  const id = Number(value.id);
  const name = String(value.name || "").trim();
  if (!Number.isInteger(id) || id <= 0 || !name) return null;
  return {
    id,
    name,
    profilePath:
      typeof value.profile_path === "string" && value.profile_path.trim()
        ? value.profile_path
        : null,
  };
}

function creditList(value: unknown): TMDBCreditPerson[] {
  return Array.isArray(value)
    ? value.filter(
        (entry): entry is TMDBCreditPerson =>
          typeof entry === "object" && entry !== null,
      )
    : [];
}

export function movieCredits(payload: unknown, castLimit = 6): MovieCredits {
  const source =
    typeof payload === "object" && payload !== null
      ? (payload as TMDBCreditsPayload)
      : {};

  const seenDirectors = new Set<number>();
  const directors = creditList(source.crew).flatMap((entry) => {
    if (entry.job !== "Director") return [];
    const normalized = person(entry);
    if (!normalized || seenDirectors.has(normalized.id)) return [];
    seenDirectors.add(normalized.id);
    return [normalized];
  });

  const seenCast = new Set<number>();
  const cast = creditList(source.cast)
    .map((entry, index) => ({ entry, index }))
    .sort((left, right) => {
      const leftOrder = Number(left.entry.order);
      const rightOrder = Number(right.entry.order);
      return (
        (Number.isFinite(leftOrder) ? leftOrder : left.index) -
        (Number.isFinite(rightOrder) ? rightOrder : right.index)
      );
    })
    .flatMap(({ entry, index }) => {
      const normalized = person(entry);
      if (!normalized || seenCast.has(normalized.id)) return [];
      seenCast.add(normalized.id);
      const parsedOrder = Number(entry.order);
      return [
        {
          ...normalized,
          character: String(entry.character || "").trim(),
          order: Number.isFinite(parsedOrder) ? parsedOrder : index,
        },
      ];
    })
    .slice(0, Math.max(0, castLimit));

  return { directors, cast };
}
