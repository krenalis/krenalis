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

test('filters connection series with mouse and keyboard and updates the chart', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'connectionTypes';
	});
	await page.addScriptTag({ content: componentScript });
	const chart = page.locator('.identity-dashboard__connections-chart');
	const recognized = chart.getByRole('button', { name: 'Recognized', exact: true });
	const anonymous = chart.getByRole('button', { name: 'Anonymous', exact: true });
	const totals = chart.locator('.recharts-label-list text');
	const shares = chart.locator('.identity-dashboard__connection-axis-share');
	const ticks = chart.locator('.recharts-xAxis .recharts-cartesian-axis-tick-value');
	await expect(recognized).toHaveAttribute('aria-pressed', 'true');
	await expect(anonymous).toHaveAttribute('aria-pressed', 'true');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toHaveCount(0);
	await expect(totals).toHaveText(['1,000', '20']);
	await expect(shares).toHaveText(['98%', '2%']);
	await expect(ticks.last()).toHaveText('1K');
	await anonymous.click();
	await expect(anonymous).toHaveAttribute('aria-pressed', 'false');
	await expect(recognized).toBeEnabled();
	await expect(chart.locator('.recharts-bar')).toHaveCount(1);
	await expect(totals).toHaveText(['10', '20']);
	await expect(shares).toHaveText(['33%', '67%']);
	await expect(ticks.last()).toHaveText('20');
	await chart.locator('.recharts-bar-rectangle').first().hover();
	const tooltip = chart.locator('.identity-dashboard__chart-tooltip');
	await expect(tooltip.locator('dt')).toHaveText(['Total shown', 'Recognized']);
	await expect(tooltip.locator('dd')).toHaveText(['10', '10 (100.00%)']);
	await recognized.press('Enter');
	await expect(recognized).toHaveAttribute('aria-pressed', 'false');
	await expect(anonymous).toHaveAttribute('aria-pressed', 'false');
	await expect(recognized).toBeEnabled();
	await expect(anonymous).toBeEnabled();
	await expect(chart.locator('.recharts-bar')).toHaveCount(0);
	await expect(totals).toHaveCount(0);
	await expect(ticks).toHaveText(['0', '1']);
	await expect(shares).toHaveText(['—', '—']);
	await expect(tooltip).toHaveCount(0);
	await anonymous.press('Space');
	await expect(anonymous).toHaveAttribute('aria-pressed', 'true');
	await expect(totals).toHaveText(['990', '0']);
	await expect(shares).toHaveText(['100%', '0%']);
	await chart.locator('.recharts-bar-rectangle').first().hover();
	await expect(tooltip.locator('dt')).toHaveText(['Total shown', 'Anonymous']);
	await expect(tooltip.locator('dd')).toHaveText(['990', '990 (100.00%)']);
	await recognized.click();
	await expect(chart.locator('.recharts-bar')).toHaveCount(2);
	await expect(totals).toHaveText(['1,000', '20']);
	await expect(shares).toHaveText(['98%', '2%']);
	await expect(ticks.last()).toHaveText('1K');
	await expect(anonymous).toBeEnabled();
	await expect(
		page
			.locator('.identity-dashboard__chart:not(.identity-dashboard__connections-chart)')
			.getByRole('button', { pressed: true }),
	).toHaveCount(2);
});

