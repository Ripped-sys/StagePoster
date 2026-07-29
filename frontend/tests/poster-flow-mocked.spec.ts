import {expect, test} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

test('form flow reaches a real-data result with a deterministic API contract', async ({page}) => {
  let polls = 0;
  await page.route('**/api/generate', async (route) => {
    await route.fulfill({status: 202, contentType: 'application/json', body: JSON.stringify({jobId: 'job-ui-contract', promptId: 'prompt-ui-contract', status: 'queued', seed: 7})});
  });
  await page.route('**/api/jobs/job-ui-contract/result', async (route) => {
    await route.fulfill({status: 200, contentType: 'image/png', body: fs.readFileSync(path.resolve('artifacts/live-api/run-2/candidate-2.png'))});
  });
  await page.route('**/api/jobs/job-ui-contract', async (route) => {
    polls += 1;
    await route.fulfill({status: 200, contentType: 'application/json', body: JSON.stringify({jobId: 'job-ui-contract', status: polls > 1 ? 'succeeded' : 'running'})});
  });

  await page.goto('/create?demo=1');
  await page.locator('.steps button').nth(2).click();
  await page.locator('.style-list>button').nth(1).click();
  await page.locator('.steps button').nth(3).click();
  await page.locator('.form-actions .button').click();
  await expect(page).toHaveURL(/\/generate\//);
  await expect(page).toHaveURL(/\/result\//, {timeout: 20_000});
  await expect(page.locator('.session-publish-copy h2')).toContainText('长安双雄');
  const [download] = await Promise.all([
    page.waitForEvent('download', {timeout: 30_000}),
    page.getByRole('button', {name: /导出 PNG/}).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/\.png$/);
});
