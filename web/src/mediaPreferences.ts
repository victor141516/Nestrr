export function preferredAudioLanguage(saved?: string, configured?: string) {
  const explicit = saved?.trim();
  if (explicit) return explicit;
  return configured?.trim().split(/[-_]/)[0]?.toLowerCase() || "";
}
