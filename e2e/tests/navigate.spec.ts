import { expect } from '@playwright/test';
import { test } from '@tests//fixtures';
import { writeFile } from 'fs/promises';
import { join } from 'path';
import { setForcedUploadPath } from './utils';
import dayjs from 'dayjs';

test('shows file stats', async ({ tempDir, connection, page }) => {
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

  // check file row
  const row = page.getByRole('row').filter({ hasText: 'my-file.txt' });

  await expect(row.getByRole('cell').nth(1)).toHaveText('my-file.txt');
  expect(
    Math.abs(dayjs(await row.getByRole('cell').nth(2).textContent()).diff(dayjs(), 'minute')),
  ).toBeLessThan(2);
  await expect(row.getByRole('cell').nth(3)).toHaveText('203 B');
});

test('can nest in folders', async ({ tempDir, connection, page }) => {
  const filePath = join(tempDir, 'my-file.txt');
  await writeFile(filePath, 'abc');

  // open page
  await page.goto('');
  await page.getByLabel('Connection credentials').fill(connection);
  await page.getByRole('button', { name: 'Connect', exact: true }).click();

  await setForcedUploadPath(page, filePath);

  // open upload modal
  await page.getByRole('button', { name: 'Upload' }).click();
  await page.getByRole('textbox').fill('dir/nested/myfile.txt');
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  // go into directory
  await page.getByRole('row').filter({ hasText: 'dir' }).click();
  await page.getByRole('row').filter({ hasText: 'nested' }).click();
  expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).toBeVisible();

  // check current path
  const tree = page.getByRole('list', { name: 'Current path' }).getByRole('listitem');
  expect(await tree.count()).toBe(3);
  await expect(tree.nth(0)).toHaveText('/');
  await expect(tree.nth(1)).toHaveText('dir/');
  await expect(tree.nth(2)).toHaveText('nested/');

  // try going up a directory
  await tree.nth(1).click();
  expect(await tree.count()).toBe(2);
  await expect(tree.nth(0)).toHaveText('/');
  await expect(tree.nth(1)).toHaveText('dir/');

  // go back to start
  await tree.nth(0).click();
  expect(await tree.count()).toBe(1);
  await expect(tree.nth(0)).toHaveText('/');
  await expect(page.getByRole('row').filter({ hasText: 'dir' })).toBeVisible();
});

test('can have multiple file versions', async ({ tempDir, connection, page }) => {
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

  // upload again
  await page.getByRole('button', { name: 'Upload' }).click();
  await page.getByRole('textbox').fill('myfile.txt');
  await page.getByRole('dialog').getByRole('button', { name: 'Upload' }).click();

  // go into file
  await page.getByRole('row').filter({ hasText: 'myfile.txt' }).click();

  // check current path
  const tree = page.getByRole('list', { name: 'Current path' }).getByRole('listitem');
  expect(await tree.count()).toBe(2);
  await expect(tree.nth(0)).toHaveText('/');
  await expect(tree.nth(1)).toHaveText('myfile.txt/');

  // try going up a directory
  const fileRows = page.getByRole('row').filter({ hasText: 'myfile.txt' });
  expect(await fileRows.count()).toBe(2);
  await expect(fileRows.nth(0).getByRole('cell').nth(1)).toHaveText('myfile.txt [version 2]');
  await expect(fileRows.nth(1).getByRole('cell').nth(1)).toHaveText('myfile.txt [version 1]');

  // go back to start
  await tree.nth(0).click();
  expect(await tree.count()).toBe(1);
  await expect(tree.nth(0)).toHaveText('/');
  expect(await fileRows.count()).toBe(1);
  await expect(page.getByRole('row').filter({ hasText: 'myfile.txt' })).toBeVisible();
});
