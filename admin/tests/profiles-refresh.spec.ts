// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';
import { ResponseProfile } from '../src/lib/api/types/responses';
import { ObjectType } from '../src/lib/api/types/types';

const schema: ObjectType = {
	kind: 'object',
	properties: [
		{
			name: 'email',
			type: { kind: 'string' },
			displayName: 'Email',
			description: '',
			prefilled: '',
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
		},
		{
			name: 'name',
			type: { kind: 'string' },
			displayName: 'Name',
			description: '',
			prefilled: '',
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
		},
	],
};
const profiles: ResponseProfile[] = Array.from({ length: 120 }, (_, index) => ({
	kpid: `profile-${index + 1}`,
	updatedAt: '2026-11-01T00:00:00Z',
	attributes: { email: `profile-${index + 1}@example.test`, name: `Person ${index + 1}` },
}));

let componentScript: string;
let componentStyles: string;

test.beforeAll(async () => {
	const result = await build({
		entryPoints: [resolve(__dirname, 'profiles-refresh.fixture.tsx')],
		bundle: true,
		write: false,
		platform: 'browser',
		format: 'iife',
		outdir: '/tmp/profiles-refresh-fixture',
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	componentScript = result.outputFiles.find((file) => file.path.endsWith('.js'))!.text;
	componentStyles = result.outputFiles.find((file) => file.path.endsWith('.css'))!.text;
});

test.beforeEach(async ({ page }) => {
	await page.route('**/*', (route) => route.abort());
	await page.route('https://example.test/', (route) =>
		route.fulfill({ contentType: 'text/html', body: '<div id="root"></div>' }),
	);
	await page.route('**/v1/profiles/schema', (route) =>
		route.fulfill({
			json: {
				schema,
				propertyRoles: { firstName: '', lastName: '', email: '', country: '', photo: '' },
				primarySources: {},
			},
		}),
	);
	await page.route('**/v1/identity-resolution/latest', (route) =>
		route.fulfill({ json: { startTime: null, endTime: null } }),
	);
	await page.route('**/v1/profiles/*/attributes*', (route) =>
		route.fulfill({ json: { attributes: profiles[50].attributes } }),
	);
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
});

for (const failure of [
	{ status: 503, code: 'ServiceUnavailable', message: 'Data warehouse unavailable' },
	{ status: 422, code: 'SchemaNotAligned', message: 'Profile schema has changed' },
]) {
	test(`Preserve the profile view after Refresh fails with ${failure.code} and allow a retry`, async ({ page }) => {
		let refreshReads = 0;
		let markRefreshStarted = () => {};
		let releaseRefresh = () => {};
		const refreshStarted = new Promise<void>((resolve) => {
			markRefreshStarted = resolve;
		});
		const refreshCanFinish = new Promise<void>((resolve) => {
			releaseRefresh = resolve;
		});

		await page.route('**/v1/profiles?*', async (route) => {
			const url = new URL(route.request().url());
			const first = Number(url.searchParams.get('first'));
			const limit = Number(url.searchParams.get('limit'));
			if (first === 0) {
				refreshReads++;
				if (refreshReads === 2) {
					markRefreshStarted();
					await refreshCanFinish;
					await route.fulfill({ status: failure.status, json: { error: failure } });
					return;
				}
			}
			const result = profiles.slice(first, first + limit);
			await route.fulfill({
				json: {
					profiles: result,
					total: profiles.length,
					hasNext: first + result.length < profiles.length,
				},
			});
		});
		await page.addScriptTag({ content: componentScript });

		const rows = page.locator('.profiles-list .grid__row--clickable');
		const pagination = page.getByRole('navigation', { name: 'Profiles pagination' });
		await expect(rows).toHaveCount(50);
		await pagination.getByRole('button', { name: 'Next page' }).click();
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
		await rows.first().click();
		await expect(page.locator('.profile-drawer')).toContainText('profile-51');
		const displayedProfileIDs = await rows.evaluateAll((rows) => rows.map((row) => row.getAttribute('data-id')));

		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await refreshStarted;
		await expect(rows).toHaveCount(50);
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
		await expect(rows.first()).toHaveClass(/grid__row--active/);
		await expect(page.locator('.profile-drawer')).toContainText('profile-51');
		releaseRefresh();

		if (failure.code === 'SchemaNotAligned') {
			await expect(page.locator('.profiles-list__stale-notice')).toBeVisible();
			await expect(page.getByRole('alert')).toBeEmpty();
		} else {
			await expect(page.getByRole('alert')).toContainText(failure.message);
		}
		await expect(rows).toHaveCount(50);
		expect(await rows.evaluateAll((rows) => rows.map((row) => row.getAttribute('data-id')))).toEqual(
			displayedProfileIDs,
		);
		await expect(page.locator('.profiles-list__grid-summary')).toContainText('120 profiles');
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
		await expect(rows.first()).toHaveClass(/grid__row--active/);
		await expect(page.locator('.profile-drawer')).toContainText('profile-51');

		await pagination.getByRole('button', { name: 'Next page' }).click();
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText('101–120 of 120');
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 120');
		await expect(rows).toHaveCount(50);
		await expect(page.locator('.profile-drawer')).toHaveCount(0);
		await expect(page.locator('.profiles-list__stale-notice')).toHaveCount(0);
		expect(refreshReads).toBe(3);
	});
}

for (const shouldShowColumn of [false, true]) {
	test(`Complete a pending Refresh when ${shouldShowColumn ? 'showing' : 'hiding'} a loaded column`, async ({
		page,
	}) => {
		let refreshReads = 0;
		let markRefreshStarted = () => {};
		let releaseRefresh = () => {};
		const refreshStarted = new Promise<void>((resolve) => {
			markRefreshStarted = resolve;
		});
		const refreshCanFinish = new Promise<void>((resolve) => {
			releaseRefresh = resolve;
		});
		await page.route('**/v1/profiles?*', async (route) => {
			const url = new URL(route.request().url());
			const first = Number(url.searchParams.get('first'));
			const limit = Number(url.searchParams.get('limit'));
			if (first === 0) {
				refreshReads++;
				if (refreshReads === 2) {
					markRefreshStarted();
					await refreshCanFinish;
				}
			}
			const projection = url.searchParams.get('properties')?.split(',') ?? [];
			const result = profiles.slice(first, first + limit).map((profile) => ({
				...profile,
				attributes: Object.fromEntries(
					Object.entries(profile.attributes)
						.filter(([name]) => projection.includes(name))
						.map(([name, value]) => [name, refreshReads === 2 ? `Updated ${value}` : value]),
				),
			}));
			await route.fulfill({ json: { profiles: result, total: profiles.length, hasNext: true } });
		});
		await page.addScriptTag({ content: componentScript });
		const rows = page.locator('.profiles-list .grid__row--clickable');
		await expect(rows.first()).toContainText('profile-1@example.test');
		const columns = page.locator('.profiles-list__toggle-columns');
		await columns.getByRole('button', { name: 'Columns', exact: true }).click();
		const nameColumn = columns.locator('sl-checkbox', { hasText: 'Name' });
		const nameHeader = page.locator('.profiles-list .grid__header-cell', { hasText: 'Name' });
		if (shouldShowColumn) {
			await nameColumn.click();
			await expect(nameHeader).toHaveCount(0);
		}
		await columns.evaluate((dropdown: any) => dropdown.hide());
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await refreshStarted;
		await columns.evaluate((dropdown: any) => dropdown.show());
		await nameColumn.click();
		await expect(nameColumn).toHaveJSProperty('checked', shouldShowColumn);
		await expect(nameHeader).toHaveCount(shouldShowColumn ? 1 : 0);
		releaseRefresh();
		await expect(rows.first()).toContainText('Updated profile-1@example.test');
		if (shouldShowColumn) {
			await expect(rows.first()).toContainText('Updated Person 1');
		}
		await expect(nameHeader).toHaveCount(shouldShowColumn ? 1 : 0);
		await expect(page.getByRole('alert')).toBeEmpty();
		expect(refreshReads).toBe(2);
	});
}
