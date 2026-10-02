import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';

let componentScript: string;
let componentStyles: string;

test.beforeAll(async () => {
	const result = await build({
		entryPoints: [resolve(__dirname, 'identity-overview.fixture.tsx')],
		bundle: true,
		write: false,
		platform: 'browser',
		format: 'iife',
		outdir: '/tmp/identity-overview-fixture',
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	componentScript = result.outputFiles.find((file) => file.path.endsWith('.js'))!.text;
	componentStyles = result.outputFiles.find((file) => file.path.endsWith('.css'))!.text;
});

test.beforeEach(async ({ page }) => {
	await page.setContent('<div id="root"></div>');
	await page.addStyleTag({ content: componentStyles });
});

test('keeps current KPIs and the breakdown when history fails', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'historyError';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.identity-overview__kpi-value').first()).toHaveText('8');
	await expect(page.getByText('Identity history could not be loaded', { exact: true })).toBeVisible();
	await expect(page.getByText('Identity metrics could not be loaded', { exact: true })).toBeVisible();
	await expect(page.getByText('Connection metrics could not be loaded', { exact: true })).toHaveCount(0);
	await expect(page.locator('.identity-overview__connections-chart')).toBeVisible();
});

test('shows unavailable current state when latest fails', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'latestError';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByText('Current identity state is unavailable', { exact: true })).toBeVisible();
	await expect(page.locator('.identity-overview__kpi-value').first()).toHaveText('—');
});

test('loads removed connection history after refreshing the current state to zero', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'removed';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.identity-overview__kpi-value').first()).toHaveText('5');
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.locator('.identity-overview__kpi-value').first()).toHaveText('0');
	await page.locator('sl-select[aria-label="Connection"]').click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect
		.poll(async () =>
			page.evaluate(() => window.identityOverviewRequests.some((url) => url.includes('?connection=deleted'))),
		)
		.toBe(true);
	await expect(page.getByText('No identity data in this period', { exact: true })).toHaveCount(0);
	await expect(page.locator('.recharts-area').first()).toBeVisible();
});
