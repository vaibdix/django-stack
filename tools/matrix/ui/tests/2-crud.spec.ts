import { expect, test } from "@playwright/test";
import { logIn, watch } from "./watch";

const app = process.env.APP ?? "";
const stamp = Date.now().toString().slice(-9);
const title = `Matrix Sweep Book ${stamp}`;
const renamed = `${title}, Revised`;

function isbn13(): string {
  const digits = `978${stamp}`;
  const sum = [...digits].reduce((total, digit, index) => total + Number(digit) * (index % 2 === 0 ? 1 : 3), 0);
  return `${digits}${(10 - (sum % 10)) % 10}`;
}

test.skip(!app, "this project has no app pages");

test("a book can be created, edited and deleted", async ({ page }) => {
  await logIn(page);
  const problems = watch(page);

  await page.goto(`/${app}/create/`);
  if (process.env.HTMX === "true") {
    expect(await page.evaluate(() => "htmx" in window)).toBe(true);
  }
  await page.fill("[name=title]", title);
  await page.fill("[name=author]", "Matrix");
  await page.fill("[name=isbn]", isbn13());
  await page.fill("[name=publication_date]", "2020-01-01");
  await page.fill("[name=price]", "12.50");
  await page.locator("form [type=submit]").last().click();
  await expect(page.getByText(title).first()).toBeVisible();

  await page.goto(`/${app}/?q=${encodeURIComponent(title)}`);
  await page.locator(`a[href*="/edit/"]`).first().click();
  await page.fill("[name=title]", renamed);
  await page.locator("form [type=submit]").last().click();
  await expect(page.getByText(renamed).first()).toBeVisible();

  await page.goto(`/${app}/?q=${encodeURIComponent(renamed)}`);
  const htmxDelete = page.locator(`[hx-post*="/delete/"]`);
  if ((await htmxDelete.count()) > 0) {
    await page.evaluate(() => Object.assign(window, { matrixMarker: true }));
    page.once("dialog", (dialog) => dialog.accept());
    await htmxDelete.first().click();
    await expect(page.locator(`a[href*="/edit/"]`)).toHaveCount(0);
    expect(await page.evaluate(() => "matrixMarker" in window), "the delete was swapped in place").toBe(true);
  } else {
    await page.locator(`a[href*="/delete/"]`).first().click();
    await page.locator("form [type=submit]").last().click();
  }
  await page.goto(`/${app}/?q=${encodeURIComponent(renamed)}`);
  await expect(page.locator(`a[href*="/edit/"]`)).toHaveCount(0);

  expect(problems).toEqual([]);
});
