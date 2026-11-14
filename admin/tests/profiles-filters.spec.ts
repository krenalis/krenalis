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
		await expect(applyFilters).toBeEnabled();
		await expect(valueInput).toHaveValue('"');
		await expect(editor.locator('.filter-editor__value-input [role="alert"]')).toContainText('not valid');
		await expect(valueInput).toBeFocused();
		expect(gridFilters).toEqual([null, filter]);
		await expect(page.locator('p[role="alert"]')).toBeEmpty();

		await valueInput.fill(draftValue);
		await propertyInput.focus();
		await expect(propertyInput).toHaveValue('name');
		await propertyInput.fill('');
		await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		await expect(editor).toHaveCount(0);
		await expect(filters.getByRole('button', { name: 'Add filter', exact: true })).toBeVisible();
		await expect(rows).toHaveCount(3);
		expect(gridFilters).toEqual([null, filter, null]);
	});
}

test(`Close an empty filter after clearing its placeholder property`, async ({ page }) => {
	const gridFilters: (Filter | null)[] = [];
	await page.route(/\/v1\/profiles\/query$/, (route) => {
		gridFilters.push(route.request().postDataJSON().filter ?? null);
		return route.fulfill({ json: { profiles, total: profiles.length, hasNext: false } });
	});
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
	await page.addScriptTag({ content: componentScript });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(rows).toHaveCount(3);
	const filters = page.locator('.profiles-list__filters');
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	for (const closeWith of ['Collapse', 'Escape']) {
		await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
		const property = editor.locator('.filter-editor__property');
		const input = property.locator('input');
		await input.fill('nam');
		await input.fill('');
		await expect(editor.getByRole('button', { name: 'Remove condition 1', exact: true })).toHaveCount(0);
		await expect(editor.getByRole('button', { name: 'Apply filters', exact: true })).toBeDisabled();
		await expect(property).toHaveClass(/combobox--open/);
		await input.press('Escape');
		await expect(property).not.toHaveClass(/combobox--open/);
		await expect(editor).toBeVisible();
		if (closeWith === 'Escape') {
			await editor.click({ position: { x: 8, y: 8 } });
			await expect(editor).toBeFocused();
			await page.keyboard.press('Escape');
		} else {
			await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		}
		await expect(editor).toHaveCount(0);
		await expect(filters.getByRole('button', { name: 'Add filter', exact: true })).toBeVisible();
		await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	}

	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
	await editor.getByRole('button', { name: 'Add a condition', exact: true }).click();
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(editor).toBeVisible();
	await expect(editor.locator('.filter-editor__filter')).toHaveCount(2);
	await expect(editor.getByRole('alert')).toHaveText(['Choose a property.', 'Choose a property.']);
	await expect(editor.locator('.filter-editor__property input').first()).toBeFocused();
	await editor.getByRole('button', { name: 'Remove condition 2', exact: true }).click();
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(editor).toHaveCount(0);
	await expect(rows).toHaveCount(3);
	expect(gridFilters).toEqual([null]);
});

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