for (const [title, maximum, anonymousMaximum] of [
	['Identities over time (daily)', '1K', '990'],
	['Profiles over time (daily)', '100', '90'],
] as const) {
	test(`filters ${title} independently and rescales its visible series`, async ({ page }) => {
		await page.evaluate(() => {
			window.identityOverviewScenario = 'legendFilters';
		});
		await page.addScriptTag({ content: componentScript });
		const card = page.locator('.identity-dashboard__chart-card').filter({ hasText: title });
		const recognized = card.getByRole('button', { name: 'Recognized', exact: true });
		const anonymous = card.getByRole('button', { name: 'Anonymous', exact: true });
		const ticks = card.locator('.recharts-yAxis .recharts-cartesian-axis-tick-value');
		const tooltip = card.locator('.identity-dashboard__chart-tooltip');
		await expect(ticks.last()).toHaveText(maximum);
		await anonymous.click();
		await expect(anonymous).toHaveAttribute('aria-pressed', 'false');
		await expect(
			page.locator('.identity-dashboard__connection-legend').getByRole('button', { pressed: false }),
		).toHaveCount(1);
		await expect(card.locator('.recharts-area')).toHaveCount(1);
		await expect(ticks.last()).toHaveText('10');
		await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
		await expect(anonymous).toHaveAttribute('aria-pressed', 'false');
		await page.clock.runFor(2000);
		await card.locator('.recharts-wrapper').scrollIntoViewIfNeeded();
		const point = await card.locator('.recharts-area-dot').last().boundingBox();
		const firstTick = await card
			.locator('.recharts-xAxis .recharts-cartesian-axis-tick-value')
			.first()
			.boundingBox();
		await page.mouse.move(firstTick!.x + firstTick!.width / 2, point!.y + point!.height / 2 + 10);
		await expect(tooltip).toHaveCount(0);
		await page.mouse.move(point!.x + point!.width / 2, point!.y + point!.height / 2 + 1);
		await expect(tooltip.locator('dt')).toHaveText(['Recognized']);
		await expect(tooltip.locator('dd')).toHaveText(['10 (100.00%)']);
		await recognized.press('Enter');
		await expect(card.locator('.recharts-area')).toHaveCount(0);
		await expect(ticks).toHaveText(['0', '1']);
		await expect(tooltip).toHaveCount(0);
		await anonymous.press('Space');
		await expect(ticks.last()).toHaveText(anonymousMaximum);
		await recognized.click();
		await expect(card.getByRole('button', { pressed: true })).toHaveCount(2);
		await expect(ticks.last()).toHaveText(maximum);
		await expect(
			page.locator('.identity-dashboard__connections-chart').getByRole('button', { pressed: true }),
		).toHaveCount(2);
		await expect(
			page.locator('.identity-dashboard__composition-legend').getByRole('button', { pressed: true }),
		).toHaveCount(6);
		if (title === 'Identities over time (daily)') {
			await anonymous.click();
			await page.locator('sl-select[aria-label="Connection"]').click();
			await page.getByRole('option', { name: 'Removed connections' }).click();
			await expect(ticks.last()).toHaveText('8');
			await expect(anonymous).toHaveAttribute('aria-pressed', 'false');
		}
	});
}

for (const [title, scenario, hiddenType] of [
	['Identities over time (daily)', 'singleDayZero', 'Anonymous'],
	['Profiles over time (daily)', 'resolutionSingleDayZero', 'Recognized'],
] as const) {
	test(`keeps the scale and zero point in ${title} when the visible series contains only zeros`, async ({ page }) => {
		await page.evaluate((scenario) => {
			window.identityOverviewScenario = scenario;
		}, scenario);
		await page.addScriptTag({ content: componentScript });
		const card = page.locator('.identity-dashboard__chart-card').filter({ hasText: title });
		await card.getByRole('button', { name: hiddenType, exact: true }).click();
		await expect(card.locator('.recharts-yAxis .recharts-cartesian-axis-tick-value')).toHaveText(['0', '1']);
		await expect(card.locator('.recharts-area')).toHaveCount(1);
		await expect(card.locator('.recharts-area-dot')).toHaveCount(1);
		await expect(card.locator('.recharts-area-dot')).toBeVisible();
	});
}

