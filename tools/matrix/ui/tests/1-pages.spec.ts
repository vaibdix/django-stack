import { expect, test } from "@playwright/test";
import { watch } from "./watch";

const maxPages = 30;
const skip = [/logout/, /\/delete\//, /__reload__/, /\.(css|js|png|svg|ico|txt|xml)$/];

function startPaths(): string[] {
  const paths = ["/", "/admin/login/"];
  if (process.env.ALLAUTH === "true") {
    paths.push("/accounts/login/", "/accounts/signup/");
  }
  if (process.env.APP) {
    paths.push(`/${process.env.APP}/`);
  }
  return paths;
}

function snapshotName(path: string): string {
  const slug = path.replace(/^\/|\/$/g, "").replace(/[^a-z0-9]+/gi, "-");
  return `${slug || "home"}.png`;
}

test("every reachable page loads cleanly", async ({ page, baseURL }) => {
  const origin = new URL(baseURL ?? "").origin;
  const queue = startPaths();
  const seen = new Set<string>();

  while (queue.length > 0 && seen.size < maxPages) {
    const path = queue.shift() ?? "/";
    if (seen.has(path) || skip.some((pattern) => pattern.test(path))) {
      continue;
    }
    seen.add(path);

    await test.step(path, async () => {
      const problems = watch(page);
      const download = page.waitForEvent("download", { timeout: 2_000 }).catch(() => null);
      const response = await page.goto(path).catch(async (error: Error) => {
        if (error.message.includes("Download is starting") && (await download)) {
          return null;
        }
        throw error;
      });
      if (response === null) {
        page.removeAllListeners();
        return;
      }
      await page.waitForLoadState("load");
      expect.soft(response.status(), `${path} status`).toBeLessThan(400);
      expect.soft(problems, `${path} console and network`).toEqual([]);
      const docs = (await page.locator("#swagger-ui").count()) > 0;
      if (docs) {
        await page.locator("#swagger-ui .info").waitFor({ timeout: 15_000 });
        await page.waitForLoadState("networkidle");
      }
      await expect.soft(page).toHaveScreenshot(snapshotName(path), { fullPage: true, timeout: docs ? 20_000 : 5_000 });

      const links = await page.locator("a[href]").evaluateAll((anchors) =>
        anchors.map((anchor) => (anchor as HTMLAnchorElement).href),
      );
      for (const link of links) {
        const url = new URL(link);
        if (url.origin === origin && !seen.has(url.pathname)) {
          queue.push(url.pathname);
        }
      }
      page.removeAllListeners();
    });
  }
  expect(seen.size).toBeGreaterThan(0);
});
