import { Page } from "@playwright/test";

const ignored = [/\/__reload__\//];

export function watch(page: Page): string[] {
  const problems: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      problems.push(`console: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => problems.push(`page error: ${error.message}`));
  page.on("requestfailed", (request) => {
    if (!ignored.some((pattern) => pattern.test(request.url()))) {
      problems.push(`request failed: ${request.url()} ${request.failure()?.errorText ?? ""}`);
    }
  });
  page.on("response", (response) => {
    const url = new URL(response.url());
    if (response.status() >= 400 && url.origin === new URL(page.url() || response.url()).origin) {
      problems.push(`HTTP ${response.status()}: ${url.pathname}`);
    }
  });
  return problems;
}

export async function logIn(page: Page): Promise<void> {
  await page.goto("/admin/login/");
  await page.fill("#id_username", process.env.ADMIN_USER ?? "");
  await page.fill("#id_password", process.env.ADMIN_PASSWORD ?? "");
  await Promise.all([page.waitForURL((url) => !url.pathname.endsWith("/login/")), page.click("[type=submit]")]);
}
