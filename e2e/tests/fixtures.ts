import { test as base } from '@playwright/test';
import { mkdtemp, rm } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';
import getPort from 'get-port';
import { spawn } from 'child_process';

export const test = base.extend<{ tempDir: string; connection: string }, {}>({
  tempDir: [
    async ({}, use) => {
      const tempDir = await mkdtemp(join(tmpdir(), 'playwright-test-'));

      await use(tempDir);

      await rm(tempDir, { recursive: true, force: true });
    },
    { scope: 'test' },
  ],
  connection: [
    async ({}, use) => {
      const port = await getPort();

      const fakes3 = spawn('go', ['run', '../cmd/fakes3'], {
        env: {
          ...process.env,
          FAKES3_HTTP_PORT: `${port}`,
        },
      });

      const credentials = {
        v: 1,
        d: {
          u: `127.0.0.1:${port}`,
          r: 'my-region',
          b: 'my-bucket',
          c: '',
          k: 'key-id',
          ks: 'shh',
          a: 'AGE-SECRET-KEY-1U3PMPY7ACSYU7CRZWJMW4A74LJ9874NQ8SWJDQE2JUNSVH80AKFSZMY8LV',
          l: 0,
        },
      };
      await use(Buffer.from(JSON.stringify(credentials)).toString('base64'));

      fakes3.kill('SIGKILL');
    },
    { scope: 'test' },
  ],
});