test('filters composition buckets while keeping colors, percentages and the shown total consistent', async ({
	page,
}) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'legendFilters';
	});
	await page.addScriptTag({ content: componentScript });
	const card = page.locator('.identity-dashboard__composition-card');
	const legend = card.locator('.identity-dashboard__composition-legend');
	const buttons = legend.getByRole('button');
	const percentages = legend.locator('strong');
	const center = card.locator('.identity-dashboard__donut-center');
	const sectors = card.locator('.recharts-pie-sector path');
	await expect(center).toHaveText('100Profiles');
	await expect(percentages).toHaveText(['80.0%', '15.0%', '5.0%', '0.0%', '0.0%', '0.0%']);
	await expect(sectors).toHaveCount(3);
	const colors = await sectors.evaluateAll((paths) => paths.map((path) => path.getAttribute('fill')));
	await buttons.nth(0).click();
	await expect(center).toHaveText('20Profiles shown');
	await expect(percentages).toHaveText(['—', '75.0%', '25.0%', '0.0%', '0.0%', '0.0%']);
	await expect(sectors).toHaveCount(2);
	await expect(sectors.nth(0)).toHaveAttribute('fill', colors[1]!);
	await expect(sectors.nth(1)).toHaveAttribute('fill', colors[2]!);
	const ring = await card.locator('.identity-dashboard__donut-ring').boundingBox();
	await page.mouse.move(ring!.x + ring!.width / 2 + 42, ring!.y + ring!.height / 2 - 42);
	await expect(card.locator('.identity-dashboard__chart-tooltip strong')).toHaveText('2 identities');
	await expect(card.locator('.identity-dashboard__chart-tooltip dd')).toHaveText('15');
	await buttons.nth(1).press('Enter');
	await expect(center).toHaveText('5Profiles shown');
	await expect(percentages).toHaveText(['—', '—', '100.0%', '0.0%', '0.0%', '0.0%']);
	await buttons.nth(2).press('Space');
	await expect(center).toHaveText('0Profiles shown');
	await expect(card.locator('.identity-dashboard__composition-ring-placeholder')).toBeVisible();
	await expect(card.locator('.identity-dashboard__chart-tooltip')).toHaveCount(0);
	for (const button of (await buttons.all()).slice(3)) await button.click();
	await expect(buttons).toHaveCount(6);
	await expect(legend.getByRole('button', { pressed: true })).toHaveCount(0);
	await expect(percentages).toHaveText(['—', '—', '—', '—', '—', '—']);
	for (const button of await buttons.all()) await button.click();
	await expect(center).toHaveText('100Profiles');
	await expect(percentages).toHaveText(['80.0%', '15.0%', '5.0%', '0.0%', '0.0%', '0.0%']);
	await expect(sectors).toHaveCount(3);
	expect(await sectors.evaluateAll((paths) => paths.map((path) => path.getAttribute('fill')))).toEqual(colors);
	await expect(
		page.locator('.identity-dashboard__connection-legend').getByRole('button', { pressed: true }),
	).toHaveCount(6);
});

for (const [hiddenType, revision] of [
	['Recognized', 1],
	['Anonymous', 2],
] as const) {
	test(`keeps the connection scale and zero labels when hiding ${hiddenType} leaves only zeros`, async ({ page }) => {
		await page.evaluate((revision) => {
			window.identityOverviewScenario = 'connectionTypes';
			window.identityOverviewRevision = revision;
		}, revision);
		await page.addScriptTag({ content: componentScript });
		const chart = page.locator('.identity-dashboard__connections-chart');
		const toggle = chart.getByRole('button', { name: hiddenType, exact: true });
		const totals = chart.locator('.recharts-label-list text');
		const ticks = chart.locator('.recharts-xAxis .recharts-cartesian-axis-tick-value');
		await expect(totals).toHaveText(revision === 1 ? ['20', '10'] : ['990', '0']);
		await toggle.click();
		await expect(toggle).toHaveAttribute('aria-pressed', 'false');
		await expect(chart.getByRole('button', { pressed: true })).toBeEnabled();
		await expect(ticks).toHaveText(['0', '1']);
		await expect(totals).toHaveText(['0', '0']);
		await expect(chart.locator('.identity-dashboard__connection-axis-share')).toHaveText(['—', '—']);
		await toggle.click();
		await expect(toggle).toHaveAttribute('aria-pressed', 'true');
		await expect(totals).toHaveText(revision === 1 ? ['20', '10'] : ['990', '0']);
		await toggle.click();
		await expect(ticks).toHaveText(['0', '1']);
		await expect(totals).toHaveText(['0', '0']);
	});
}

test('hides the since-resolution comparison until identities have been observed after resolution', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'resolution';
	});
	await page.addScriptTag({ content: componentScript });
	const card = page.locator('.identity-dashboard__kpi').filter({ hasText: 'Since last resolution' });
	await expect(card.locator('.identity-dashboard__kpi-value')).toHaveText('—');
	await expect(card.locator('.identity-dashboard__sparkline')).toHaveCount(0);
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

