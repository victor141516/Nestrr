export function playbackPlayerURL(
  playerUrl: string,
  context: URLSearchParams,
): string {
  const url = new URL(playerUrl);
  const [path, query = ""] = url.hash.split("?");
  const params = new URLSearchParams(query);
  context.forEach((value, key) => params.set(key, value));
  url.hash = `${path}?${params}`;
  return url.toString();
}
