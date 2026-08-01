import {defineConfig} from '@playwright/test';

// Default live-env flags so the full suite runs without extra CLI vars.
process.env.RUN_LIVE_UI   ??= '1';
process.env.RUN_LIVE_E2E  ??= '1';

export default defineConfig({
  testDir:'./tests', timeout:60_000, expect:{timeout:8_000}, fullyParallel:false, workers:1, reporter:'line',
  webServer:{command:'node node_modules/vite/bin/vite.js --host 0.0.0.0 --port 4173 --strictPort',url:'http://127.0.0.1:4173',reuseExistingServer:true,timeout:30_000},
  use:{baseURL:'http://127.0.0.1:4173',channel:'chrome',headless:true,viewport:{width:1440,height:1000},screenshot:'only-on-failure',trace:'retain-on-failure',env:{RUN_LIVE_UI:'1',RUN_LIVE_E2E:'1'}}
});
