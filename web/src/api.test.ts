import { afterEach, describe, expect, it, vi } from "vitest";
import {
  acceptTokens,
  accessTokenNeedsRefresh,
  clearSession,
  refresh,
  session,
  startSessionRefresh,
} from "./api";

const user = { id: 1, username: "viewer", isAdmin: false, language: "es-ES" };
const tokenResponse = (accessToken: string, expiresIn = 7200) =>
  new Response(JSON.stringify({ accessToken, expiresIn, user }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

afterEach(() => {
  clearSession();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("access token refresh", () => {
  it("tracks the expiration supplied by the server", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-28T10:00:00Z"));
    acceptTokens({ accessToken: "access", expiresIn: 7200, user });

    expect(session.accessTokenExpiresAt).toBe(Date.now() + 7200_000);
    expect(accessTokenNeedsRefresh()).toBe(false);
  });

  it("renews one minute before expiration while the page is visible", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-28T10:00:00Z"));
    const fetchMock = vi.fn().mockResolvedValue(tokenResponse("renewed"));
    vi.stubGlobal("fetch", fetchMock);
    const stop = startSessionRefresh();
    acceptTokens({ accessToken: "access", expiresIn: 120, user });

    await vi.advanceTimersByTimeAsync(59_999);
    expect(fetchMock).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(session.accessToken).toBe("renewed");
    stop();
  });

  it("shares an in-flight refresh within the same page", async () => {
    let complete!: (response: Response) => void;
    const pending = new Promise<Response>((resolve) => {
      complete = resolve;
    });
    const fetchMock = vi.fn().mockReturnValue(pending);
    vi.stubGlobal("fetch", fetchMock);

    const first = refresh();
    const second = refresh();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    complete(tokenResponse("renewed"));

    await expect(Promise.all([first, second])).resolves.toEqual([true, true]);
  });

  it("does not restore a session cleared while refresh is in flight", async () => {
    acceptTokens({ accessToken: "old", expiresIn: 1, user });
    let complete!: (response: Response) => void;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockReturnValue(
        new Promise<Response>((resolve) => {
          complete = resolve;
        }),
      ),
    );

    const pending = refresh();
    clearSession();
    complete(tokenResponse("stale"));

    await expect(pending).resolves.toBe(false);
    expect(session.user).toBeNull();
    expect(session.accessToken).toBe("");
  });
});
