// Run with Node's test runner and Playwright on the module path; see README.md.
// A click before start.js has run (#4537): the shell installs the sidebar's
// handlers in app.js, but a plugin's page is mounted only by magpie.start,
// which start.js calls. A click held there used to run the page's open on an
// unmounted page — a null querySelector — and the page stayed broken even
// after start.js arrived. The kernel now holds the call until start mounted
// the page, then runs it in order; nothing breaks, and the page the reader
// asked for opens by itself once start.js lands.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
// sessions/manage's answer: nothing installed, nothing kept
const EMPTY_SESSIONS = { agent: "", agents: [], sessions: [], trash: [], trashDir: "/tmp/fixture-trash" };

function serve(gate) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",pluginConfig:{}};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], gateway: { running: false, calls: [], groups: [] } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/caller-keys") return json({ keys: [] });
    if (url.pathname === "/sessions/manage") return json(EMPTY_SESSIONS);
    if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/sessions/")) return json({});
    if (url.host !== "magpie.test") return route.fulfill({ status: 404, body: "" });
    if (url.pathname === "/start.js") await gate; // held: start.js is late (#4537)
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" }[path.extname(file)];
    let body;
    try { body = await fs.readFile(file); } catch { return route.fulfill({ status: 404, body: "" }); }
    await route.fulfill({ body, contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium"])) {
  test(`${engine}: a click before start.js ran waits for it, and the page opens by itself`, async (t) => {
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" });
    t.after(() => browser.close());
    let letStart;
    const gate = new Promise((resolve) => { letStart = resolve; });
    const context = await browser.newContext({ viewport: { width: 1100, height: 700 }, reducedMotion: "reduce" });
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    page.setDefaultNavigationTimeout(60000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve(gate));
    // start.js is a plain <script>: while it is held, even DOMContentLoaded
    // never fires. Commit the moment the answer starts, then wait for the
    // sidebar — exactly the reader who clicks while start.js is still late
    await page.goto("http://magpie.test/", { waitUntil: "commit" });
    await page.locator('#nav button[data-view="sessions"]').waitFor();

    // start.js is still held: the sidebar is clickable, a click must not
    // crash the page it opens — the call waits for start
    await page.locator('#nav button[data-view="sessions"]').click();
    await page.waitForTimeout(300);
    assert.deepEqual(errors, [], "no null querySelector: the click was held, not run");
    assert.equal(await page.locator("#view-sessions .slot-error").count(), 0, "the page never broke");
    assert.equal(await page.locator("#view-sessions").isVisible(), true, "the reader is on the page they asked for");

    // start.js lands: the held call runs once the page is mounted, and the
    // Sessions page draws on its own — no Try again needed
    letStart();
    await page.locator("#view-sessions .empty-state").first().waitFor();
    assert.deepEqual(errors, [], "start's replay broke nothing");
    assert.equal(await page.locator("#view-sessions .slot-error").count(), 0, "no card once it drew");

    // and it keeps working: an ordinary click after start reloads it
    await page.locator('#nav button[data-view="providers"]').click();
    await page.locator("#addProvider").waitFor();
    await page.locator('#nav button[data-view="sessions"]').click();
    await page.locator("#view-sessions .empty-state").first().waitFor();
    assert.deepEqual(errors, [], "the page goes on");
    await context.close();
  });
}
