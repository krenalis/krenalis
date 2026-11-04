// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';
import { Filter, FilterCondition } from '../src/lib/api/types/pipeline';
import { ObjectType } from '../src/lib/api/types/types';
import { PROFILES_SHOW_MATCHES_AUTOMATICALLY_KEY } from '../src/constants/storage';

const schema: ObjectType = {
	kind: 'object',
	properties: [
		{
			name: 'name',
			type: { kind: 'string' },
			displayName: 'Name',
			description: 'x'.repeat(32 * 1024),
			prefilled: '',
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
		},
	],
};
const values = ['  Ada  ', '"Ada"', "'Ada'"];
const profiles = values.map((name, index) => ({
	kpid: `profile-${index + 1}`,
	updatedAt: '2026-11-01T00:00:00Z',
	attributes: { name },
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
		outdir: '/tmp/profiles-filters-fixture',
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
				propertyRoles: { firstName: 'name', lastName: '', email: '', country: '', photo: '' },
				primarySources: {},
			},
		}),
	);
	await page.route('**/v1/identity-resolution/latest', (route) =>
		route.fulfill({ json: { startTime: null, endTime: null } }),
	);
});

for (const [index, value] of values.entries()) {
	test(`Preserve ${JSON.stringify(value)} through Undo, Redo and closing the filter editor`, async ({ page }) => {
		const countFilters: (Filter | null)[] = [];
		const gridFilters: (Filter | null)[] = [];
		const filter: Filter = {
			operator: 'and',
			rules: [{ property: 'name', operator: 'is', values: [value] }],
		};
		const draftValue = JSON.stringify(value);
		await page.emulateMedia({ reducedMotion: 'reduce' });
		await page.route(/\/v1\/profiles\/(?:query|count)$/, async (route) => {
			const url = new URL(route.request().url());
			expect(route.request().method()).toBe('POST');
			expect(url.search).toBe('');
			expect(route.request().postDataJSON().schema).toEqual(schema);
			const requestedFilter: Filter | null = route.request().postDataJSON().filter ?? null;
			const requestedValue = (requestedFilter?.rules[0] as FilterCondition | undefined)?.values?.[0];
			const result = profiles.filter(
				(profile) => requestedFilter == null || profile.attributes.name === requestedValue,
			);
			if (url.pathname.endsWith('/count')) {
				countFilters.push(requestedFilter);
				await route.fulfill({ json: { total: result.length } });
			} else {
				gridFilters.push(requestedFilter);
				await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
			}
		});
		await page.goto('https://example.test/');
		await page.addStyleTag({ content: componentStyles });
		await page.addScriptTag({ content: componentScript });
		const rows = page.locator('.profiles-list .grid__row--clickable');
		await expect(rows).toHaveCount(3);

		const filters = page.locator('.profiles-list__filters');
		await filters.getByRole('button', { name: 'Change', exact: true }).click();
		const editor = filters.getByRole('region', { name: 'Profile filters' });
		const property = editor.locator('.filter-editor__property');
		const propertyInput = property.locator('input');
		await propertyInput.pressSequentially('name');
		await expect(propertyInput).toHaveValue('name');
		await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
		const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
		await operator.press('Home');
		await expect(propertyInput).toHaveValue('Name');
		await propertyInput.focus();
		await expect(propertyInput).toHaveValue('name');
		await propertyInput.press('Backspace');
		await expect(propertyInput).toHaveValue('nam');
		await propertyInput.pressSequentially('e');
		await operator.press('Home');
		await expect(propertyInput).toHaveValue('Name');
		await operator.press('Enter');
		const valueInput = editor.locator('.filter-editor__value-input input');
		await valueInput.fill(draftValue);
		await valueInput.press('Enter');
		const showMatches = page.getByRole('button', { name: 'Show 1 match', exact: true });
		await expect(showMatches).toBeVisible();
		expect(countFilters).toEqual([filter]);
		await expect(rows).toHaveCount(3);

		await valueInput.fill('');
		const undo = filters.getByRole('button', { name: 'Undo', exact: true });
		await undo.click();
		await expect(valueInput).toHaveValue(draftValue);
		await expect(showMatches).toBeVisible();
		expect(countFilters).toEqual([filter]);

		await undo.click();
		await expect(property.locator('input')).toHaveValue('');
		const redo = filters.getByRole('button', { name: 'Redo', exact: true });
		await expect(redo).toBeEnabled();
		await redo.click();
		await expect(valueInput).toHaveValue(draftValue);
		await expect(showMatches).toBeVisible();
		expect(countFilters).toEqual([filter, null, filter]);
		await expect(rows).toHaveCount(3);

		await filters.getByRole('button', { name: 'Reduce', exact: true }).click();
		await expect(filters.getByRole('button', { name: 'Change', exact: true })).toBeVisible();
		await expect(rows).toHaveCount(1);
		await expect(rows).toHaveAttribute('data-id', `profile-${index + 1}`);
		expect(gridFilters).toEqual([null, filter]);

		await filters.getByRole('button', { name: 'Change', exact: true }).click();
		await expect(valueInput).toHaveValue(draftValue);
		await valueInput.fill('"');
		await filters.getByRole('button', { name: 'Reduce', exact: true }).click();
		await filters.getByRole('button', { name: 'Change', exact: true }).click();
		await expect(valueInput).toHaveValue(draftValue);
		expect(countFilters).toEqual([filter, null, filter]);
		expect(gridFilters).toEqual([null, filter]);
		await expect(page.getByRole('alert')).toBeEmpty();
	});
}

