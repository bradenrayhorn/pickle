import { Page } from '@playwright/test';

export async function setForcedUploadPath(page: Page, filePath: string) {
  await page.evaluate((path) => {
    window['__dev_pickle_forced_upload_path'] = path;
  }, filePath);
}

export async function setForcedDownloadPath(page: Page, filePath: string) {
  await page.evaluate((path) => {
    window['__dev_pickle_forced_download_path'] = path;
  }, filePath);
}
