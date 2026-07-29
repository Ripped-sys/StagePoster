import {defineConfig} from '@playwright/test';

export default defineConfig({
  testDir:'./tests', timeout:60_000, expect:{timeout:8_000}, fullyParallel:false, workers:1, reporter:'line',
  webServer:{command:'node node_modules/vite/bin/vite.js --host 127.0.0.1 --port 4173 --strictPort',url:'http://127.0.0.1:4173',reuseExistingServer:true,timeout:30_000},
  use:{baseURL:'http://127.0.0.1:4173',channel:'chrome',headless:true,viewport:{width:1440,height:1000},screenshot:'only-on-failure',trace:'retain-on-failure'}
});
