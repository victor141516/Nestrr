export interface DetailRequest {
  revision: number;
  kind: string;
  id: string;
}

export function createDetailRequest(
  revision: number,
  kind: string,
  id: string | number,
): DetailRequest {
  return { revision, kind, id: String(id) };
}

export function isLatestDetailRequest(
  request: DetailRequest,
  activeRevision: number,
): boolean {
  return request.revision === activeRevision;
}

export function isCurrentDetailRequest(
  request: DetailRequest,
  activeRevision: number,
  currentKind: string,
  currentDetail: { id?: string | number } | null | undefined,
): boolean {
  return (
    isLatestDetailRequest(request, activeRevision) &&
    currentKind === request.kind &&
    String(currentDetail?.id ?? "") === request.id
  );
}
