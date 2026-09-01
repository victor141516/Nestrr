import assert from "node:assert/strict";
import { mkdir, stat } from "node:fs/promises";
import { chromium } from "playwright";

const nestrr = process.env.NESTRR_URL || "http://nestrr:8080";
const jackett = process.env.JACKETT_MOCK_URL || "http://mock-jackett:9117";
const qb = process.env.QBITTORRENT_MOCK_URL || "http://mock-qbittorrent:8080";
const title = "Temporal Horizon";
const release = "Temporal.Horizon.2026.1080p.WEB-DL.VP9-E2E";
const sourcePath =
  "/runtime/data/downloads/completed/Movies/Temporal.Horizon.2026.1080p.WEB-DL.VP9-E2E.webm";
const libraryPath = "/runtime/data/library/Movies/Temporal Horizon (2026).webm";
const artifacts = "/runtime/artifacts";

async function waitForTitle(locator, expected, timeout = 5_000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if ((await locator.getAttribute("title")) === expected) return;
    await new Promise((resolve) => setTimeout(resolve, 150));
  }
  assert.fail(`Timed out waiting for torrent state ${expected}`);
}

await mkdir(artifacts, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  locale: "en-US",
  viewport: { width: 1440, height: 900 },
});
const page = await context.newPage();
const browserErrors = [];
const plarrRequests = [];
const discoverRequests = [];
page.on("pageerror", (error) => browserErrors.push(error.message));
page.on("request", (request) => {
  if (request.url().includes("/hls/")) plarrRequests.push(request.url());
  if (request.url().includes("/catalog/discover/movie?"))
    discoverRequests.push(request.url());
});
page.on("console", (message) => {
  if (message.type() === "error") browserErrors.push(message.text());
});

