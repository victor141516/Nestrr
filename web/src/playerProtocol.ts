export type PlarrEventType =
  | "ready"
  | "opened"
  | "play"
  | "pause"
  | "seek"
  | "heartbeat"
  | "ended"
  | "closed"
  | "page-hidden"
  | "audio-track-changed"
  | "subtitle-track-changed"
  | "subtitles-enabled"
  | "subtitles-disabled"
  | "automatic-rendition-changed"
  | "manual-quality-changed"
  | "next-requested"
  | "previous-requested"
  | "playback-error";

export interface PlarrEvent {
  source: "plarr";
  version: 1;
  eventId: string;
  sessionId: string;
  type: PlarrEventType;
  mediaPath: string;
  positionSeconds: number;
  durationSeconds: number;
  paused: boolean;
  occurredAt: string;
  [key: string]: unknown;
}

const eventTypes = new Set<PlarrEventType>([
  "ready",
  "opened",
  "play",
  "pause",
  "seek",
  "heartbeat",
  "ended",
  "closed",
  "page-hidden",
  "audio-track-changed",
  "subtitle-track-changed",
  "subtitles-enabled",
  "subtitles-disabled",
  "automatic-rendition-changed",
  "manual-quality-changed",
  "next-requested",
  "previous-requested",
  "playback-error",
]);

export function isPlarrEvent(value: unknown): value is PlarrEvent {
  if (!value || typeof value !== "object") return false;
  const event = value as Record<string, unknown>;
  return (
    event.source === "plarr" &&
    event.version === 1 &&
    typeof event.eventId === "string" &&
    event.eventId.length > 0 &&
    typeof event.sessionId === "string" &&
    event.sessionId.length > 0 &&
    typeof event.type === "string" &&
    eventTypes.has(event.type as PlarrEventType) &&
    typeof event.mediaPath === "string" &&
    typeof event.positionSeconds === "number" &&
    Number.isFinite(event.positionSeconds) &&
    typeof event.durationSeconds === "number" &&
    Number.isFinite(event.durationSeconds) &&
    typeof event.paused === "boolean" &&
    typeof event.occurredAt === "string"
  );
}