for (const scenario of ['resolutionAtIdentity', 'resolutionSingleDayZero'] as const) {
	test(`shows effectiveness points without hover with only one observation (${scenario})`, async ({ page }) => {
		await page.evaluate((scenario) => {
			window.identityOverviewScenario = scenario;
		}, scenario);
		await page.addScriptTag({ content: componentScript });
		const hasZeroLinkRate = scenario === 'resolutionSingleDayZero';
		for (const [name, value] of [
			['Identities per profile', hasZeroLinkRate ? '1' : '2'],
			['Identity link rate', hasZeroLinkRate ? '0.0%' : '100.0%'],
		]) {
			await expect(
				page
					.locator('.identity-dashboard__kpi')
					.filter({ hasText: name })
					.locator('.identity-dashboard__kpi-value'),
			).toHaveText(value);
		}
		await expect(page.locator('sl-button').filter({ hasText: 'Last 30 days' })).toHaveAttribute(
			'variant',
			'primary',
		);
		const charts = page.locator('.identity-dashboard__effectiveness-chart');
		await expect(charts).toHaveCount(2);
		await page.mouse.move(0, 0);
		await page.clock.runFor(2000);
		for (const chart of await charts.all()) {
			const dots = chart.locator('.recharts-line-dot');
			await expect(dots).toHaveCount(1);
			await expect(dots).toBeVisible();
			await expect(dots).toHaveAttribute('r', '3');
			await expect(chart.locator('.recharts-active-dot')).toHaveCount(0);
			await expect(dots).toHaveAttribute(
				'fill',
				await chart.locator('.recharts-line-curve').getAttribute('stroke'),
			);
			await expect(dots).toHaveCSS('opacity', '1');
			expect(
				await chart.locator('.recharts-line-curve').evaluate((path: SVGPathElement) => path.getTotalLength()),
			).toBe(0);
		}
	});
}

for (const [ratioCount, linkRateCount] of [
	[1, 2],
	[2, 1],
	[2, 2],
	[1, 0],
	[0, 1],
	[0, 0],
]) {
	test(`handles effectiveness series independently (${ratioCount} ratio, ${linkRateCount} link rate values)`, async ({
		page,
	}) => {
		await page.evaluate(
			({ ratioCount, linkRateCount }) => {
				window.identityOverviewEffectivenessData = Array.from({ length: 30 }, (_, index) => ({
					day: new Date(Date.UTC(2026, 6, 12 + index)).toISOString().slice(0, 10),
					identitiesPerProfile: index >= 30 - ratioCount ? 1 : null,
					linkedIdentitiesRatePercent: index >= 30 - linkRateCount ? 0 : null,
				}));
			},
			{ ratioCount, linkRateCount },
		);
		await page.addScriptTag({ content: componentScript });
		await expect(page.getByText('Resolution effectiveness over time (daily)', { exact: true })).toBeVisible();
		await page.mouse.move(0, 0);
		await page.clock.runFor(2000);
		if (ratioCount === 0 && linkRateCount === 0) {
			await expect(page.getByText('No identity resolution data in this period', { exact: true })).toBeVisible();
			await expect(page.locator('.identity-dashboard__effectiveness-chart')).toHaveCount(0);
			return;
		}
		for (const [index, count] of [ratioCount, linkRateCount].entries()) {
			const metric = page.locator('.identity-dashboard__effectiveness-metric').nth(index);
			const dots = metric.locator('.recharts-line-dot');
			await expect(dots).toHaveCount(count === 1 ? 1 : 0);
			if (count === 0) {
				await expect(metric.getByText('No data available', { exact: true })).toBeVisible();
				await expect(metric.locator('.recharts-surface')).toHaveCount(0);
			} else if (count === 1) {
				await expect(dots).toBeVisible();
				await expect(dots).toHaveAttribute('r', '3');
				await expect(dots).toHaveAttribute(
					'fill',
					await metric.locator('.recharts-line-curve').getAttribute('stroke'),
				);
				await expect(dots).toHaveCSS('opacity', '1');
			} else {
				expect(
					await metric
						.locator('.recharts-line-curve')
						.evaluate((path: SVGPathElement) => path.getTotalLength()),
				).toBeGreaterThan(0);
			}
			await expect(metric.locator('.recharts-active-dot')).toHaveCount(0);
		}
	});
}

test('loads removed connection history when the current state is zero', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'removed';
	});
	await page.addScriptTag({ content: componentScript });
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
		await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
		await expectUpdatedDashboard(page);
	});
}

