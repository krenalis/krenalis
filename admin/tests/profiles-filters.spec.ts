// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';
import { Filter, FilterCondition } from '../src/lib/api/types/pipeline';
import { ObjectType } from '../src/lib/api/types/types';

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
	await page.route('**/v1/profiles/count', (route) => {
		expect(route.request().url(), 'expected no separate count request').toBe('');
	});
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
		const gridFilters: (Filter | null)[] = [];
		const filter: Filter = {
			operator: 'and',
			rules: [{ property: 'name', operator: 'is', values: [value] }],
		};
		const draftValue = JSON.stringify(value);
		await page.emulateMedia({ reducedMotion: 'reduce' });
		await page.route(/\/v1\/profiles\/query$/, async (route) => {
			const url = new URL(route.request().url());
			expect(route.request().method()).toBe('POST');
			expect(url.search).toBe('');
			expect(route.request().postDataJSON().schema).toEqual(schema);
			const requestedFilter: Filter | null = route.request().postDataJSON().filter ?? null;
			const requestedValue = (requestedFilter?.rules[0] as FilterCondition | undefined)?.values?.[0];
			const result = profiles.filter(
				(profile) => requestedFilter == null || profile.attributes.name === requestedValue,
			);
			gridFilters.push(requestedFilter);
			await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
		});
		await page.goto('https://example.test/');
		await page.addStyleTag({ content: componentStyles });
		await page.addScriptTag({ content: componentScript });
		const rows = page.locator('.profiles-list .grid__row--clickable');
		await expect(rows).toHaveCount(3);

		const filters = page.locator('.profiles-list__filters');
		await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
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
		const applyFilters = page.getByRole('button', { name: 'Apply filters', exact: true });
		await expect(applyFilters).toBeEnabled();
		await expect(rows).toHaveCount(3);

		await valueInput.fill('');
		const undo = filters.getByRole('button', { name: 'Undo', exact: true });
		await undo.click();
		await expect(valueInput).toHaveValue(draftValue);
		await expect(applyFilters).toBeEnabled();

		await undo.click();
		await expect(property.locator('input')).toHaveValue('');
		await expect(applyFilters).toBeDisabled();
		const redo = filters.getByRole('button', { name: 'Redo', exact: true });
		await expect(redo).toBeEnabled();
		await redo.click();
		await expect(valueInput).toHaveValue(draftValue);
		await expect(applyFilters).toBeEnabled();
		await expect(rows).toHaveCount(3);

		await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		await expect(filters.getByRole('button', { name: 'Edit filters', exact: true })).toBeVisible();
		await expect(rows).toHaveCount(1);
		await expect(rows).toHaveAttribute('data-id', `profile-${index + 1}`);
		expect(gridFilters).toEqual([null, filter]);

		await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
		await expect(valueInput).toHaveValue(draftValue);
		await valueInput.fill('"');
		await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		await expect(editor).toBeVisible();
		await expect(applyFilters).toBeDisabled();
		await expect(valueInput).toHaveValue('"');
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
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
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

