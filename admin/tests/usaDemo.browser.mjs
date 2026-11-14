import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { WORKSPACE_ID_KEY } from '../src/constants/storage.ts';

const [baseURL, workspaceID] = process.argv.slice(2);
assert.equal(new URL(baseURL).protocol, 'http:');
assert.ok(['localhost', '127.0.0.1'].includes(new URL(baseURL).hostname));
assert.ok(workspaceID != null && workspaceID !== '');

const browser = await chromium.launch({ headless: true });
try {
	const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
	const login = await context.request.post(`${baseURL}/v1/members/login`, {
		data: { email: 'acme@krenalis.com', password: 'krenalis-password' },
	});
	assert.equal(login.status(), 200);
	await context.addInitScript(({ key, workspace }) => localStorage.setItem(key, workspace), {
		key: WORKSPACE_ID_KEY,
		workspace: workspaceID,
	});
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.goto(`${baseURL}/admin/simulated-world/create`);
	await page.getByLabel('Name', { exact: true }).fill('USA internal demo with photographs');
	await page.locator('wa-select#simulated-system-user-count').click();
	await page.locator('wa-select#simulated-system-user-count wa-option[value="10"]').click();
	await page.locator('wa-select#simulated-system-country').click();
	await page.locator('wa-select#simulated-system-country wa-option[value="US"]').click();
	const created = page.waitForResponse(
		(response) =>
			response.request().method() === 'POST' && new URL(response.url()).pathname === '/v1/simulated/systems',
	);
	await page.getByRole('button', { name: 'Create', exact: true }).click();
	const response = await created;
	assert.equal(response.status(), 200);
	assert.deepEqual(response.request().postDataJSON().countries, { US: 100 });
	const { id } = await response.json();
	await page.getByRole('heading', { name: 'Simulated system created', exact: true }).waitFor();
	await page.getByRole('button', { name: 'OK', exact: true }).click();
	await page.goto(`${baseURL}/admin/simulated-world/${id}`);
	const preview = page.getByRole('region', { name: 'United States internal demo' });
	await preview.waitFor();
	await page.waitForFunction(() => {
		const images = [...document.querySelectorAll('.usa-demo-preview img')];
		return (
			images.length === 8 &&
			images.every((image) => image.complete && image.naturalWidth === 256 && image.naturalHeight === 256)
		);
	});
	assert.equal(await preview.locator('article').count(), 8);
	assert.match(await preview.textContent(), /AI-generated photographs/);
	assert.match(await preview.textContent(), /\$[\d,]+ \/ year/);
	const photos = await preview.locator('img').evaluateAll((images) => images.map((image) => image.src));
	const directory = new URL('../test-results/usa-demo/', import.meta.url);
	await mkdir(directory, { recursive: true });
	const screenshot = fileURLToPath(new URL('usa-internal-demo.png', directory));
	await page.screenshot({ path: screenshot, fullPage: true });
	await page.reload();
	await preview.locator('article').first().waitFor();
	assert.deepEqual(await preview.locator('img').evaluateAll((images) => images.map((image) => image.src)), photos);
	assert.deepEqual(errors, []);
	process.stdout.write(
		`PASS: USA creation, eight original WebP portraits, USD facts and reload. Screenshot: ${screenshot}\n`,
	);
} finally {
	await browser.close();
}