for (const shouldFail of [false, true]) {
	test(`ignores superseded range responses (${shouldFail ? 'failure' : 'success'})`, async ({ page }) => {
		await page.evaluate(() => {
			window.identityOverviewScenario = 'atomic';
		});
		await page.addScriptTag({ content: componentScript });
		await expect(page.locator('.identity-dashboard__history-status')).toHaveText('Successful');
		const previous = await dashboardSnapshot(page);
		await page.evaluate((shouldFail) => {
			window.identityOverviewHold = '/metrics/identity-resolution/dates/';
			window.identityOverviewFailure = shouldFail ? window.identityOverviewHold : undefined;
		}, shouldFail);
		await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
		await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
		await expect(page.locator('.identity-dashboard__card[aria-busy="true"] h3')).toHaveText([
			'Resolution effectiveness over time (daily)',
			'Profiles over time (daily)',
			'Identities over time (daily)',
		]);
		expect(await dashboardSnapshot(page)).toEqual(previous);
		await page.evaluate(() => {
			window.identityOverviewHold = undefined;
			window.identityOverviewFailure = undefined;
		});
		await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
		await expect.poll(() => dashboardSnapshot(page)).not.toEqual(previous);
		await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
		const current = await dashboardSnapshot(page);
		await page.evaluate(() => window.identityOverviewPending[0]());
		expect(await dashboardSnapshot(page)).toEqual(current);
		await expect(page.getByRole('alert')).toHaveCount(0);
		await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
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

test('keeps cached snapshots and represented ranges after a failed range change', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.identity-dashboard__history-status')).toHaveText('Successful');
	const previous = await dashboardSnapshot(page);
	await page.evaluate(() => {
		window.identityOverviewRevision = 1;
		window.identityOverviewFailure = '/metrics/identity-resolution/dates/';
	});
	await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await page.evaluate(() => {
		window.identityOverviewRevision = 0;
		window.identityOverviewFailure = undefined;
	});
	await page.getByRole('button', { name: 'Last 30 days', exact: true }).click();
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await expect(page.getByRole('alert')).toHaveCount(0);
});

test('accepts empty resolution data and a zero identity snapshot', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'singleDayZero';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(
		page
			.locator('.identity-dashboard__kpi')
			.filter({ hasText: 'Total identities' })
			.locator('.identity-dashboard__kpi-value'),
	).toHaveText('0');
	await expect(page.getByText('No identity resolution runs yet', { exact: true })).toBeVisible();
	await expect(page.getByRole('alert')).toHaveCount(0);
});

test('keeps the requested connection selected and allows leaving it after a failure', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.identity-dashboard__history-status')).toHaveText('Successful');
	const previous = await dashboardSnapshot(page);
	await page.evaluate(() => {
		window.identityOverviewRequests = [];
		window.identityOverviewFailure = '?connection=deleted';
		window.identityOverviewHold = '?connection=deleted';
	});
	const select = page.locator('sl-select[aria-label="Connection"]');
	await select.click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect.poll(() => page.evaluate(() => window.identityOverviewPending.length)).toBe(1);
	await expect(select).toHaveJSProperty('value', 'deleted');
	await expect(page.locator('.identity-dashboard__card[aria-busy="true"] h3')).toHaveText([
		'Identities over time (daily)',
	]);
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await page.evaluate(() => window.identityOverviewPending[0]());
	await expect(page.getByText('Dashboard data could not be loaded', { exact: true })).toBeVisible();
	await expect(select).toHaveJSProperty('value', 'deleted');
	await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await select.click();
	await page.getByRole('option', { name: 'All connections', exact: true }).click();
	await expect(select).toHaveJSProperty('value', '');
	await expect(page.getByRole('alert')).toHaveCount(0);
	expect(await dashboardSnapshot(page)).toEqual(previous);
	await page.evaluate(() => {
		window.identityOverviewHold = undefined;
	});
	await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
	await expect.poll(() => dashboardSnapshot(page)).not.toEqual(previous);
	await expect(select).toHaveJSProperty('value', '');
	await expect(page.getByRole('alert')).toHaveCount(0);
	const requests = await page.evaluate(() =>
		window.identityOverviewRequests.filter((url) => url.startsWith('https://example.test/')),
	);
	expect(requests).toHaveLength(3);
	expect(requests.filter((url) => url.includes('?connection=deleted'))).toHaveLength(1);
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

test('clears the previous workspace and ignores its unfinished load after a workspace change', async ({ page }) => {
	await page.evaluate(() => {
		window.identityOverviewScenario = 'atomic';
	});
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.identity-dashboard__history-status')).toHaveText('Successful');
	const select = page.locator('sl-select[aria-label="Connection"]');
	await select.click();
	await page.getByRole('option', { name: 'Removed connections' }).click();
	await expect(select).toHaveJSProperty('value', 'deleted');
	await page.evaluate(() => {
		window.identityOverviewHold = '/metrics/identity-resolution/dates/';
	});
	await page.getByRole('button', { name: 'Last 90 days', exact: true }).click();
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
	await page.getByRole('button', { name: 'Last 7 days', exact: true }).click();
	await expectUpdatedDashboard(page);
});
