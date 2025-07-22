import { expect } from '@playwright/test';
import { test } from '@tests//fixtures';
import { writeFile } from 'fs/promises';
import { join } from 'path';
import { setForcedUploadPath } from './utils';

test('can delete files', async ({ tempDir, connection, page }) => {
  const filePath = join(tempDir, 'my-file.txt');
  await writeFile(filePath, 'abc');

  // open page
  await page.goto('');
  await page.getByLabel('Connection credentials').fill(connection);
  await page.getByRole('button', { name: 'Connect', exact: true }).click();

  await setForcedUploadPath(page, filePath);

  // open upload modal
  await page.getByRole('button', { name: 'Upload' }).click();
  await page.getByRole('textbox').fill('myfile.txt');
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  await page.getByRole('button', { name: 'Upload' }).click();
  await page.getByRole('textbox').fill('good.txt');
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  // delete file
  await page.getByRole('row').filter({ hasText: 'myfile.txt' }).click();
  await page.getByRole('button', { name: 'Delete' }).click();

  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).not.toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'good.txt' })).toBeVisible();

  // check trash bin
  await page.getByRole('button', { name: 'Trash bin' }).click();
  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'good.txt' })).not.toBeVisible();

  // back to main page
  await page.getByRole('button', { name: 'Exit trash bin' }).click();
  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).not.toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'good.txt' })).toBeVisible();

  // restore from trash
  await page.getByRole('button', { name: 'Trash bin' }).click();
  await page.getByRole('row').filter({ hasText: 'myfile.txt' }).click();
  await page.getByRole('button', { name: 'Restore' }).click();

  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).not.toBeVisible();

  await page.getByRole('button', { name: 'Exit trash bin' }).click();
  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'good.txt' })).toBeVisible();
});