test(`Apply several edits with one request and keep history independent from queries`, async ({ page }) => {
	const gridFilters: (Filter | null)[] = [];
	await page.route(/\/v1\/profiles\/query$/, async (route) => {
		const filter: Filter | null = route.request().postDataJSON().filter ?? null;
		gridFilters.push(filter);
		const value = (filter?.rules[0] as FilterCondition | undefined)?.values?.[0];
		const result = profiles.filter((profile) => filter == null || profile.attributes.name === value);
		await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
	});
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.clock.install({ time: new Date('2026-11-12T12:00:00Z') });
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(rows).toHaveCount(3);
	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const applyFilters = editor.getByRole('button', { name: 'Apply filters', exact: true });
	await expect(applyFilters).toBeDisabled();
	await expect(editor.locator('.profiles-list__filter-apply')).toHaveAttribute('variant', 'neutral');
	const property = editor.locator('.filter-editor__property');
	await property.locator('input').fill('Name');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
	const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = editor.locator('.filter-editor__value-input input');
	await valueInput.fill('Ad');
	await page.clock.runFor(2000);
	await valueInput.fill(JSON.stringify(values[0]));
	await page.clock.runFor(2000);
	await expect(applyFilters).toBeEnabled();
	await expect(editor.locator('.profiles-list__filter-apply')).toHaveAttribute('variant', 'brand');
	await valueInput.press('Enter');
	await valueInput.fill(JSON.stringify(values[1]));
	await valueInput.blur();
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(valueInput).toHaveValue(JSON.stringify(values[0]));
	await filters.getByRole('button', { name: 'Redo', exact: true }).click();
	await expect(valueInput).toHaveValue(JSON.stringify(values[1]));
	await expect(rows).toHaveCount(3);
	expect(gridFilters).toEqual([null]);
	const buttonLayout = await applyFilters.boundingBox();
	await page.clock.pauseAt(new Date('2026-11-12T12:01:00Z'));
	await applyFilters.click();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-2');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	const spinner = editor.locator('.profiles-list__filter-apply wa-spinner');
	await expect(spinner).toBeVisible();
	await expect(applyFilters).toBeDisabled();
	await expect(editor.locator('.profiles-list__filter-apply')).toHaveAttribute('variant', 'brand');
	await expect(editor.locator('.profiles-list__filter-shortcut')).toHaveAttribute('aria-disabled', 'true');
	expect(await applyFilters.boundingBox()).toEqual(buttonLayout);
	await page.clock.runFor(399);
	await expect(spinner).toBeVisible();
	await page.clock.runFor(1);
	await expect(spinner).toHaveCount(0);
	await expect(applyFilters).toBeDisabled();
	expect(gridFilters).toEqual([
		null,
		{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [values[1]] }] },
	]);
});