try {
  console.log("[e2e] Bootstrapping a fresh Nestrr instance");
  await page.goto(nestrr, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Create administrator" }).waitFor();
  await page.locator('input[name="username"]').fill("e2e-admin");
  await page.locator('input[name="password"]').fill("e2e-password-2026");
  await page.getByRole("button", { name: "Create administrator" }).click();

  console.log("[e2e] Verifying the three mock movies rendered from mock TMDB");
  await page.getByRole("heading", { name: title }).waitFor();
  for (const mockTitle of [title, "Jurassic Park", "Alien"]) {
    await page.getByText(mockTitle, { exact: true }).first().waitFor();
  }

  console.log(
    "[e2e] Verifying RouterLink navigation and combining URL-backed filters",
  );
  assert.match(
    await page.getByRole("link", { name: "Nestrr home" }).getAttribute("href"),
    /#\/$/,
  );
  assert.match(
    await page
      .getByRole("link", { name: /Explore/ })
      .first()
      .getAttribute("href"),
    /#\/explore$/,
  );
  assert.match(
    await page.locator(".media-card").first().getAttribute("href"),
    /#\/movie\/\d+$/,
  );
  await page
    .getByRole("link", { name: /Explore/ })
    .first()
    .click();
  await page.getByRole("button", { name: /Filters/ }).click();
  const filters = page.locator(".movie-filter-panel");
  await filters.waitFor();
  await filters
    .getByRole("slider", { name: "Release year · Minimum", exact: true })
    .fill("1990");
  await filters
    .getByRole("slider", { name: "Release year · Maximum", exact: true })
    .fill("2030");
  await filters.getByText("1990 – 2030", { exact: true }).waitFor();
  await filters
    .getByRole("slider", { name: "Rating · Minimum", exact: true })
    .fill("7");
  await filters
    .getByRole("slider", { name: "Rating · Maximum", exact: true })
    .fill("9");
  await filters
    .getByRole("slider", { name: "Runtime · Minimum", exact: true })
    .fill("80");
  await filters
    .getByRole("slider", { name: "Runtime · Maximum", exact: true })
    .fill("180");
  await filters
    .getByRole("slider", { name: "Age rating · Minimum", exact: true })
    .fill("10");
  await filters
    .getByRole("slider", { name: "Age rating · Maximum", exact: true })
    .fill("16");
  await filters.getByText("10 years – 16 years", { exact: true }).waitFor();
  const countryTrigger = filters.locator(".country-select-trigger");
  await countryTrigger.click();
  assert(
    (await filters.locator(".country-select-menu button").count()) <= 15,
    "production-country filter exposes more than 15 choices",
  );
  await filters
    .getByRole("button", { name: "United States", exact: true })
    .click();
  await filters.getByRole("button", { name: "France", exact: true }).click();
  await filters.getByRole("button", { name: "East Asia", exact: true }).click();
  assert.equal(
    await countryTrigger.getAttribute("aria-expanded"),
    "true",
    "country multiselect closed after a selection",
  );
  await filters.getByRole("button", { name: "Neutral: Adventure" }).focus();
  assert.equal(
    await countryTrigger.getAttribute("aria-expanded"),
    "false",
    "country multiselect did not close after focus moved outside",
  );
  await filters.getByRole("button", { name: "Neutral: Adventure" }).click();
  await filters
    .getByRole("button", { name: "Neutral: Science Fiction" })
    .click();
  await filters
    .getByRole("button", { name: "Included: Science Fiction" })
    .click();
  await filters.getByRole("button", { name: /Show results/ }).click();
  await page.getByText("Temporal Horizon", { exact: true }).first().waitFor();
  await page.getByText("Jurassic Park", { exact: true }).first().waitFor();
  assert.equal(
    await page.getByText("Alien", { exact: true }).count(),
    0,
    "the R-rated movie should be outside the 10–16 age range",
  );
  const filteredRequest = new URL(discoverRequests.at(-1));
  assert.equal(
    filteredRequest.searchParams.get("primary_release_date.gte"),
    "1990-01-01",
  );
  assert.equal(
    filteredRequest.searchParams.get("primary_release_date.lte"),
    "2030-12-31",
  );
  assert.equal(filteredRequest.searchParams.get("vote_average.gte"), "7");
  assert.equal(filteredRequest.searchParams.get("vote_average.lte"), "9");
  assert.equal(filteredRequest.searchParams.get("with_runtime.gte"), "80");
  assert.equal(filteredRequest.searchParams.get("with_runtime.lte"), "180");
  assert.equal(filteredRequest.searchParams.get("with_genres"), "12");
  assert.equal(filteredRequest.searchParams.get("without_genres"), "878");
  assert.equal(
    filteredRequest.searchParams.get("with_origin_country"),
    "US|FR|HK|KP|MN|MO|TW",
  );
  assert.equal(filteredRequest.searchParams.get("certification_country"), null);
  const exploreURL = new URL(page.url());
  const exploreParams = new URLSearchParams(exploreURL.hash.split("?")[1]);
  assert.equal(exploreParams.get("yearMin"), "1990");
  assert.equal(exploreParams.get("ageMin"), "10");
  assert.equal(exploreParams.get("ageMax"), "16");
  assert.equal(exploreParams.get("includeGenres"), "12");
  assert.equal(exploreParams.get("excludeGenres"), "878");

  console.log("[e2e] Reloading the URL and restoring every filter");
  await page.reload({ waitUntil: "domcontentloaded" });
  await page.getByText("Temporal Horizon", { exact: true }).first().waitFor();
  await page.getByRole("button", { name: /Filters/ }).click();
  await filters
    .getByRole("slider", { name: "Release year · Minimum", exact: true })
    .waitFor();
  assert.equal(
    await filters
      .getByRole("slider", { name: "Release year · Minimum", exact: true })
      .inputValue(),
    "1990",
  );
  assert.equal(
    await filters
      .getByRole("slider", { name: "Age rating · Maximum", exact: true })
      .inputValue(),
    "16",
  );
  await filters.locator(".close").click();

  console.log(
    "[e2e] Verifying the filter panel at the 390px mobile breakpoint",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: /Filters/ }).click();
  await filters.waitFor();
  const mobileLayout = await page.evaluate(() => ({
    viewport: window.innerWidth,
    documentWidth: document.documentElement.scrollWidth,
  }));
  assert(
    mobileLayout.documentWidth <= mobileLayout.viewport,
    `movie filters overflow horizontally: ${JSON.stringify(mobileLayout)}`,
  );
  await filters
    .getByRole("button", { name: /Show results/ })
    .waitFor({ state: "visible" });
  await filters.locator(".close").click();
  await page.setViewportSize({ width: 1440, height: 900 });

  console.log(
    "[e2e] Switching to TV filters without leaking movie genre IDs or runtime",
  );
  await page.getByRole("button", { name: "Series", exact: true }).click();
  await page.waitForURL(/type=tv/);
  const tvExploreParams = new URLSearchParams(
    new URL(page.url()).hash.split("?")[1],
  );
  assert.equal(tvExploreParams.get("includeGenres"), null);
  assert.equal(tvExploreParams.get("excludeGenres"), null);
  assert.equal(tvExploreParams.get("runtimeMin"), null);
  assert.equal(tvExploreParams.get("runtimeMax"), null);
  await page.getByRole("button", { name: /Filters/ }).click();
  assert.equal(
    await filters.getByRole("group", { name: "Runtime" }).count(),
    0,
  );
  await filters.locator(".close").click();

  await page.getByRole("link", { name: /Home/ }).first().click();
  await page.getByRole("heading", { name: title }).waitFor();

  await page.getByRole("link", { name: "View details" }).click();
  await page.getByRole("heading", { name: title }).waitFor();
  await page
    .locator(".facts")
    .getByText("Not downloaded", { exact: false })
    .waitFor();

  console.log(
    "[e2e] Searching mock Jackett and submitting its magnet to mock qBittorrent",
  );
  await page.getByRole("button", { name: /Search torrents/ }).click();
  const torrentModal = page.locator(".torrent-modal");
  assert.equal(
    await torrentModal.locator(".search-input input").inputValue(),
    title,
    "torrent query should use the localized catalog title",
  );
  assert.equal(await torrentModal.locator("select.category-input").count(), 0);
  assert.equal(
    await torrentModal.getByText("JACKETT / QBITTORRENT").count(),
    0,
  );
  const torrentFilters = torrentModal.locator(".torrent-filters");
  assert.equal(await torrentFilters.isVisible(), true);
  assert.equal(
    (await torrentFilters.locator("summary").innerText()).trim(),
    "Filters",
  );
  await torrentFilters.locator("summary").click();
  const includeTerms = torrentFilters.getByRole("textbox", {
    name: /Include words/,
  });
  await includeTerms.fill("Temporal.Horizon");
  await torrentFilters
    .getByRole("spinbutton", { name: /Minimum seeds/ })
    .click();
  await includeTerms.fill("2026");
  await includeTerms.press("Tab");
  const excludeTerms = torrentFilters.getByRole("textbox", {
    name: /Exclude words/,
  });
  await excludeTerms.fill("CAM");
  await excludeTerms.press("Tab");
  assert.deepEqual(
    JSON.parse(
      await torrentFilters.locator("#torrent-include").inputValue(),
    ).map((tag) => tag.value),
    ["Temporal.Horizon", "2026"],
  );
  assert.deepEqual(
    JSON.parse(
      await torrentFilters.locator("#torrent-exclude").inputValue(),
    ).map((tag) => tag.value),
    ["CAM"],
  );
  await torrentModal
    .getByRole("button", { name: "Search", exact: true })
    .click();
  await torrentModal.getByText(release, { exact: true }).waitFor();
  const firstTorrent = torrentModal.locator(".torrent-results article").first();
  const primarySend = firstTorrent.getByRole("button", {
    name: "Send to Movies",
    exact: true,
  });
  await primarySend.waitFor();
  const categoryMenu = firstTorrent.locator(".torrent-send-more > summary");
  assert.equal(
    await categoryMenu.getAttribute("aria-label"),
    "Send to the other qBittorrent category",
  );
  await categoryMenu.click();
  await firstTorrent
    .getByRole("button", { name: "Send to Shows", exact: true })
    .waitFor();
  await primarySend.click();
  await page
    .locator(".toast")
    .getByText(/Added to qBittorrent/)
    .waitFor();
  assert.equal(
    await torrentModal.getByText(/Added to qBittorrent/).count(),
    0,
    "the redundant torrent confirmation must not be rendered inside the modal",
  );
  const activeDownload = page.locator(".detail-actions [role=progressbar]");
  await activeDownload.waitFor({ timeout: 5_000 });
  assert.equal(
    await activeDownload.getAttribute("title"),
    "Downloading",
    "the selected qBittorrent state should be shown in English",
  );
  assert.equal(
    (await activeDownload.innerText()).trim(),
    "5%",
    "the qBittorrent progress should be shown numerically",
  );
  await torrentModal.locator(".close").click();
  await fetch(`${qb}/mock/state`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ state: "stalledDL", progress: 0.21 }),
  });
  await waitForTitle(activeDownload, "Stalled");
  assert.equal((await activeDownload.innerText()).trim(), "21%");
  await page
    .getByRole("button", { name: "Force download", exact: true })
    .click();
  await waitForTitle(activeDownload, "Forced downloading");

  const deadline = Date.now() + 45_000;
  let organized = false;
  console.log(
    "[e2e] Waiting for qBittorrent -> real video -> Chiprr hard link",
  );
  while (Date.now() < deadline) {
    try {
      const [source, library] = await Promise.all([
        stat(sourcePath),
        stat(libraryPath),
      ]);
      assert(source.size > 10_000, "the generated video is unexpectedly small");
      assert.equal(
        source.ino,
        library.ino,
        "Chiprr did not create a hard link",
      );
      organized = true;
      break;
    } catch {
      await page.waitForTimeout(500);
    }
  }
  assert(
    organized,
    "Chiprr did not organize the completed download before timeout",
  );

  console.log("[e2e] Reloading Nestrr's filesystem-backed library");
  await page
    .getByRole("link", { name: /Downloaded/ })
    .first()
    .click();
  await page
    .locator(".library-list a", { hasText: title })
    .waitFor({ timeout: 15_000 });
  await page.locator(".library-list a", { hasText: title }).click();
  await page.getByRole("button", { name: /Play/ }).waitFor({ timeout: 15_000 });

  console.log("[e2e] Opening Plarr and proving the real video clock advances");
  await page.getByRole("button", { name: /Play/ }).click();
  let frame = page.frameLocator("iframe.player-frame");
  await frame.locator("video").waitFor({ state: "attached", timeout: 15_000 });
  const playbackURL = new URL(page.url());
  assert.equal(
    new URLSearchParams(playbackURL.hash.split("?")[1]).get("play"),
    "1",
    "opening the player did not persist playback in the URL",
  );
  assert.equal(await page.title(), `${title} · Nestrr`);

  console.log("[e2e] Reloading the playback URL and restoring Plarr");
  await page.reload({ waitUntil: "domcontentloaded" });
  frame = page.frameLocator("iframe.player-frame");
  await frame.locator("video").waitFor({ state: "attached", timeout: 15_000 });
  await frame.getByText(title, { exact: true }).waitFor();
  assert.equal(await page.title(), `${title} · Nestrr`);
  assert.equal(
    new URLSearchParams(new URL(page.url()).hash.split("?")[1]).get("play"),
    "1",
    "the restored player lost its playback URL state",
  );

  console.log("[e2e] Closing Plarr and removing playback from the URL");
  await frame.getByRole("button", { name: "Back" }).click();
  await page.locator("iframe.player-frame").waitFor({ state: "detached" });
  assert.equal(
    new URLSearchParams(new URL(page.url()).hash.split("?")[1]).get("play"),
    null,
    "closing the player left playback enabled in the URL",
  );
  assert.equal(await page.title(), `${title} · Nestrr`);
  await page.getByRole("button", { name: /Play/ }).click();
  frame = page.frameLocator("iframe.player-frame");
  const video = frame.locator("video");
  await video.waitFor({ state: "attached", timeout: 15_000 });
  await frame.getByText(title, { exact: true }).waitFor();
  assert.equal(
    await page.locator(".player-close").count(),
    0,
    "Nestrr still renders a redundant close button over Plarr",
  );
  assert.equal(
    await frame.getByText("Download original", { exact: true }).count(),
    0,
    "Plarr still renders the removed header download",
  );
  const iconButtons = frame.locator("button.icon-button");
  assert.equal(
    await iconButtons.count(),
    await iconButtons.locator("svg").count(),
    "an icon-only player button does not contain a real SVG icon",
  );
  for (let index = 0; index < (await iconButtons.count()); index++) {
    const box = await iconButtons.nth(index).boundingBox();
    assert(box, `icon button ${index} has no rendered box`);
    assert(
      Math.abs(box.width - box.height) < 0.5,
      `icon button ${index} is not square: ${box.width}x${box.height}`,
    );
  }
  await frame.getByRole("button", { name: "Audio tracks" }).click();
  await frame.getByText("Language: Not declared · Codec: OPUS").waitFor();
  await frame.getByRole("button", { name: "Audio tracks" }).click();
  await frame
    .locator(".big-play")
    .waitFor({ state: "visible", timeout: 60_000 });
  await frame.locator(".big-play").click();
  await video.evaluate(
    (element) =>
      new Promise((resolve, reject) => {
        const deadline = Date.now() + 12_000;
        const check = () => {
          if (element.error)
            return reject(
              new Error(
                element.error.message || `media error ${element.error.code}`,
              ),
            );
          if (
            element.currentTime > 1.5 &&
            element.videoWidth > 0 &&
            element.duration > 4
          )
            return resolve();
          if (Date.now() > deadline)
            return reject(
              new Error(
                `video did not advance: time=${element.currentTime}, ready=${element.readyState}, duration=${element.duration}`,
              ),
            );
          setTimeout(check, 200);
        };
        check();
      }),
  );
  const playback = await video.evaluate((element) => ({
    currentTime: element.currentTime,
    duration: element.duration,
    width: element.videoWidth,
    height: element.videoHeight,
    paused: element.paused,
    readyState: element.readyState,
  }));
  await page.screenshot({ path: `${artifacts}/playback.png`, fullPage: true });

  const playerSource = await page
    .locator("iframe.player-frame")
    .getAttribute("src");
  const encodedMediaId = playerSource?.match(/#\/play\/([^?]+)/)?.[1];
  assert(
    encodedMediaId,
    `could not extract media id from Plarr URL: ${playerSource}`,
  );
  const plarrMedia = `http://plarr:8090/api/v1/media/${encodeURIComponent(encodedMediaId)}`;
  const master = await fetch(`${plarrMedia}/hls/master.m3u8`).then((response) =>
    response.text(),
  );
  assert(
    master.indexOf("240p/index.m3u8") < master.indexOf("360p/index.m3u8"),
    "the HLS master does not put the smallest rendition first",
  );
  const lowPlaylist = await fetch(`${plarrMedia}/hls/240p/index.m3u8`).then(
    (response) => response.text(),
  );
  const lowSegment = lowPlaylist.match(/segment-\d+\.ts/)?.[0];
  assert(
    lowSegment,
    `the 240p rendition did not publish a real segment: ${lowPlaylist}`,
  );
  const lowSegmentBytes = await fetch(
    `${plarrMedia}/hls/240p/${lowSegment}`,
  ).then((response) => response.arrayBuffer());
  assert(
    lowSegmentBytes.byteLength > 1_000,
    `the 240p media segment is unexpectedly small: ${lowSegmentBytes.byteLength}`,
  );

  console.log(
    "[e2e] Proving a transcoded MP4 download streams before FFmpeg finishes",
  );
  const downloadStarted = Date.now();
  const downloadResponse = await fetch(`${plarrMedia}/download/240p`);
  const headersAfterMs = Date.now() - downloadStarted;
  assert.equal(downloadResponse.status, 200);
  assert.equal(
    downloadResponse.headers.get("content-length"),
    null,
    "streaming download unexpectedly has a precomputed length",
  );
  assert.equal(downloadResponse.headers.get("x-accel-buffering"), "no");
  const reader = downloadResponse.body.getReader();
  const first = await reader.read();
  const firstBytesAfterMs = Date.now() - downloadStarted;
  assert.equal(first.done, false, "streaming download returned no bytes");
  let downloadedBytes = first.value.byteLength;
  let chunks = 1;
  while (true) {
    const chunk = await reader.read();
    if (chunk.done) break;
    downloadedBytes += chunk.value.byteLength;
    chunks++;
  }
  const completedAfterMs = Date.now() - downloadStarted;
  assert(
    downloadedBytes > 10_000,
    `transcoded download is unexpectedly small: ${downloadedBytes}`,
  );
  assert(
    chunks > 1,
    "the complete transcoded MP4 was buffered into a single response chunk",
  );
  assert(
    firstBytesAfterMs <= completedAfterMs,
    "download completion preceded its first bytes",
  );

  const [jackettState, qbState] = await Promise.all([
    fetch(`${jackett}/mock/state`).then((response) => response.json()),
    fetch(`${qb}/mock/state`).then((response) => response.json()),
  ]);
  assert(jackettState.searches >= 1, "mock Jackett did not receive a search");
  assert.equal(
    qbState.accepted,
    1,
    "mock qBittorrent did not receive exactly one torrent",
  );
  assert.equal(
    qbState.generated,
    true,
    qbState.error || "mock qBittorrent did not generate the video",
  );
  assert.equal(qbState.category, "Movies");

  const relevantErrors = browserErrors.filter(
    (message) => !message.includes("favicon"),
  );
  assert.deepEqual(
    relevantErrors,
    [],
    `browser errors: ${relevantErrors.join(" | ")}`,
  );
  console.log(
    JSON.stringify(
      {
        result: "PASS",
        moviesRendered: 3,
        jackettSearches: jackettState.searches,
        qbittorrentAccepted: qbState.accepted,
        generatedVideo: sourcePath,
        organizedVideo: libraryPath,
        playback,
        adaptiveStart: "240p",
        streamingDownload: {
          headersAfterMs,
          firstBytesAfterMs,
          completedAfterMs,
          downloadedBytes,
          chunks,
        },
        hardwareAcceleration: "cpu",
      },
      null,
      2,
    ),
  );
} catch (error) {
  await page
    .screenshot({ path: `${artifacts}/failure.png`, fullPage: true })
    .catch(() => {});
  console.error("[e2e] FAILED", error);
  if (browserErrors.length)
    console.error("[e2e] Browser errors", browserErrors);
  process.exitCode = 1;
} finally {
  await browser.close();
}
