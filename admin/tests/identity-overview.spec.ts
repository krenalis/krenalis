import { expect, Page, test } from '@playwright/test';
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
	await page.clock.install({ time: new Date('2026-08-10T12:00:00Z') });
	await page.setContent('<div id="root"></div>');
	await page.addStyleTag({ content: componentStyles });
});

test('does not publish current KPIs or the breakdown when history fails', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'historyError';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(
		page.locator('.identity-dashboard__section--current-state .identity-dashboard__kpi-value').first(),
	).toHaveText('—');
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(page.getByText('Identity metrics could not be loaded', { exact: true })).toBeVisible();
	await expect(page.locator('.identity-dashboard__connections-chart')).toHaveCount(0);
});

test('shows unavailable current state when latest fails', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'latestError';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
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

test('reports an unauthorized response even without an error message', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'resolutionUnauthorized';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(page.locator('.identity-dashboard__kpi-value')).toHaveText(Array(8).fill('—'));
	await expect(page.locator('.recharts-surface')).toHaveCount(0);
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

const dashboardSnapshot = async (page: Page) => {
	await page.mouse.move(0, 0);
	// Finish chart animations before comparing data and represented ranges.
	await page.clock.runFor(2000);
	return page.locator('.identity-dashboard__section').evaluateAll((sections) =>
		sections.map((section) => ({
			text: Array.from(section.children)
				.filter((child) => child.getAttribute('role') !== 'alert')
				.map((child) => child.textContent)
				.join(''),
			paths: Array.from(section.querySelectorAll('.recharts-surface path')).map((path) => path.getAttribute('d')),
		})),
	);
};

const expectUpdatedDashboard = async (page: Page) => {
	await expect(
		page
			.locator('.identity-dashboard__kpi')
			.filter({ hasText: 'Total identities' })
			.locator('.identity-dashboard__kpi-value'),
	).toHaveText('18');
	await expect(
		page
			.locator('.identity-dashboard__kpi')
			.filter({ hasText: 'Unified profiles' })
			.locator('.identity-dashboard__kpi-value'),
	).toHaveText('6');
	await expect(page.locator('.identity-dashboard__section--current-state')).toContainText('Aug 3, 2026');
	await expect(page.locator('.identity-dashboard__history-card')).toContainText('Aug 3, 2026');
	await expect(
		page.locator('.identity-dashboard__kpi').filter({ hasText: 'Last successful identity resolution' }),
	).toContainText('Aug 3, 2026');
	await expect(page.getByRole('alert')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
};

const LOAD_ENDPOINTS = [
	'/metrics/identities/latest',
	'/metrics/identities/dates/',
	'/metrics/identity-resolution/latest',
	'/metrics/identity-resolution/dates/',
	'/identity-resolution/runs',
];

for (const endpoint of LOAD_ENDPOINTS) {
	test(`keeps the first load unpublished until all requests succeed (${endpoint})`, async ({ page }) => {
		await page.evaluate((endpoint) => {
			window.identityOverviewScenario = 'atomic';
			window.identityOverviewFailure = endpoint;
			window.identityOverviewHold = endpoint;
		}, endpoint);
		await page.addScriptTag({ content: componentScript });
		await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
		await page.clock.runFor(2000);
		await expect(page.locator('.identity-dashboard__kpi-value')).toHaveCount(0);
		await expect(page.locator('.recharts-surface')).toHaveCount(0);
		await expect(page.locator('.identity-dashboard__history-status')).toHaveCount(0);

		await page.evaluate(() => window.identityOverviewPending[0]());
		await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
		await expect(page.locator('.identity-dashboard__kpi-value')).toHaveText(Array(8).fill('—'));
		await expect(page.locator('.recharts-surface')).toHaveCount(0);
		await expect(page.locator('.identity-dashboard__history-status')).toHaveCount(0);

		await page.evaluate(() => {
			window.identityOverviewFailure = undefined;
			window.identityOverviewHold = undefined;
			window.identityOverviewRevision = 1;
		});
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expectUpdatedDashboard(page);
	});
}

for (const endpoint of [...LOAD_ENDPOINTS, '/metrics/identities/refresh', '?connection=deleted']) {
	test(`preserves the complete dashboard on refresh failure and recovers (${endpoint})`, async ({ page }) => {
		await page.evaluate(() => {
			window.identityOverviewScenario = 'atomic';
		});
		await page.addScriptTag({ content: componentScript });
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		if (endpoint === '?connection=deleted') {
			await page.locator('sl-select[aria-label="Connection"]').click();
			await page.getByRole('option', { name: 'Removed connections' }).click();
			await expect
				.poll(() =>
					page.evaluate(() =>
						window.identityOverviewRequests.some((url) => url.includes('?connection=deleted')),
					),
				)
				.toBe(true);
			await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		}
		const previous = await dashboardSnapshot(page);
		await page.evaluate((endpoint) => {
			window.identityOverviewRevision = 1;
			window.identityOverviewFailure = endpoint;
			window.identityOverviewHold = endpoint;
		}, endpoint);
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
		expect(await dashboardSnapshot(page)).toEqual(previous);
		await page.evaluate(() => window.identityOverviewPending[0]());
		await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
		expect(await dashboardSnapshot(page)).toEqual(previous);
		const alert = page.getByRole('alert');
		const alertBox = await alert.boundingBox();
		expect(alertBox).not.toBeNull();
		expect(alertBox!.height).toBeLessThan(100);

		await page.evaluate(() => {
			window.identityOverviewFailure = undefined;
			window.identityOverviewHold = undefined;
		});
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expectUpdatedDashboard(page);
		const updated = await dashboardSnapshot(page);
		for (let i = 0; i < previous.length; i++) expect(updated[i]).not.toEqual(previous[i]);
	});
}

for (const endpoint of ['/metrics/identity-resolution/dates/', '/identity-resolution/runs']) {
	test(`publishes a successful refresh together after the last response (${endpoint})`, async ({ page }) => {
		await page.evaluate(() => {
			window.identityOverviewScenario = 'atomic';
		});
		await page.addScriptTag({ content: componentScript });
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		const previous = await dashboardSnapshot(page);
		await page.evaluate((endpoint) => {
			window.identityOverviewRevision = 1;
			window.identityOverviewHold = endpoint;
		}, endpoint);
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
		expect(await dashboardSnapshot(page)).toEqual(previous);
		await page.evaluate(() => window.identityOverviewPending[0]());
		await expectUpdatedDashboard(page);
		const updated = await dashboardSnapshot(page);
		for (let i = 0; i < previous.length; i++) expect(updated[i]).not.toEqual(previous[i]);
		const freshPage = await page.context().newPage();
		await freshPage.clock.install({ time: new Date('2026-08-10T12:00:00Z') });
		await freshPage.setContent('<div id="root"></div>');
		await freshPage.addStyleTag({ content: componentStyles });
		await freshPage.evaluate(() => {
			window.identityOverviewScenario = 'atomic';
			window.identityOverviewRevision = 1;
		});
		await freshPage.addScriptTag({ content: componentScript });
		await expectUpdatedDashboard(freshPage);
		expect(updated).toEqual(await dashboardSnapshot(freshPage));
		await freshPage.close();
	});
}

for (const shouldFail of [false, true]) {
	test(`ignores superseded range responses (${shouldFail ? 'failure' : 'success'})`, async ({ page }) => {
		await page.evaluate(() => {
			window.identityOverviewScenario = 'atomic';
		});
		await page.addScriptTag({ content: componentScript });
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		const previous = await dashboardSnapshot(page);
		await page.evaluate((shouldFail) => {
			window.identityOverviewHold = '/metrics/identity-resolution/dates/';
			window.identityOverviewFailure = shouldFail ? window.identityOverviewHold : undefined;
		}, shouldFail);
		await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
		await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
		expect(await dashboardSnapshot(page)).toEqual(previous);
		await page.evaluate(() => {
			window.identityOverviewHold = undefined;
			window.identityOverviewFailure = undefined;
		});
		await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		const current = await dashboardSnapshot(page);
		expect(current).not.toEqual(previous);
		await page.evaluate(() => window.identityOverviewPending[0]());
		expect(await dashboardSnapshot(page)).toEqual(current);
		await expect(page.getByRole('alert')).toHaveCount(0);
	});
}

test('does not cache partial results from a failed initial load', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
		window.identityOverviewFailure = '/metrics/identity-resolution/dates/';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await page.evaluate(() => {
		window.identityOverviewFailure = undefined;
		window.identityOverviewRevision = 1;
	});
	await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
	await expectUpdatedDashboard(page);
	const latestRequests = await page.evaluate(() =>
		window.identityOverviewRequests.filter((url) => url.endsWith('/latest')),
	);
	expect(latestRequests).toHaveLength(4);
});

test('keeps cached snapshots and represented ranges after a failed refresh and range change', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const previous = await dashboardSnapshot(page);
	await page.evaluate(() => {
		window.identityOverviewRevision = 1;
		window.identityOverviewFailure = '/metrics/identity-resolution/dates/';
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await page.evaluate(() => {
		window.identityOverviewRevision = 0;
		window.identityOverviewFailure = undefined;
	});
	await page.getByRole('button', { name: 'Last 30 days', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await expect(page.getByRole('alert')).toHaveCount(0);
});

test('accepts empty resolution data and a zero identity snapshot as a successful refresh', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await page.evaluate(() => {
		window.identityOverviewScenario = 'singleDayZero';
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(
		page
			.locator('.identity-dashboard__kpi')
			.filter({ hasText: 'Total identities' })
			.locator('.identity-dashboard__kpi-value'),
	).toHaveText('0');
	await expect(page.getByText('No identity resolution runs yet', { exact: true })).toBeVisible();
	await expect(page.getByRole('alert')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
});

test('keeps a custom represented range when a refresh advances the latest observation', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await page.getByRole('button', { name: 'Custom range', exact: true }).click();
	const inputs = page.locator('.rdrDateInput input');
	await inputs.first().fill('Jul 1, 2026');
	await inputs.first().press('Enter');
	await inputs.nth(1).fill('Aug 2, 2026');
	await inputs.nth(1).press('Enter');
	await page.getByRole('heading', { name: 'Identity dashboard', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const axes = page.locator(
		'.identity-dashboard__chart:not(.identity-dashboard__connections-chart) .recharts-xAxis, .identity-dashboard__effectiveness-chart .recharts-xAxis',
	);
	await expect(axes).toHaveCount(3);
	const previous = await axes.allTextContents();
	await page.evaluate(() => {
		window.identityOverviewRevision = 1;
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expectUpdatedDashboard(page);
	expect(await axes.allTextContents()).toEqual(previous);
});

test('keeps the connection selector disabled until the refresh completes', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await page.evaluate(() => {
		window.identityOverviewRevision = 1;
		window.identityOverviewHold = '/metrics/identities/refresh';
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
	const select = page.locator('sl-select[aria-label="Connection"]');
	await expect(select).toHaveJSProperty('disabled', true);
	await page.evaluate(() => window.identityOverviewPending[0]());
	await expectUpdatedDashboard(page);
	await expect(select).toHaveJSProperty('disabled', false);
});

test('retries a failed connection selection without reloading the complete histories', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const previous = await dashboardSnapshot(page);
	await page.evaluate(() => {
		window.identityOverviewRequests = [];
		window.identityOverviewFailure = '?connection=deleted';
	});
	const select = page.locator('sl-select[aria-label="Connection"]');
	await select.click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(select).toHaveJSProperty('value', '');
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await page.evaluate(() => {
		window.identityOverviewFailure = undefined;
	});
	await select.click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect(select).toHaveJSProperty('value', 'deleted');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(page.getByRole('alert')).toHaveCount(0);
	const requests = await page.evaluate(() =>
		window.identityOverviewRequests.filter((url) => url.startsWith('https://example.test/')),
	);
	expect(requests).toHaveLength(2);
	for (const url of requests) expect(url).toContain('?connection=deleted');
	await select.click();
	await page.getByRole('option', { name: 'All connections', exact: true }).click();
	await expect(select).toHaveJSProperty('value', '');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	expect(await dashboardSnapshot(page)).toEqual(previous);
	expect(
		await page.evaluate(() =>
			window.identityOverviewRequests.filter((url) => url.startsWith('https://example.test/')),
		),
	).toEqual(requests);
});

test('does not publish or cache a superseded latest response', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
		window.identityOverviewHold = '/metrics/identities/latest';
	});
	await page.addScriptTag({ content: componentScript });
	await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
	await page.evaluate(() => {
		window.identityOverviewRevision = 1;
		window.identityOverviewHold = undefined;
	});
	await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
	await expectUpdatedDashboard(page);
	const current = await dashboardSnapshot(page);
	await page.evaluate(() => window.identityOverviewPending[0]());
	expect(await dashboardSnapshot(page)).toEqual(current);
	await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
	await expectUpdatedDashboard(page);
	const latestRequests = await page.evaluate(() =>
		window.identityOverviewRequests.filter((url) => url.endsWith('/latest')),
	);
	expect(latestRequests).toHaveLength(4);
});

test('clears the previous workspace and ignores its unfinished refresh after a workspace change', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const select = page.locator('sl-select[aria-label="Connection"]');
	await select.click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect(select).toHaveJSProperty('value', 'deleted');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await page.evaluate(() => {
		window.identityOverviewHold = '/metrics/identity-resolution/dates/';
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
	await page.evaluate(() => {
		window.identityOverviewHold = undefined;
		window.identityOverviewRevision = 1;
		window.identityOverviewFailure = '/metrics/identities/latest';
		window.identityOverviewChangeWorkspace('workspace222');
	});
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(page.locator('.identity-dashboard__kpi-value')).toHaveText(Array(8).fill('—'));
	await expect(select).toHaveJSProperty('value', '');
	const current = await dashboardSnapshot(page);
	await page.evaluate(() => window.identityOverviewPending[0]());
	expect(await dashboardSnapshot(page)).toEqual(current);
	await expect(page.locator('.recharts-surface')).toHaveCount(0);
	await page.evaluate(() => {
		window.identityOverviewFailure = undefined;
	});
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expectUpdatedDashboard(page);
});