test(`Validate an invalid date when applying filters and allow correcting it`, async ({ page }) => {
	const dateSchema: ObjectType = {
		...schema,
		properties: [
			...schema.properties,
			{
				...schema.properties[0],
				name: 'personal_data',
				displayName: 'Personal data',
				type: {
					kind: 'object',
					properties: [
						{
							...schema.properties[0],
							name: 'birth_date',
							displayName: 'Birth date',
							type: { kind: 'date' },
						},
					],
				},
			},
		],
	};
	const gridFilters: (Filter | null)[] = [];
	await page.route('**/v1/profiles/schema', (route) =>
		route.fulfill({
			json: {
				schema: dateSchema,
				propertyRoles: { firstName: 'name', lastName: '', email: '', country: '', photo: '' },
				primarySources: {},
			},
		}),
	);
	await page.route(/\/v1\/profiles\/query$/, async (route) => {
		const filter: Filter | null = route.request().postDataJSON().filter ?? null;
		gridFilters.push(filter);
		const result = filter == null ? profiles : profiles.slice(0, 1);
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
	await property.locator('input').fill('Birth date');
	await property
		.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Personal data › Birth date$/ })
		.click();
	const operator = editor.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = editor.locator('.filter-editor__value-input input');
	await valueInput.fill('13/11/2026');
	const applyFilters = editor.getByRole('button', { name: 'Apply filters', exact: true });
	await expect(applyFilters).toBeEnabled();
	const valueControl = editor.locator('.filter-editor__value-input');
	const error = valueControl.getByRole('alert');
	await valueInput.press('Escape');
	await expect(editor).toBeVisible();
	await operator.focus();
	await expect(error).toHaveCount(0);
	await applyFilters.click();
	await expect(error).toHaveText('The filter value on the property Personal data › Birth date is not a valid date.');
	await expect(error.locator('strong')).toHaveText('Personal data › Birth date');
	await expect(valueControl).toHaveAttribute('aria-invalid', 'true');
	await expect(valueInput).toBeFocused();
	await expect(valueInput).toHaveValue('13/11/2026');
	await expect(rows).toHaveCount(3);
	expect(gridFilters).toEqual([null]);

	await valueInput.fill('2026-11-');
	await expect(error).toBeVisible();
	await valueInput.fill('2026-11-13');
	await expect(error).toHaveCount(0);
	await expect(valueControl).not.toHaveAttribute('aria-invalid');
	await valueInput.press('Escape');
	await expect(editor).toBeVisible();
	await applyFilters.click();
	await expect(rows).toHaveCount(1);
	await expect(rows).toHaveAttribute('data-id', 'profile-1');
	await expect(applyFilters).toBeDisabled();
	await expect(page.locator('p[role="alert"]')).toBeEmpty();
	expect(gridFilters).toEqual([
		null,
		{ operator: 'and', rules: [{ property: 'personal_data.birth_date', operator: 'is', values: ['2026-11-13'] }] },
	]);
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
	test(`Apply filters with ${shortcut} from the editor or after a passive click`, async ({ page }) => {
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
		const drawerSchema: ObjectType = {
			...schema,
			properties: [
				...schema.properties,
				{
					...schema.properties[0],
					name: 'details',
					displayName: 'Details',
					description: '',
					readOptional: true,
					type: { kind: 'object', properties: schema.properties },
				},
			],
		};
		await page.route('**/v1/profiles/schema', (route) =>
			route.fulfill({
				json: {
					schema: drawerSchema,
					propertyRoles: { firstName: 'name', lastName: '', email: '', country: '', photo: '' },
					primarySources: {},
				},
			}),
		);
		await page.route('**/v1/profiles/*/attributes/query', (route) =>
			route.fulfill({
				json: { attributes: { ...profiles[0].attributes, details: { name: 'Sample detail' } } },
			}),
		);
		await page.route(/\/v1\/profiles\/query$/, async (route) => {
			const filter: Filter | null = route.request().postDataJSON().filter ?? null;
			gridFilters.push(filter);
			if (filter != null) {
				await gridCanFinish;
				const value = (filter.rules[0] as FilterCondition).values?.[0];
				const result = profiles.filter((profile) => profile.attributes.name === value);
				await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
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
		await expect(applyFilters).toBeEnabled();
		const valueError = editor.locator('.filter-editor__value-input [role="alert"]');
		await expect(valueError).toHaveText('The filter value on the property Name cannot be empty.');
		await expect(valueError.locator('strong')).toHaveText('Name');
		await expect(valueInput).toBeFocused();
		expect(gridFilters).toEqual([null]);
		await valueInput.fill(JSON.stringify(values[1]));
		await valueInput.press('Enter');
		await expect(hint).not.toHaveAttribute('aria-disabled');
		const grid = page.locator('.profiles-list .grid');
		await grid.focus();
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		await expect(hint).toHaveClass(/grid-keyboard-hints__hint--disabled/);
		await expect(applyFilters).toBeEnabled();
		const header = grid.getByRole('columnheader', { name: 'Name', exact: true });
		await header.click();
		await expect(hint).not.toHaveAttribute('aria-disabled');
		const columns = page.locator('.profiles-list__toggle-columns');
		await columns.getByRole('button', { name: 'Columns', exact: true }).click();
		const columnSearch = columns.getByRole('textbox', { name: 'Search columns', exact: true });
		await columnSearch.fill('Name');
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		await columns.evaluate((dropdown: any) => (dropdown.open = false));
		await header.click();
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await page.keyboard.down(modifier);
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await page.keyboard.up(modifier);
		await grid.press('ArrowDown');
		const drawer = page.locator('.profile-drawer');
		await expect(drawer).toBeVisible();
		await expect(grid).toHaveAttribute('aria-activedescendant', 'profiles-grid-row-profile-1');
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		await header.click();
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await rows.first().click();
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		await page.keyboard.press(`${modifier}+Enter`);
		expect(gridFilters).toEqual([null]);
		await drawer.locator('.profile-drawer__kpid-value').click();
		expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
		await expect(hint).not.toHaveAttribute('aria-disabled');
		const details = drawer.locator('.drawer-nested-attributes');
		await details.locator('.drawer-attributes--parent .profile-drawer__attribute-key').click();
		await expect(details).toHaveClass(/drawer-nested-attributes--expand/);
		expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
		await expect(hint).toHaveAttribute('aria-disabled', 'true');
		await valueInput.fill(JSON.stringify(values[0]));
		await page.clock.pauseAt(new Date('2026-11-12T12:01:00Z'));
		try {
			await editor.click({ position: { x: 8, y: 8 } });
			await expect(editor).toBeFocused();
			// Plain Enter triggers :focus-visible for the outline check below.
			await page.keyboard.press('Enter');
			await expect(applyFilters).toBeEnabled();
			await expect(editor).toHaveCSS('outline-style', 'none');
			await header.click();
			await expect(grid).toBeFocused();
			await expect(hint).not.toHaveAttribute('aria-disabled');
			await page.keyboard.press(`${modifier}+Enter`);
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
		await valueInput.fill(JSON.stringify(values[1]));
		await valueInput.press(`${modifier}+Enter`);
		await expect.poll(() => gridFilters.length).toBe(3);
		await expect(rows).toHaveAttribute('data-id', 'profile-2');
		await page.clock.runFor(400);
		await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toHaveCount(0);
		const noMatchValue = 'No matching profile';
		await valueInput.fill(noMatchValue);
		await page.getByRole('heading', { name: 'Profiles', exact: true }).click();
		expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await expect(hint).not.toHaveClass(/grid-keyboard-hints__hint--disabled/);
		await page.keyboard.press(`${modifier}+Enter`);
		await expect.poll(() => gridFilters.length).toBe(4);
		await expect(rows).toHaveCount(0);
		await page.clock.runFor(400);
		await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toHaveCount(0);
		await valueInput.fill(JSON.stringify(values[2]));
		await grid.getByText('No profiles to show', { exact: true }).click();
		await expect(grid).toBeFocused();
		await expect(hint).not.toHaveAttribute('aria-disabled');
		await page.keyboard.press(`${modifier}+Enter`);
		await expect.poll(() => gridFilters.length).toBe(5);
		await expect(rows).toHaveAttribute('data-id', 'profile-3');
		await page.clock.runFor(400);
		await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toHaveCount(0);
		await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
		await expect(hint).not.toBeVisible();
		expect(gridFilters).toEqual([
			null,
			{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [values[0]] }] },
			{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [values[1]] }] },
			{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [noMatchValue] }] },
			{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: [values[2]] }] },
		]);
	});
}

