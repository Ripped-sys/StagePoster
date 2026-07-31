import {expect, test} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import {demoProject} from '../src/data/mock';

test('remote result composes exact information and downloads a PNG', async ({page}) => {
  const project = demoProject();
  const posterId = 'poster-browser-export';
  const candidateId = 'candidate-browser-export';
  const task = {
    id: posterId, projectId: project.id, step: 5, progress: 100, status: 'complete',
    startedAt: Date.now(), source: 'w7900', outputUrl: '',
    candidates: [{candidateId, variantKey: 'export', variantName: 'Export', status: 'ready', selected: true, attempt: 1, imageUrl: `/api/posters/${posterId}/candidates/${candidateId}/image`}],
    metrics: {gpu: 'AMD Radeon Pro W7900', rocm: 'ROCm 6.x', resolution: '1024 × 1536', duration: '12 s', peakVram: '18 GB'},
  } as const;
  const state = JSON.stringify({projects: {[project.id]: project}, tasks: {[task.id]: task}});
  await page.addInitScript((value) => localStorage.setItem('poster-mvp-state-v1', value), state);
  await page.route(`**/api/posters/${posterId}/candidates/${candidateId}/image`, async (route) => {
    await route.fulfill({status: 200, contentType: 'image/png', body: fs.readFileSync(path.resolve('public/gallery/changan-duel.png'))});
  });
  await page.goto(`/result/${project.id}`);
  await expect(page.locator('.session-publish-poster')).toBeVisible();
  await expect(page.locator('.session-publish-copy h2')).toContainText(project.titleEn ?? project.title);
  await page.getByRole('button', {name: '中文'}).click();
  await expect(page.locator('.session-publish-copy h2')).toContainText(project.title);
  const download = page.waitForEvent('download');
  await page.getByRole('button', {name: /导出 PNG/}).click();
  expect((await download).suggestedFilename()).toMatch(/\.png$/);
});