test(`Number removable groups across nested and sibling groups`, async ({ page }) => {
	await page.route(/\/v1\/profiles\/query$/, (route) =>
		route.fulfill({ json: { profiles, total: profiles.length, hasNext: false } }),
	);
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	const rootActions = filters.locator('.filter-editor__group--root > .filter-editor__group-actions');
	await rootActions.getByRole('button', { name: 'Add a group', exact: true }).click();
	const groups = filters.locator('.filter-editor__group:not(.filter-editor__group--root)');
	await groups.first().getByRole('button', { name: 'Add a group', exact: true }).click();
	await rootActions.getByRole('button', { name: 'Add a group', exact: true }).click();
	const removeGroups = filters.locator('.filter-editor__remove-group');
	await expect(removeGroups).toHaveCount(3);
	for (let index = 0; index < 3; index++) {
		await expect(removeGroups.nth(index).getByRole('button')).toHaveAccessibleName(`Remove group ${index + 1}`);
	}
	await expect(filters.locator('.filter-editor__remove-group-wrapper wa-tooltip')).toHaveText([
		'Remove groups',
		'Remove group',
		'Remove group',
	]);
	await expect(filters.locator('.filter-editor__remove-condition-wrapper wa-tooltip')).toHaveText([
		'Remove condition',
		'Remove condition',
	]);
	await filters.getByRole('button', { name: 'Remove group 1', exact: true }).click();
	await expect(removeGroups).toHaveCount(1);
	await expect(removeGroups.getByRole('button')).toHaveAccessibleName('Remove group 1');
	await expect(groups.getByRole('button', { name: 'Remove condition 2', exact: true })).toBeDisabled();
});

test(`Automatically show completed previews, preserve focus and remember the preference`, async ({ page }) => {
	const countFilters: (Filter | null)[] = [];
	const gridFilters: (Filter | null)[] = [];
	let releaseCount = () => {};
	let releaseGrid = () => {};
	const countCanFinish = new Promise<void>((resolve) => {
		releaseCount = resolve;
	});
	const gridCanFinish = new Promise<void>((resolve) => {
		releaseGrid = resolve;
	});
	await page.route(/\/v1\/profiles\/(?:query|count)$/, async (route) => {
		const url = new URL(route.request().url());
		const filter: Filter | null = route.request().postDataJSON().filter ?? null;
		const value = (filter?.rules[0] as FilterCondition | undefined)?.values?.[0];
		const result = profiles.filter((profile) => filter == null || profile.attributes.name === value);
		if (url.pathname.endsWith('/count')) {
			countFilters.push(filter);
			await countCanFinish;
			await route.fulfill({ json: { total: result.length } });
		} else {
			gridFilters.push(filter);
			if (filter != null) {
				await gridCanFinish;
			}
			await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
		}
	});
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	const filters = page.locator('.profiles-list__filters');
	const automaticUpdates = filters.getByRole('switch', { name: 'Show matches automatically' });
	await expect(rows).toHaveCount(3);
	await expect(automaticUpdates).not.toBeVisible();
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	await expect(automaticUpdates).not.toBeChecked();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const property = editor.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('input').fill('Name');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
	const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = editor.locator('.filter-editor__value-input input');
	await valueInput.fill(JSON.stringify(values[0]));
	await valueInput.press('Enter');
	await expect(page.locator('.profiles-list__filter-preview-count')).toBeVisible();
	await filters.getByText('Show matches automatically', { exact: true }).click();
	await valueInput.focus();
	expect(gridFilters).toEqual([null]);
	releaseCount();
	await expect(filters).toHaveClass(/profiles-list__filters--updating-results/);
	await expect(page.locator('.profiles-list__grid-summary-sweep')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-sweep',
	);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(rows).toHaveCount(3);
	releaseGrid();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-1');
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(valueInput).toBeFocused();
	expect(gridFilters).toEqual([null, countFilters[0]]);

	await valueInput.fill('Missing');
	await valueInput.press('Enter');
	await expect(rows).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('0 profiles');
	await expect(valueInput).toBeFocused();
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-1');
	await filters.getByRole('button', { name: 'Redo', exact: true }).click();
	await expect(rows).toHaveCount(0);
	expect(gridFilters.slice(1)).toEqual(countFilters);

	await filters.getByText('Show matches automatically', { exact: true }).click();
	await valueInput.fill(JSON.stringify(values[1]));
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(rows).toHaveCount(0);
	expect(gridFilters).toHaveLength(5);
	await filters.getByText('Show matches automatically', { exact: true }).click();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-2');
	expect(gridFilters.slice(1)).toEqual(countFilters);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await filters.getByRole('button', { name: 'Reduce', exact: true }).click();
	await expect(automaticUpdates).not.toBeVisible();
	await expect(page.getByRole('alert')).toBeEmpty();

	await page.reload();
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	await expect(automaticUpdates).toBeChecked();
	expect(await page.evaluate((key) => localStorage.getItem(key), PROFILES_SHOW_MATCHES_AUTOMATICALLY_KEY)).toBe(
		'true',
	);
	await filters.getByText('Show matches automatically', { exact: true }).click();
	await expect(automaticUpdates).not.toBeChecked();
	expect(await page.evaluate((key) => localStorage.getItem(key), PROFILES_SHOW_MATCHES_AUTOMATICALLY_KEY)).toBe(
		'false',
	);

	await page.reload();
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	await expect(automaticUpdates).not.toBeChecked();
});

