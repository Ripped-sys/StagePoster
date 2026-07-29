import {expect, test} from '@playwright/test';

test('AI chat starts from the form brief and returns design plans', async ({page}) => {
  test.setTimeout(90_000);
  test.skip(!process.env.RUN_LIVE_UI, 'Set RUN_LIVE_UI=1 to exercise the public AI Session from the browser');
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });

  await page.goto('/create?demo=1');
  await expect(page.getByText('LIVE API')).toBeVisible();
  await page.getByRole('button', {name: /地下金属演出/}).click();
  await expect(page.locator('.assistant-message.assistant').last()).toBeVisible({timeout: 60_000});
  await expect(page.locator('.assistant-plans article')).toHaveCount(3, {timeout: 60_000});
  await expect(page.locator('.assistant-missing')).toHaveCount(0);
  expect(errors).toEqual([]);
});
