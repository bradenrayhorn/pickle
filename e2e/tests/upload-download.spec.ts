import { expect } from '@playwright/test';
import { test } from '@tests//fixtures';
import { readFile, writeFile } from 'fs/promises';
import { join } from 'path';
import { setForcedDownloadPath, setForcedUploadPath } from './utils';

test('can upload and download', async ({ tempDir, connection, page }) => {
  const filePath = join(tempDir, 'my-file.txt');
  await writeFile(filePath, 'abc');

  // open page
  await page.goto('');
  await page.getByLabel('Connection credentials').fill(connection);
  await page.getByRole('button', { name: 'Connect', exact: true }).click();

  await setForcedUploadPath(page, filePath);

  // open upload modal
  await page.getByRole('button', { name: 'Upload' }).click();

  // in modal, actually upload
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  // now download the file back
  await page.getByRole('row').filter({ hasText: 'my-file.txt' }).click();

  const downloadPath = join(tempDir, 'out.txt');
  await setForcedDownloadPath(page, downloadPath);

  await page.getByRole('button', { name: 'Download' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Download complete' })).toBeVisible();

  expect((await readFile(downloadPath)).toString()).toEqual('abc');
});
