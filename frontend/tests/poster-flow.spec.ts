import {expect, test, type Page} from '@playwright/test';
import path from 'node:path';

const evidence = path.resolve('artifacts/ui-audit');
const shot = (page: Page, name: string, fullPage = true) => page.screenshot({
  path: path.join(evidence, name), fullPage, animations: 'disabled',
});

function captureErrors(page: Page) {
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });
  return errors;
}

test('desktop poster MVP remains usable with live agent integration', async ({page}) => {
  // A cold W7900/ComfyUI run can take several minutes over the public tunnel.
  test.setTimeout(480_000);
  const errors = captureErrors(page);
  await page.goto('/');
  await expect(page.locator('.hero-studio')).toBeVisible();
  await shot(page, '01-landing-hero-desktop.png', false);
  await page.locator('#scenes').scrollIntoViewIfNeeded();
  await page.waitForTimeout(700);
  await shot(page, '01b-landing-scenes-desktop.png', false);

  await page.goto('/create?demo=1');
  await expect(page.getByText('LIVE API')).toBeVisible();
  await expect(page.locator('.assistant-head')).toContainText('AI CREATIVE AGENT');
  await shot(page, '02-create-live-agent.png', false);

  const logoInput = page.locator('.bands article').first().locator('.upload').first().locator('input[type=file]');
  await logoInput.setInputFiles({
    name: 'natp-logo.svg', mimeType: 'image/svg+xml',
    buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" width="200" height="80"><rect width="200" height="80" fill="#111"/><text x="20" y="52" fill="#fff" font-size="34">NATP</text></svg>'),
  });
  await expect(page.locator('.bands article').first().locator('.upload').first()).toContainText('natp-logo.svg');
  await page.locator('.steps button').nth(2).click();
  await page.locator('.style-list>button').nth(1).click();
  await shot(page, '05-style-selection.png', false);
  await page.locator('.steps button').nth(3).click();
  await shot(page, '06-output-confirmation.png', false);
  await page.locator('.form-actions .button').click();
  await expect(page).toHaveURL(/\/generate\//);
  await shot(page, '07-generation-progress.png', false);
  await page.waitForURL(/\/result\//, {timeout: 420_000});
  await shot(page, '08-result-page.png');
  const [download] = await Promise.all([
    page.waitForEvent('download', {timeout: 30_000}),
    page.getByRole('button', {name: /导出 PNG/}).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/\.png$/);
  expect(errors).toEqual([]);
});

test('mobile landing and create have no horizontal overflow', async ({page}) => {
  const errors = captureErrors(page);
  await page.setViewportSize({width: 390, height: 844});
  await page.goto('/');
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBeTruthy();
  await page.goto('/create');
  await expect(page.getByText('LIVE API')).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBeTruthy();
  await shot(page, '10-create-mobile-live-agent.png');
  expect(errors).toEqual([]);
});

test('required fields still block generation and return to scene step', async ({page}) => {
  await page.goto('/create');
  await page.locator('.steps button').nth(3).click();
  await page.locator('.form-actions .button').click();
  await expect(page.locator('.steps button').first()).toHaveClass(/active/);
  await expect(page.locator('.field-error')).toBeVisible();
  await expect(page).toHaveURL(/\/create/);
});
