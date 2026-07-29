import {expect, test} from '@playwright/test';

test('AI conversation accepts an optional logo attachment and binds it', async ({page}) => {
  test.setTimeout(180_000);
  test.skip(!process.env.RUN_LIVE_UI, 'Set RUN_LIVE_UI=1 to exercise the public asset API from the browser');

  await page.goto('/create?demo=1');
  await page.getByText('添加附件（可选）').click();
  const logoUpload = page.locator('.assistant-upload-grid .upload').nth(1);
  await logoUpload.locator('input[type=file]').setInputFiles({
    name: 'chat-logo.svg',
    mimeType: 'image/svg+xml',
    buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" width="180" height="80"><rect width="180" height="80" fill="#111"/><text x="15" y="52" fill="#fff" font-size="32">POSTER</text></svg>'),
  });
  await expect(logoUpload).toContainText('chat-logo.svg');
  await page.getByRole('button', {name: /地下金属演出/}).click();
  await expect(page.locator('.assistant-plans article')).toHaveCount(3, {timeout: 60_000});
  const bindButton = page.getByRole('button', {name: /上传并绑定/});
  await expect(bindButton).toBeVisible();
  await bindButton.click();
  await expect(page.getByRole('button', {name: /已绑定 1 项真实素材/})).toBeVisible({timeout: 120_000});
});
