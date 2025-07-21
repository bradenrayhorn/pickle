import { expect } from '@playwright/test';
import { test } from '@tests//fixtures';
import { writeFile } from 'fs/promises';
import { join } from 'path';

test('can open', async ({ tempDir, connection, page }) => {
  const filePath = join(tempDir, 'my-file.txt');
  await writeFile(filePath, 'abc');

  await page.goto('');
  await page.getByLabel('Connection credentials').fill(connection);
  await page.getByRole('button', { name: 'Connect', exact: true }).click();

  await page.evaluate((path) => {
    window['__dev_pickle_forced_upload_path'] = path;
  }, filePath);

  await page.getByRole('button', { name: 'Upload' }).click();

  // in modal, actually upload
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  await expect(page.getByRole('button', { name: 'my-file.txt' })).toBeVisible();
});