test(`Reapply filters with the shortcut after a failed collapse on a narrow viewport`, async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.addInitScript(() => {
		Object.defineProperty(navigator, 'platform', { value: 'Win32' });
	});
	const filter: Filter = {
		operator: 'and',
		rules: [{ property: 'name', operator: 'is', values: [values[0]] }],
	};
	const gridFilters: (Filter | null)[] = [];
	await page.route(/\/v1\/profiles\/query$/, async (route) => {
		const requestedFilter: Filter | null = route.request().postDataJSON().filter ?? null;
		gridFilters.push(requestedFilter);
		if (gridFilters.length === 2) {
			await route.fulfill({
				status: 503,
				json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
			});
			return;
		}
		const result = requestedFilter == null ? profiles : [profiles[0]];
		await route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
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
	await expect(failure).toContainText('Unable to apply filters');
	await expect(page.getByRole('button', { name: 'Retry', exact: true })).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(3);
	const toolbar = await page.locator('.profiles-list__toolbar').boundingBox();
	const actions = await page.locator('.profiles-list__toolbar-actions').boundingBox();
	expect(actions.x + actions.width).toBeLessThanOrEqual(toolbar.x + toolbar.width);
	await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
	const error = editor.locator('.profiles-list__filter-grid-error');
	const applyFilters = editor.getByRole('button', { name: 'Apply filters', exact: true });
	await expect(error).toHaveText('Unable to apply filters');
	await expect(applyFilters).toBeEnabled();
	await expect(editor.locator('.profiles-list__filter-shortcut')).not.toHaveAttribute('aria-disabled');
	const errorLayout = await error.boundingBox();
	const buttonLayout = await applyFilters.boundingBox();
	expect(errorLayout.x + errorLayout.width).toBeLessThan(buttonLayout.x);
	expect(
		Math.abs(errorLayout.y + errorLayout.height / 2 - buttonLayout.y - buttonLayout.height / 2),
	).toBeLessThanOrEqual(1);
	await page.keyboard.press('Control+Enter');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(1);
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveAttribute('data-id', 'profile-1');
	await expect(valueInput).toHaveValue(JSON.stringify(values[0]));
	await expect(applyFilters).toBeDisabled();
	await expect(error).toHaveCount(0);
	expect(gridFilters).toEqual([null, filter, filter]);
});

test(`Clear all filters reloads an incompatible schema without confirmation`, async ({ page }) => {
	const updatedSchema: ObjectType = {
		...schema,
		properties: [{ ...schema.properties[0], name: 'email', displayName: 'Email' }],
	};
	let schemaReads = 0;
	await page.route('**/v1/profiles/schema', (route) => {
		schemaReads++;
		return route.fulfill({
			json: {
				schema: schemaReads === 1 ? schema : updatedSchema,
				propertyRoles: { firstName: '', lastName: '', email: '', country: '', photo: '' },
				primarySources: {},
			},
		});
	});
	const gridFilters: (Filter | null)[] = [];
	await page.route(/\/v1\/profiles\/query$/, (route) => {
		const filter: Filter | null = route.request().postDataJSON().filter ?? null;
		gridFilters.push(filter);
		if (filter != null) {
			return route.fulfill({
				status: 422,
				json: { error: { code: 'SchemaNotAligned', message: '«name» property no longer exists' } },
			});
		}
		const result =
			schemaReads === 1
				? profiles
				: profiles.map((profile) => ({ ...profile, attributes: { email: `${profile.kpid}@example.test` } }));
		return route.fulfill({ json: { profiles: result, total: result.length, hasNext: false } });
	});
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
	await editor.locator('.filter-editor__value-input input').fill('Ada');
	await editor.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await expect(editor.locator('.profiles-list__filter-grid-error')).toHaveText('Unable to apply filters');
	const notice = page.locator('.profiles-list__stale-notice');
	await expect(notice).toHaveClass(/error-notice/);
	await expect(notice).toHaveAttribute('role', 'alert');
	await expect(notice).toContainText('The profile schema has changed.');
	await expect(notice).toContainText(
		'The property Name no longer exists. Your filters need to be cleared to continue.',
	);
	await expect(notice.locator('strong')).toHaveText('Name');
	const iconLayout = await notice.locator(':scope > wa-icon').boundingBox();
	const titleLayout = await notice.locator('.error-notice__title').boundingBox();
	expect(Math.abs(iconLayout.y + iconLayout.height / 2 - titleLayout.y - titleLayout.height / 2)).toBeLessThanOrEqual(
		1,
	);
	await expect(page.getByRole('button', { name: 'Retry', exact: true })).toHaveCount(0);
	await notice.getByRole('button', { name: 'Clear all filters', exact: true }).click();
	await expect.poll(() => schemaReads).toBe(2);
	await expect(page.locator('wa-dialog[open]')).toHaveCount(0);
	await expect(notice).toHaveCount(0);
	await expect(filters).toContainText('No filters applied');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(3);
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
	await expect(editor.locator('.profiles-list__filter-grid-error')).toHaveCount(0);
	expect(gridFilters).toEqual([
		null,
		{ operator: 'and', rules: [{ property: 'name', operator: 'is', values: ['Ada'] }] },
		null,
	]);
});
