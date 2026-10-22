import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  outputDir: process.env.RESULTS_DIR ?? "test-results",
  snapshotPathTemplate: `${process.env.SNAPSHOT_DIR ?? "snapshots"}/{arg}{ext}`,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 300_000,
  use: {
    baseURL: process.env.BASE_URL,
    trace: "retain-on-failure",
  },
  expect: {
    toHaveScreenshot: { maxDiffPixelRatio: 0.01, animations: "disabled" },
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
