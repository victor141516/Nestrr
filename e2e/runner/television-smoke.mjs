import assert from "node:assert/strict";

// Exercises our browser behavior with LG key events. This is not a substitute
// for checking the installed app, virtual keyboard and decoder on an actual TV.
export async function televisionSmoke(browser, origin, artifacts) {
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    locale: "en-US",
  });
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  try {
    await page.goto(`${origin}/?tv=1#/?downloaded=1&type=Movies`);
    await page.locator('input[name="username"]:focus').waitFor();
    await page.keyboard.type("e2e-admin");
    await page.keyboard.press("ArrowDown");
    await page.locator('input[name="password"]:focus').waitFor();
    await page.keyboard.type("e2e-password-2026");
    await page.keyboard.press("ArrowDown");
    await page.locator(".auth-card button:focus").waitFor();
    await page.keyboard.press("Enter");
    await page
      .locator(".media-card:focus")
      .filter({ hasText: "Temporal Horizon" })
      .waitFor();
    await page.keyboard.press("Enter");
    await page
      .getByRole("heading", { name: "Temporal Horizon", exact: true })
      .waitFor();
    await page.locator("[data-tv-play]:focus").waitFor();
    await page.keyboard.press("Enter");
    const iframe = page.locator("iframe.player-frame");
    await iframe.waitFor();
    const source = await iframe.getAttribute("src");
    assert.equal(
      new URLSearchParams(new URL(source).hash.split("?")[1]).get("tv"),
      "1",
    );
    await page.waitForFunction(
      () => document.activeElement?.tagName === "IFRAME",
    );
    const frame = page.frameLocator("iframe.player-frame");
    const video = frame.locator("video");
    await video.evaluate(
      (element) =>
        new Promise((resolve, reject) => {
          const timeout = setTimeout(
            () => reject(new Error("TV smoke playback did not advance")),
            60_000,
          );
          const check = () => {
            if (
              element.currentTime > 1 &&
              element.videoWidth > 0 &&
              !element.paused
            ) {
              clearTimeout(timeout);
              element.removeEventListener("timeupdate", check);
              resolve();
            }
          };
          element.addEventListener("timeupdate", check);
          check();
        }),
    );
    const playback = await video.evaluate((v) => ({
      time: v.currentTime,
      width: v.videoWidth,
      height: v.videoHeight,
    }));
    // No pointer clicks or programmatic focus between entering and leaving Plarr.
    await page.keyboard.press("Enter");
    assert.equal(await video.evaluate((v) => v.paused), true);
    await page.keyboard.press("Enter");
    await video.evaluate(
      (v) =>
        new Promise((resolve) => {
          if (!v.paused) resolve();
          else v.addEventListener("play", resolve, { once: true });
        }),
    );
    await video.evaluate(() =>
      window.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Unidentified",
          keyCode: 461,
          bubbles: true,
          cancelable: true,
        }),
      ),
    );
    await iframe.waitFor({ state: "detached" });
    assert.equal(
      new URLSearchParams(new URL(page.url()).hash.split("?")[1]).has("play"),
      false,
    );
    await page.locator("[data-tv-play]:focus").waitFor();
    await page.screenshot({ path: `${artifacts}/television-return.png` });
    await page.evaluate(() =>
      window.dispatchEvent(
        new KeyboardEvent("keydown", {
          keyCode: 461,
          bubbles: true,
          cancelable: true,
        }),
      ),
    );
    await page
      .locator(".media-card:focus")
      .filter({ hasText: "Temporal Horizon" })
      .waitFor();
    assert.deepEqual(errors, []);
    return {
      result: "PASS",
      loginWithKeyboard: true,
      playback,
      remoteBack: true,
      restoredFocus: true,
      physicalWebOS: false,
    };
  } catch (error) {
    await page
      .screenshot({ path: `${artifacts}/television-failure.png` })
      .catch(() => {});
    throw error;
  } finally {
    await context.close();
  }
}
