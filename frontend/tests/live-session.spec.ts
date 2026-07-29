import {expect, test} from '@playwright/test';
import path from 'node:path';

const sessionId = process.env.LIVE_SESSION_ID;
const runName = process.env.LIVE_RUN_NAME ?? 'run-1';

test('live AI session renders messages, candidates and final poster', async ({page}) => {
  test.setTimeout(180_000);
  test.skip(!sessionId, 'Set LIVE_SESSION_ID to verify a real StagePoster session');
  await page.addInitScript((id) => {
    localStorage.setItem('poster-ai-session:demo-changan', id as string);
  }, sessionId);
  const browserErrors: string[] = [];
  page.on('pageerror', (error) => browserErrors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') browserErrors.push(message.text()); });

  await page.goto('/create?demo=1');
  await expect(page.getByText('LIVE API')).toBeVisible();
  await expect(page.locator('.assistant-message.user')).toHaveCount(1, {timeout: 30_000});
  await page.getByRole('button', {name: /将 AI 已理解的信息同步到表单/}).click();
  await expect(page.locator('.candidate-grid article')).toHaveCount(3);
  await expect(page.locator('.assistant-panel').getByText('completed_with_warnings').first()).toBeVisible();
  await expect.poll(async () => page.locator('.session-publish-visual').evaluate((image) =>
    (image as HTMLImageElement).complete && (image as HTMLImageElement).naturalWidth > 0,
  ), {timeout: 90_000}).toBeTruthy();
  await expect.poll(async () => page.locator('.candidate-grid img').evaluateAll((images) =>
    images.length === 3 && images.every((image) => (image as HTMLImageElement).naturalWidth > 0),
  ), {timeout: 90_000}).toBeTruthy();
  await expect(page.locator('.session-publish-poster')).toBeVisible();
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', {name: /导出精确信息发布版/}).click();
  expect((await downloadPromise).suggestedFilename()).toMatch(/-publish\.png$/);
  await page.locator('.assistant-panel').screenshot({
    path: path.resolve(`artifacts/live-api/${runName}/frontend-live-session.png`),
    animations: 'disabled',
  });
  expect(browserErrors).toEqual([]);
});
