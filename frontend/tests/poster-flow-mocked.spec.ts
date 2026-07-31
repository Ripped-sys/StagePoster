import {expect, test} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

test('form flow reaches a real-data result with the current poster contract', async ({page}) => {
  const posterId = 'poster-ui-contract';
  const candidateId = 'candidate-ui-contract';
  const image = fs.readFileSync(path.resolve('public/gallery/changan-duel.png'));
  const candidate = {
    candidateId, variantKey: 'contract-balanced', variantName: 'Contract · Balanced',
    status: 'ready', attempt: 1, selected: false,
    imageUrl: `/api/posters/${posterId}/candidates/${candidateId}/image`,
  };
  let polls = 0;

  await page.route('**/api/posters', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback();
    await route.fulfill({status: 202, contentType: 'application/json', body: JSON.stringify({
      posterId, status: 'generating_candidates', progress: {completed: 0, total: 1, stage: 'generating_candidates', percent: 20}, candidates: [{...candidate, status: 'generating'}],
    })});
  });
  await page.route(`**/api/posters/${posterId}/candidates/${candidateId}/image`, async (route) => {
    await route.fulfill({status: 200, contentType: 'image/png', body: image});
  });
  await page.route(`**/api/posters/${posterId}/select`, async (route) => {
    await route.fulfill({status: 200, contentType: 'application/json', body: JSON.stringify({
      posterId, status: 'succeeded', resultUrl: `/api/posters/${posterId}/result`,
      progress: {completed: 1, total: 1, stage: 'succeeded', percent: 100, elapsedSeconds: 12},
      candidates: [{...candidate, selected: true}],
    })});
  });
  await page.route(`**/api/posters/${posterId}`, async (route) => {
    polls += 1;
    await route.fulfill({status: 200, contentType: 'application/json', body: JSON.stringify({
      posterId, status: polls > 1 ? 'awaiting_selection' : 'partial_ready',
      progress: {completed: 1, total: 1, stage: polls > 1 ? 'awaiting_selection' : 'partial_ready', percent: polls > 1 ? 65 : 60},
      candidates: [candidate],
    })});
  });

  await page.goto('/create?demo=1');
  await page.locator('.steps button').nth(2).click();
  await page.locator('.style-list>button').nth(1).click();
  await page.locator('.steps button').nth(3).click();
  await page.locator('.form-actions .button').click();
  await expect(page).toHaveURL(new RegExp(`/generate/${posterId}`));
  await expect(page.getByRole('button', {name: '选择这张并合成'})).toBeVisible({timeout: 20_000});
  await page.getByRole('button', {name: '选择这张并合成'}).click();
  await expect(page).toHaveURL(/\/result\//, {timeout: 20_000});
  await expect(page.locator('.session-publish-copy h2')).toContainText("CHANG'AN DUEL");
  await page.getByRole('button', {name: '中文'}).click();
  await expect(page.locator('.session-publish-copy h2')).toContainText('长安双雄');
  const [download] = await Promise.all([
    page.waitForEvent('download', {timeout: 30_000}),
    page.getByRole('button', {name: /导出 PNG/}).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/\.png$/);
});
