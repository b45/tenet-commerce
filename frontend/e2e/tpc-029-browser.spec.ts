import { expect, test } from "@playwright/test";

test.describe("TPC-029 browser language and responsive smoke", () => {
  test("login is usable by keyboard", async ({ page }) => {
    await page.goto("/login");
    await expect(page).toHaveTitle(/Tenet Commerce/i);
    await expect(page.getByRole("main")).toBeVisible();
    const firstControl = page.locator("main button, main input, main select").first();
    await firstControl.focus();
    await expect(firstControl).toBeFocused();
  });

  test("locale selector changes document direction to RTL for Arabic", async ({ page }) => {
    await page.goto("/login");
    const selector = page.getByRole("combobox");
    await expect(selector).toBeVisible();
    await selector.selectOption("ar");
    await expect(page.locator("html")).toHaveAttribute("dir", "rtl");
    await expect(page.locator("html")).toHaveAttribute("lang", "ar");
    await expect(page.getByRole("main")).toBeVisible();
  });
});
