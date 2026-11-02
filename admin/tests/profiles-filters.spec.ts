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
			description: '',
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
		await page.route(/\/v1\/profiles(?:\/count)?\?/, async (route) => {
			const url = new URL(route.request().url());
			const requestedFilter: Filter | null = JSON.parse(url.searchParams.get('filter') ?? 'null');
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
		await filters.getByRole('button', { name: 'Edit', exact: true }).click();
		const editor = filters.getByRole('region', { name: 'Profile filters' });
		const property = editor.locator('.pipeline__filters-property');
		await property.locator('wa-input').click();
		await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Name$/ }).click();
		const operator = editor.locator('.pipeline__filters-operator').getByRole('combobox');
		await operator.press('Home');
		await operator.press('Enter');
		const valueInput = editor.locator('.pipeline__filters-value-input input');
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

		await filters.getByRole('button', { name: 'Done', exact: true }).click();
		await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeVisible();
		await expect(rows).toHaveCount(1);
		await expect(rows).toHaveAttribute('data-id', `profile-${index + 1}`);
		expect(gridFilters).toEqual([null, filter]);

		await filters.getByRole('button', { name: 'Edit', exact: true }).click();
		await expect(valueInput).toHaveValue(draftValue);
		await valueInput.fill('"');
		await filters.getByRole('button', { name: 'Done', exact: true }).click();
		await filters.getByRole('button', { name: 'Edit', exact: true }).click();
		await expect(valueInput).toHaveValue(draftValue);
		expect(countFilters).toEqual([filter, null, filter]);
		expect(gridFilters).toEqual([null, filter]);
		await expect(page.getByRole('alert')).toBeEmpty();
	});
}
