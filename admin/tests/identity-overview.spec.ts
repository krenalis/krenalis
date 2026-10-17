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
	await expect(
		page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
	).toHaveText('8');
	await expect(page.getByText('Identity history could not be loaded', { exact: true })).toBeVisible();
	await expect(page.getByText('Identity metrics could not be loaded', { exact: true })).toBeVisible();
	await expect(page.getByText('Connection metrics could not be loaded', { exact: true })).toHaveCount(0);
	await expect(page.locator('.identity-dashboard__connections-chart')).toBeVisible();
});

test('shows unavailable current state when latest fails', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'latestError';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByText('Current identity state is unavailable', { exact: true })).toBeVisible();
	await expect(
		page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
	).toHaveText('—');
});

test('hides the since-resolution comparison until identities have been observed after resolution', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'resolution';
	});
	await page.addScriptTag({ content: componentScript });
	const card = page.locator('.identity-dashboard__kpi').filter({ hasText: 'Since last resolution' });
	await expect(card.locator('.identity-dashboard__kpi-value')).toHaveText('—');
	await expect(card.locator('.identity-dashboard__sparkline')).toHaveCount(0);
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(card.locator('.identity-dashboard__kpi-value')).toHaveText('-2 identities');
	await expect(card.locator('.identity-dashboard__sparkline')).toBeVisible();
});

test('shows the since-resolution comparison for observations with equal timestamps', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'resolutionAtIdentity';
	});
	await page.addScriptTag({ content: componentScript });
	const card = page.locator('.identity-dashboard__kpi').filter({ hasText: 'Since last resolution' });
	await expect(card.locator('.identity-dashboard__kpi-value')).toHaveText('-2 identities');
	await expect(card.locator('.identity-dashboard__sparkline')).toBeVisible();
});

test('preserves resolution empty states when an unauthorized response has no error message', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'resolutionUnauthorized';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(
		page
			.locator('.identity-dashboard__resolution-chart-card')
			.getByText('No identity resolution data in this period', { exact: true }),
	).toBeVisible();
	await expect(
		page
			.locator('.identity-dashboard__composition-card')
			.getByText('Profile composition is unavailable', { exact: true }),
	).toBeVisible();
	await expect(page.getByText('Resolution metrics could not be loaded', { exact: true })).toHaveCount(0);
	await expect(page.getByText('Composition could not be loaded', { exact: true })).toHaveCount(0);
});

for (const scenario of ['singleDay', 'singleDayZero'] as const) {
	test(`shows daily identity points with only one observation (${scenario})`, async ({ page }) => {
		await page.evaluate((scenario) => {
			window.identityOverviewScenario = scenario;
		}, scenario);
		await page.addScriptTag({ content: componentScript });
		await expect(
			page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
		).toHaveText(scenario === 'singleDay' ? '10' : '0');
		const areas = page.locator(
			'.identity-dashboard__section--current-state .identity-dashboard__chart .recharts-area',
		);
		await expect(areas).toHaveCount(2);
		for (const area of await areas.all()) {
			const dots = area.locator('.recharts-area-dot');
			await expect(dots).toHaveCount(1);
			await expect(dots).toBeVisible();
			await expect(dots).toHaveAttribute('r', '3');
		}
	});
}

test('loads removed connection history after refreshing the current state to zero', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'removed';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(
		page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
	).toHaveText('5');
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(
		page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
	).toHaveText('0');
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
