export type TorrentDownload = {
  infoHash: string;
  name: string;
  state: string;
  progress: number;
};

const stateNames: Record<string, string> = {
  error: "Error",
  missingFiles: "Missing files",
  uploading: "Uploading",
  pausedUP: "Paused uploading",
  stoppedUP: "Stopped uploading",
  queuedUP: "Queued uploading",
  stalledUP: "Stalled uploading",
  checkingUP: "Checking upload",
  forcedUP: "Forced uploading",
  allocating: "Allocating",
  downloading: "Downloading",
  metaDL: "Downloading metadata",
  pausedDL: "Paused",
  stoppedDL: "Stopped",
  queuedDL: "Queued",
  stalledDL: "Stalled",
  checkingDL: "Checking download",
  forcedDL: "Forced downloading",
  checkingResumeData: "Checking resume data",
  moving: "Moving",
  unknown: "Unknown",
};

export function torrentStateName(state: string) {
  return stateNames[state] || state || "Unknown";
}

export function isTorrentDownloading(state: string) {
  return state === "downloading" || state === "forcedDL";
}

export function isTorrentDownloaded(state: string) {
  return [
    "uploading",
    "pausedUP",
    "stoppedUP",
    "queuedUP",
    "stalledUP",
    "checkingUP",
    "forcedUP",
  ].includes(state);
}

export function isTorrentStalled(state: string) {
  return state === "stalledDL";
}

export function isTorrentForceable(state: string) {
  return state === "stalledDL" || state === "queuedDL";
}

type ForceConfirmationOptions = {
  attempts?: number;
  intervalMs?: number;
  wait?: (milliseconds: number) => Promise<void>;
};

export async function waitForTorrentForceConfirmation(
  refresh: () => Promise<TorrentDownload | undefined>,
  options: ForceConfirmationOptions = {},
) {
  const attempts = Math.max(1, options.attempts ?? 41);
  const intervalMs = Math.max(0, options.intervalMs ?? 500);
  const wait =
    options.wait ||
    ((milliseconds: number) =>
      new Promise<void>((resolve) => setTimeout(resolve, milliseconds)));
  let latest: TorrentDownload | undefined;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      latest = await refresh();
      if (latest && !isTorrentForceable(latest.state)) return latest;
    } catch {
      // qBittorrent may briefly be unavailable while applying the command.
      // Keep waiting; the regular download poll remains as a later fallback.
    }
    if (attempt + 1 < attempts) await wait(intervalMs);
  }
  return latest;
}

export function torrentPercent(progress: number) {
  return Math.round(Math.max(0, Math.min(1, Number(progress) || 0)) * 100);
}
