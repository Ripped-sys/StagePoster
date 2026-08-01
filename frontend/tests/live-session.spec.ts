import {expect, test} from '@playwright/test';
import path from 'node:path';

const runName = process.env.LIVE_RUN_NAME ?? 'run-1';

test('live AI session: full flow from form brief through plans to final poster', async ({page}) => {
  // Full end-to-end against the real backend — no LIVE_SESSION_ID needed.
  test.setTimeout(480_000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });

  // 1. Open create page with demo project; AI panel auto-loads.
  await page.goto('/create?demo=1');
  await expect(page.locator('.assistant-head')).toBeVisible({timeout: 15_000});

  // 2. Click the quick-fill scene button to trigger the AI session.
  await page.getByRole('button', {name: /地下金属演出/}).click();

  // 3. Wait for the AI to produce design plans (3 articles).
  await expect(page.locator('.assistant-plans article')).toHaveCount(3, {timeout: 120_000});

  // 4. Confirm the first plan — this starts poster generation.
  await page.getByRole('button', {name: '确认此方案'}).first().click();

  // 5. Wait for all 3 candidates to reach ready status.
  await expect(page.locator('.assistant-generation .candidate-image-button')).toHaveCount(3, {timeout: 300_000});

  // 6. Select the first candidate to trigger composition.
  await page.getByRole('button', {name: /选择这张/}).first().click();

  // 7. Wait for the session to succeed and the publish panel to appear.
  await expect(page.locator('.session-publish-poster')).toBeVisible({timeout: 300_000});

  // 8. Verify all 3 candidate images have loaded.
  await expect.poll(async () => {
    return page.locator('.candidate-grid img').evaluateAll((images) =>
      images.length === 3 && images.every((img) => (img as HTMLImageElement).naturalWidth > 0),
    );
  }, {timeout: 120_000}).toBeTruthy();

  // 9. Verify the composed result image loaded.
  // .session-publish-visual is the <img> class inside .session-publish-poster.
  await expect.poll(async () => {
    return page.locator('.session-publish-poster img.session-publish-visual').evaluate((img) =>
      (img as HTMLImageElement).complete && (img as HTMLImageElement).naturalWidth > 0,
    );
  }, {timeout: 120_000}).toBeTruthy();

  // 10. Download the publish-ready PNG.
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', {name: /导出精确信息发布版/}).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/-publish\.png$/);

  // 11. Screenshot the AI panel for the live-api artifact.
  await page.locator('.assistant-panel').screenshot({
    path: path.resolve(`artifacts/live-api/${runName}/frontend-live-session.png`),
    animations: 'disabled',
  });

  expect(errors).toEqual([]);
});