for (const { platform, modifier, shortcut } of [
	{ platform: 'MacIntel', modifier: 'Meta', shortcut: '⌘ + Enter' },
	{ platform: 'Win32', modifier: 'Control', shortcut: 'Ctrl + Enter' },
]) {
	test(`Apply filters with ${shortcut} only inside the editor`, async ({ page }) => {
		await page.emulateMedia({ reducedMotion: 'reduce' });
		await page.clock.install({ time: new Date('2026-11-12T12:00:00Z') });
		await page.addInitScript((platform) => {
			Object.defineProperty(navigator, 'platform', { value: platform });
		}, platform);
		const gridFilters: (Filter | null)[] = [];
		let releaseGrid = () => {};
		const gridCanFinish = new Promise<void>((resolve) => {
			releaseGrid = resolve;
		});
		await page.route(/\/v1\/profiles\/query$/, async (route) => {
			const filter: Filter | null = route.request().postDataJSON().filter ?? null;
			gridFilters.push(filter);
			if (filter != null) {
				await gridCanFinish;
				await route.fulfill({ json: { profiles: [profiles[0]], total: 1, hasNext: false } });
				return;
			}
			await route.fulfill({ json: { profiles, total: profiles.length, hasNext: false } });
		});
		await page.goto('https://example.test/');
		await page.addStyleTag({ content: componentStyles });
		await page.addScriptTag({ content: componentScript });
		const rows = page.locator('.profiles-list .grid__row--clickable');
		await expect(rows).toHaveCount(3);
		const filters = page.locator('.profiles-list__filters');
		await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
		const editor = filters.getByRole('region', { name: 'Profile filters' });
		const applyFilters = editor.getByRole('button', { name: 'Apply filters', exact: true });
		const hint = editor.locator('.profiles-list__filter-shortcut');
		await expect(hint.locator('kbd')).toHaveText([modifier === 'Meta' ? '⌘' : 'Ctrl', 'Enter']);
		await expect(hint).toBeVisible();
		await expect(applyFilters).toBeDisabled();
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		const buttonLayout = await applyFilters.boundingBox();
		const hintLayout = await hint.boundingBox();
		const editorLayout = await editor.boundingBox();
		const bottomInset = editorLayout.y + editorLayout.height - buttonLayout.y - buttonLayout.height;
		const rightInset = editorLayout.x + editorLayout.width - buttonLayout.x - buttonLayout.width;
		expect(Math.abs(bottomInset - rightInset)).toBeLessThanOrEqual(1);
		expect(hintLayout.y).toBeGreaterThanOrEqual(buttonLayout.y + buttonLayout.height);
		expect(
			Math.abs(hintLayout.x + hintLayout.width / 2 - buttonLayout.x - buttonLayout.width / 2),
		).toBeLessThanOrEqual(1);
		const property = editor.locator('.filter-editor__property');
		await property.locator('input').focus();
		await page.keyboard.press(`${modifier}+Enter`);
		expect(gridFilters).toEqual([null]);
		await property.locator('input').fill('Name');
		await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
		const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
		await operator.press('Home');
		await operator.press('Enter');
		const valueInput = editor.locator('.filter-editor__value-input input');
		await valueInput.press(`${modifier}+Enter`);
		await expect(applyFilters).toBeDisabled();
		expect(gridFilters).toEqual([null]);
		await valueInput.fill(JSON.stringify(values[1]));
		await valueInput.press('Enter');
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await valueInput.blur();
		await page.keyboard.press(`${modifier}+Enter`);
		expect(gridFilters).toEqual([null]);
		await valueInput.fill(JSON.stringify(values[0]));
		await page.clock.pauseAt(new Date('2026-11-12T12:01:00Z'));
		try {
			await valueInput.press(`${modifier}+Enter`);
			await expect(applyFilters).toBeDisabled();
			await expect.poll(() => gridFilters.length).toBe(2);
			await expect(rows).toHaveCount(3);
			await page.clock.runFor(500);
			await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toBeVisible();
			await valueInput.press(`${modifier}+Enter`);
			expect(gridFilters).toEqual([
				null,
				{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [values[0]] }] },
			]);
		} finally {
			releaseGrid();
		}
		await expect(rows).toHaveCount(1);
		await expect(rows).toHaveAttribute('data-id', 'profile-1');
		await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toHaveCount(0);
		await expect(applyFilters).toBeDisabled();
		await expect(hint).toBeVisible();
		await valueInput.press(`${modifier}+Enter`);
		await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		await page.keyboard.press(`${modifier}+Enter`);
		await expect(hint).not.toBeVisible();
		expect(gridFilters).toHaveLength(2);
	});
}

test(`Retry a failed collapse without reopening the editor on a narrow viewport`, async ({ page }) => {
	let filteredRequests = 0;
	await page.route(/\/v1\/profiles\/query$/, async (route) => {
		if (route.request().postDataJSON().filter != null) {
			filteredRequests++;
			if (filteredRequests === 1) {
				await route.fulfill({
					status: 503,
					json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
				});
				return;
			}
			await route.fulfill({ json: { profiles: [profiles[0]], total: 1, hasNext: false } });
			return;
		}
		await route.fulfill({ json: { profiles, total: profiles.length, hasNext: false } });
	});
	await page.setViewportSize({ width: 600, height: 800 });
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const property = editor.locator('.filter-editor__property');
	await property.locator('input').fill('Name');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
	const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = editor.locator('.filter-editor__value-input input');
	await valueInput.fill(JSON.stringify(values[0]));
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	const failure = page.locator('.profiles-list__filter-grid-action');
	await expect(editor).toHaveCount(0);
	await expect(failure).toContainText('Unable to apply filters.');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(3);
	const toolbar = await page.locator('.profiles-list__toolbar').boundingBox();
	const actions = await page.locator('.profiles-list__toolbar-actions').boundingBox();
	expect(actions.x + actions.width).toBeLessThanOrEqual(toolbar.x + toolbar.width);
	await failure.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(1);
	await expect(editor).toHaveCount(0);
	expect(filteredRequests).toBe(2);
	await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
	await expect(valueInput).toHaveValue(JSON.stringify(values[0]));
	await expect(editor.getByRole('button', { name: 'Apply filters', exact: true })).toBeDisabled();
});