test(`Keep results after failed automatic updates and recover with Retry and Undo`, async ({ page }) => {
	let gridRequests = 0;
	await page.route(/\/v1\/profiles\/(?:query|count)$/, async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname.endsWith('/count')) {
			await route.fulfill({ json: { total: 1 } });
			return;
		}
		if (route.request().postDataJSON().filter == null) {
			await route.fulfill({ json: { profiles, total: profiles.length, hasNext: false } });
			return;
		}
		gridRequests++;
		if (gridRequests === 1 || gridRequests === 3) {
			await route.fulfill({
				status: 503,
				json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
			});
			return;
		}
		await route.fulfill({ json: { profiles: [profiles[0]], total: 1, hasNext: false } });
	});
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	const filters = page.locator('.profiles-list__filters');
	await expect(rows).toHaveCount(3);
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	await filters.getByText('Show matches automatically', { exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const property = editor.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('input').fill('Name');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
	const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = editor.locator('.filter-editor__value-input input');
	await valueInput.fill(JSON.stringify(values[0]));
	await valueInput.press('Enter');
	const failure = editor.locator('.profiles-list__filter-grid-error');
	await expect(failure).toContainText('Unable to show profiles.');
	await expect(rows).toHaveCount(3);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('3 profiles');
	await expect(filters).not.toHaveClass(/profiles-list__filters--updating-results/);
	await expect(filters).not.toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(valueInput).toBeFocused();
	expect(gridRequests).toBe(1);

	await failure.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-1');
	await expect(failure).toHaveCount(0);
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	expect(gridRequests).toBe(2);

	await valueInput.fill(JSON.stringify(values[1]));
	await valueInput.press('Enter');
	await expect(failure).toContainText('Unable to show profiles.');
	expect(gridRequests).toBe(3);
	await valueInput.fill('"');
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect.poll(() => gridRequests).toBe(4);
	await expect(failure).toHaveCount(0);
	await expect(rows).toHaveCount(1);
});

test(`Resume automatic matches after Refresh cancels their loading`, async ({ page }) => {
	let filteredReads = 0;
	let releaseMatches = () => {};
	const matchesCanFinish = new Promise<void>((resolve) => {
		releaseMatches = resolve;
	});
	await page.route(/\/v1\/profiles\/(?:query|count)$/, async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname.endsWith('/count')) {
			await route.fulfill({ json: { total: 1 } });
			return;
		}
		const hasFilter = route.request().postDataJSON().filter != null;
		if (hasFilter) {
			filteredReads++;
			await matchesCanFinish;
		}
		await route.fulfill({
			json: {
				profiles: hasFilter ? [profiles[0]] : profiles,
				total: hasFilter ? 1 : profiles.length,
				hasNext: false,
			},
		});
	});
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	const filters = page.locator('.profiles-list__filters');
	await expect(rows).toHaveCount(3);
	await filters.getByRole('button', { name: 'Change', exact: true }).click();
	await filters.getByText('Show matches automatically', { exact: true }).click();
	const property = filters.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('input').fill('Name');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
	const operator = filters.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = filters.locator('.filter-editor__value-input input');
	await valueInput.fill(JSON.stringify(values[0]));
	await valueInput.press('Enter');
	try {
		await expect.poll(() => filteredReads).toBe(1);
		await page.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expect.poll(() => filteredReads).toBe(2);
	} finally {
		releaseMatches();
	}
	await expect(rows).toHaveCount(1);
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(page.getByRole('alert')).toBeEmpty();
});
