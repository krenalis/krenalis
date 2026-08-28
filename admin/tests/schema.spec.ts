import { test, expect } from '@playwright/test';
import { login, logout, adminURL, logValidationErrors } from './utils';
import { ObjectType, Property } from '../src/lib/api/types/types';

const selectPropertyType = async (page, option: string) => {
	const panel = page.locator('.property-panel');
	await expect(panel.locator('.property-type-selector__structure-trigger')).toContainText('one value');
	await panel.locator('.property-type-selector__structure-trigger').click();
	await expect(panel.locator('.property-type-selector__structure-option-label')).toHaveText([
		'one value',
		'array',
		'object',
		'map',
	]);
	await panel.locator('.property-type-selector__structure-trigger').click();
	await panel.locator('.property-type-selector__trigger').click();
	await expect(panel.locator('.property-type-selector__group-label')).toHaveText([
		'Basic values',
		'Date and time',
		'Specialized values',
	]);
	await expect(
		panel.locator(
			'[data-type-option="uuid"] .property-type-selector__option-label, ' +
				'[data-type-option="json"] .property-type-selector__option-label, ' +
				'[data-type-option="ip"] .property-type-selector__option-label',
		),
	).toHaveText(['uuid', 'json', 'ip']);
	await expect(panel.locator('[data-type-option="datetime"] .property-type-selector__option-label')).toHaveText(
		'datetime',
	);
	await panel.locator(`[data-type-option="${option}"]`).click();
};

const editSchema = async (page) => {
	const button = page.locator('.schema-grid__alter-button');
	await expect(button).not.toHaveAttribute('disabled');
	await button.click();
};

const openProperty = async (page, property: string) => {
	const isEditing = new URL(page.url()).pathname.endsWith('/schema/edit');
	const schema = page.locator(isEditing ? '.schema-edit' : '.schema-grid');
	await schema.locator(`.grid__row[data-id="${property}"]`).click();
};

const expandAllObjects = async (page) => {
	const isEditing = new URL(page.url()).pathname.endsWith('/schema/edit');
	await page.click(isEditing ? '.schema-edit__expand-all-button' : '.schema-grid__expand-all-button');
};

const removeProperty = async (page, property: string) => {
	await openProperty(page, property);
	await page.locator('.property-panel__remove').click();
	await page.click('.schema-edit__confirm-remove-property');
};

const getKeyboardHintsBottomGap = async (container) => {
	return container.evaluate((element) => {
		const hints = element.querySelector('.grid-keyboard-hints');
		return Math.abs(element.getBoundingClientRect().bottom - hints.getBoundingClientRect().bottom);
	});
};

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Disable profile schema editing until the schema has loaded`, async ({ page }) => {
	let schemaRequestCount = 0;
	let releaseSchemaReload = () => {};
	const schemaReloadGate = new Promise<void>((resolve) => {
		releaseSchemaReload = resolve;
	});
	let schemaReloadStarted = () => {};
	const schemaReloadPromise = new Promise<void>((resolve) => {
		schemaReloadStarted = resolve;
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		if (route.request().method() === 'GET' && schemaRequestCount++ > 0) {
			schemaReloadStarted();
			await schemaReloadGate;
		}
		await route.continue();
	});
	let latestAlterRequestCount = 0;
	await page.route('**/v1/profiles/schema/latest-alter', async (route) => {
		latestAlterRequestCount++;
		await route.fulfill({
			json: {
				startTime: '2026-08-25T00:00:00Z',
				endTime: latestAlterRequestCount === 1 ? null : '2026-08-25T00:00:01Z',
				error: null,
			},
		});
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	const editSchemaButton = page.locator('.schema-grid__alter-button');
	try {
		await schemaReloadPromise;
		await expect(editSchemaButton).toHaveJSProperty('loading', false);
		await expect(editSchemaButton).toHaveAttribute('disabled');
	} finally {
		releaseSchemaReload();
	}
	await expect(editSchemaButton).not.toHaveAttribute('disabled', { timeout: 10000 });
});

// Search profile schema properties by the technical type shown in the list.
test(`Search profile schema properties by technical type`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	const searchButton = page.locator('.schema-grid__search-button');
	await expect(searchButton).toBeVisible();
	await searchButton.click();
	const searchInput = page.locator('.schema-grid__search >> input');
	await expect(searchInput).toBeFocused();
	await searchInput.fill('string');
	await expect(page.locator('.grid__row[data-id="email"]')).toBeVisible();
	await expect(page.locator('.grid__row[data-id="favorite_movie.length"]')).toHaveCount(0);
	await searchInput.fill('no_matching_property');
	await expect(
		page.locator('.schema-grid').getByText('No properties match your search', { exact: true }),
	).toBeVisible();
	await page.locator('.schema-grid__search [part="clear-button"]').click();
	await expect(searchInput).toHaveValue('');
	await expect(page.locator('.grid__row[data-id="favorite_movie.length"]')).toHaveCount(1);
	await expect(searchInput).toBeFocused();
	await page.locator('.schema-grid__expand-all-button').click();
	await expect(searchInput).toHaveCount(0);
	await expect(searchButton).toBeVisible();

	await editSchema(page);
	const editSearchButton = page.locator('.schema-edit__search-button');
	await editSearchButton.click();
	const editSearchInput = page.locator('.schema-edit__search >> input');
	await expect(editSearchInput).toBeFocused();
	await editSearchInput.fill('string');
	await expect(page.locator('.schema-edit .grid__row[data-id="email"]')).toBeVisible();
	await expect(page.locator('.schema-edit .grid__row[data-id="favorite_movie.length"]')).toHaveCount(0);
	await editSearchInput.fill('no_matching_property');
	await expect(
		page.locator('.schema-edit').getByText('No properties match your search', { exact: true }),
	).toBeVisible();
	await expect(editSearchInput).toBeFocused();
	await page.locator('.schema-edit__search [part="clear-button"]').click();
	await expect(editSearchInput).toHaveValue('');
	await expect(page.locator('.schema-edit .grid__row[data-id="favorite_movie.length"]')).toHaveCount(1);
	await expect(editSearchInput).toBeFocused();
	await page.locator('.schema-edit__expand-all-button').click();
	await expect(editSearchInput).toHaveCount(0);
	await openProperty(page, 'email');
	const firstVisiblePropertyKey = await page
		.locator('.schema-edit .grid__row--clickable:visible')
		.first()
		.getAttribute('data-id');
	expect(firstVisiblePropertyKey).not.toBeNull();

	await page.locator('.schema-edit__filter-button').click();
	const showChanged = page.locator('.schema-edit__show-changed');
	await showChanged.click();
	await expect(page.locator('.schema-edit__filter-dot')).toHaveClass(/schema-edit__filter-dot--active/);
	await expect(page.locator('.schema-edit .grid__row')).toHaveCount(0);
	await expect(page.locator('.property-panel--empty')).toBeEmpty();
	await expect(page.getByText('Reorder', { exact: true }).locator('..')).toHaveAttribute('aria-disabled', 'true');
	await showChanged.click();
	await expect(page.getByText('Reorder', { exact: true }).locator('..')).not.toHaveAttribute('aria-disabled', 'true');
	await expect(page.locator(`.schema-edit .grid__row[data-id="${firstVisiblePropertyKey}"]`)).toHaveClass(
		/grid__row--selected/,
	);
	await expect(page.locator('.property-panel--empty')).toHaveCount(0);
});

test(`Keep keyboard hints fixed and show disabled reorder handles while filtering`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await expect.poll(() => getKeyboardHintsBottomGap(page.locator('.schema-grid__layout'))).toBeLessThan(1);
	await editSchema(page);
	await page.locator('.schema-edit__search-button').click();
	const searchInput = page.locator('.schema-edit__search >> input');
	await searchInput.fill('string');

	const row = page.locator('.schema-edit .grid__row[data-id="email"]');
	await row.click();
	const handle = row.locator('xpath=..').locator('.draggable-wrapper__handle');
	await expect(handle).toBeVisible();
	await expect(handle).toBeDisabled();
	await expect.poll(() => getKeyboardHintsBottomGap(page.locator('.schema-edit__layout'))).toBeLessThan(1);
	await expect
		.poll(() =>
			row.evaluate((element) => {
				const cell = element.querySelector('.grid__cell');
				return getComputedStyle(element).backgroundColor === getComputedStyle(cell).backgroundColor;
			}),
		)
		.toBe(true);

	const visibleRows = page.locator('.schema-edit .grid__row--clickable:visible');
	const rowIDs = await visibleRows.evaluateAll((elements) =>
		elements.map((element: HTMLElement) => element.dataset.id),
	);
	await page.locator('.schema-edit .grid').focus();
	await page.keyboard.press('Shift+ArrowDown');
	await expect
		.poll(() => visibleRows.evaluateAll((elements) => elements.map((element: HTMLElement) => element.dataset.id)))
		.toEqual(rowIDs);

	await searchInput.fill('');
	await expect(handle).toBeEnabled();
});

test(`Expand nested object properties while viewing and editing`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		const property = schema.properties[0];
		schema.properties.push({ ...property, name: 'reference_property', type: { kind: 'string' } });
		schema.properties.push({
			...property,
			name: 'nested_object',
			type: {
				kind: 'object',
				properties: [
					{ ...property, name: 'outer_leaf', type: { kind: 'string' } },
					{
						...property,
						name: 'inner_object',
						type: {
							kind: 'object',
							properties: [{ ...property, name: 'leaf', type: { kind: 'string' } }],
						},
					},
				],
			},
		});
		await route.fulfill({ response, json: schema });
	});

	const getExpandControl = (row) => row.locator('xpath=..').locator(':scope > .grid__row-expand');
	const expectExpandControlToPointDown = async (expand) => {
		await expect
			.poll(() =>
				expand.evaluate((element) => {
					const svg = element.shadowRoot?.querySelector('svg');
					if (svg == null) {
						return null;
					}
					const transform = new DOMMatrixReadOnly(getComputedStyle(svg).transform);
					return [transform.a, transform.b, transform.c, transform.d].map(Math.round).join(',');
				}),
			)
			.toBe('0,1,-1,0');
	};
	const expectExpandControlAfterHandle = async (row) => {
		const nestedRows = row.locator('xpath=..');
		const expand = getExpandControl(row);
		const handle = nestedRows.locator('xpath=..').locator(':scope > .draggable-wrapper__handle');
		await expect(expand).toBeVisible();
		await expect
			.poll(async () => {
				const [expandBox, handleBox] = await Promise.all([expand.boundingBox(), handle.boundingBox()]);
				if (expandBox == null || handleBox == null) {
					return null;
				}
				return Math.round(expandBox.x - handleBox.x - handleBox.width);
			})
			.toBe(2);
		return expand;
	};

	await page.goto(`${adminURL}/profile-unification/schema`);
	const readOnlyOuterRow = page.locator('.schema-grid .grid__row[data-id="nested_object"]');
	const readOnlyOuterExpand = getExpandControl(readOnlyOuterRow);
	await expect(readOnlyOuterExpand).toBeVisible();
	await readOnlyOuterExpand.click({ timeout: 5000 });
	await expectExpandControlToPointDown(readOnlyOuterExpand);
	const readOnlyInnerRow = page.locator('.schema-grid .grid__row[data-id="nested_object.inner_object"]');
	const readOnlyInnerExpand = getExpandControl(readOnlyInnerRow);
	await expect(readOnlyInnerExpand).toBeVisible();
	await expect
		.poll(() =>
			readOnlyInnerRow.evaluate((element) =>
				element.parentElement == null ? null : getComputedStyle(element.parentElement).borderBottomWidth,
			),
		)
		.toBe('0px');
	await readOnlyInnerExpand.click({ timeout: 5000 });
	await expectExpandControlToPointDown(readOnlyInnerExpand);
	await expect(page.locator('.schema-grid .grid__row[data-id="nested_object.inner_object.leaf"]')).toBeVisible();

	await page.reload();
	await editSchema(page);

	const referenceName = page.locator(
		'.schema-edit .grid__row[data-id="reference_property"] .grid__cell.grid-el--first .grid__cell-content',
	);
	const outerRow = page.locator('.schema-edit .grid__row[data-id="nested_object"]');
	const outerExpand = await expectExpandControlAfterHandle(outerRow);
	const outerName = outerRow.locator('.grid__cell.grid-el--first .grid__cell-content');
	await expect
		.poll(async () => {
			const [referenceBox, outerBox] = await Promise.all([referenceName.boundingBox(), outerName.boundingBox()]);
			return referenceBox == null || outerBox == null ? null : Math.round(outerBox.x - referenceBox.x);
		})
		.toBe(0);
	await outerExpand.click({ timeout: 5000 });
	await expectExpandControlToPointDown(outerExpand);
	const innerRow = page.locator('.schema-edit .grid__row[data-id="nested_object.inner_object"]');
	const outerLeaf = page.locator('.schema-edit .grid__row[data-id="nested_object.outer_leaf"]');
	await expect(innerRow).toBeVisible();
	await expect
		.poll(() => innerRow.evaluate((element) => getComputedStyle(element, '::before').display))
		.not.toBe('none');
	const innerName = innerRow.locator('.grid__cell.grid-el--first .grid__cell-content');
	const outerLeafName = outerLeaf.locator('.grid__cell.grid-el--first .grid__cell-content');
	await expect
		.poll(async () => {
			const [innerBox, outerLeafBox] = await Promise.all([innerName.boundingBox(), outerLeafName.boundingBox()]);
			return innerBox == null || outerLeafBox == null ? null : Math.round(innerBox.x - outerLeafBox.x);
		})
		.toBe(0);
	const innerExpand = await expectExpandControlAfterHandle(innerRow);
	await expect
		.poll(async () => {
			const [outerBox, innerBox] = await Promise.all([outerExpand.boundingBox(), innerExpand.boundingBox()]);
			return outerBox == null || innerBox == null
				? null
				: Math.round(innerBox.x + innerBox.width / 2 - outerBox.x - outerBox.width / 2);
		})
		.toBe(0);
	await innerExpand.click({ timeout: 5000 });
	await expectExpandControlToPointDown(innerExpand);
	await expect(page.locator('.schema-edit .grid__row[data-id="nested_object.inner_object.leaf"]')).toBeVisible();
});

test(`Keep profile schema search selection and expansion consistent`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'search_object',
			prefilled: '',
			role: 'Both',
			type: {
				kind: 'object',
				properties: [
					{
						name: 'nested_match',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string' },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
				],
			},
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await openProperty(page, 'email');
	await page.locator('.schema-grid__search-button').click();
	const searchInput = page.locator('.schema-grid__search >> input');
	await searchInput.fill('dummy_id');
	await expect(page.locator('.schema-grid .grid__row[data-id="dummy_id"]')).toHaveClass(/grid__row--selected/);
	await expect(
		page.locator('.property-details-panel__detail').first().locator('.property-details-panel__value'),
	).toHaveText('dummy_id');

	await searchInput.fill('no_matching_property');
	await expect(page.locator('.schema-grid .grid__row--selected')).toHaveCount(0);
	await expect(page.locator('.schema-grid__workspace')).not.toHaveClass(/schema-grid__workspace--with-panel/);
	const readOnlyHints = page.locator('.schema-grid .grid-keyboard-hints__hint');
	await expect(readOnlyHints.nth(0)).toHaveAttribute('aria-disabled', 'true');
	await expect(readOnlyHints.nth(1)).toHaveAttribute('aria-disabled', 'true');
	await expect(page.locator('.schema-grid__expand-all-button')).toHaveAttribute('disabled');
	await expect(page.locator('.schema-grid__collapse-all-button')).toHaveAttribute('disabled');

	await searchInput.fill('nested_match');
	const readOnlyNestedRow = page.locator('.schema-grid .grid__row[data-id="search_object.nested_match"]');
	await expect(readOnlyNestedRow).toBeVisible();
	await expect(readOnlyNestedRow.locator('xpath=..')).toHaveClass(/grid__nested-rows--expanded/);
	await expect(readOnlyHints.nth(0)).not.toHaveAttribute('aria-disabled', 'true');
	await expect(readOnlyHints.nth(1)).toHaveAttribute('aria-disabled', 'true');
	await expect(page.locator('.schema-grid .grid__row--selected')).toHaveCount(0);

	await searchInput.fill('');
	await expect(readOnlyNestedRow).toBeHidden();
	await expect(page.locator('.schema-grid .grid__row--selected')).toHaveCount(0);
	await searchInput.blur();
	await editSchema(page);

	await page.locator('.schema-edit__search-button').click();
	const editSearchInput = page.locator('.schema-edit__search >> input');
	await editSearchInput.fill('nested_match');
	const editNestedRow = page.locator('.schema-edit .grid__row[data-id="search_object.nested_match"]');
	await expect(editNestedRow).toBeVisible();
	await expect(
		editNestedRow.locator(
			'xpath=ancestor::div[contains(concat(" ", normalize-space(@class), " "), " grid__nested-rows ")][1]',
		),
	).toHaveClass(/grid__nested-rows--expanded/);
	await expect(page.locator('.schema-edit__expand-all-button')).toHaveAttribute('disabled');
	await expect(page.locator('.schema-edit__collapse-all-button')).toHaveAttribute('disabled');
	const editHints = page.locator('.schema-edit .grid-keyboard-hints__hint');
	await expect(editHints.nth(0)).not.toHaveAttribute('aria-disabled', 'true');
	await expect(editHints.nth(1)).toHaveAttribute('aria-disabled', 'true');
	await expect(editHints.nth(2)).toHaveAttribute('aria-disabled', 'true');

	await editSearchInput.fill('no_matching_property');
	await expect(page.locator('.property-panel--empty')).toBeEmpty();
	await expect(editHints.nth(0)).toHaveAttribute('aria-disabled', 'true');
	await expect(editHints.nth(1)).toHaveAttribute('aria-disabled', 'true');
	await expect(editHints.nth(2)).toHaveAttribute('aria-disabled', 'true');
	await expect(editSearchInput).toBeFocused();

	await editSearchInput.fill('nested_match');
	await expect(editNestedRow).toBeVisible();
	await editSearchInput.fill('');
	await expect(editNestedRow).toBeHidden();
});

test(`Navigate profile schema properties when focus is outside an arrow-key control`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	const readOnlyRows = page.locator('.schema-grid .grid__row--clickable:visible');
	await expect.poll(() => readOnlyRows.count()).toBeGreaterThan(3);
	const readOnlyRowIDs = await readOnlyRows.evaluateAll((rows) => rows.map((row: HTMLElement) => row.dataset.id));

	const editSchemaButton = page.locator('.schema-grid__alter-button');
	await editSchemaButton.focus();
	await page.keyboard.press('ArrowDown');
	await expect(readOnlyRows.nth(0)).toHaveClass(/grid__row--selected/);
	await expect(page.locator('.schema-grid .grid')).toBeFocused();
	expect(await page.locator('.schema-grid .grid').evaluate((grid) => getComputedStyle(grid).outlineStyle)).toBe(
		'none',
	);
	await editSchemaButton.focus();
	await page.keyboard.press('ArrowDown');
	await expect(readOnlyRows.nth(1)).toHaveClass(/grid__row--selected/);

	await page.locator('.schema-grid__search-button').click();
	const searchInput = page.locator('.schema-grid__search >> input');
	await expect(searchInput).toBeFocused();
	await page.keyboard.press('ArrowDown');
	await expect(readOnlyRows.nth(1)).toHaveClass(/grid__row--selected/);
	await searchInput.blur();

	await editSchemaButton.click();
	const editRows = page.locator('.schema-edit .grid__row--clickable:visible');
	await expect(page.locator(`.schema-edit .grid__row[data-id="${readOnlyRowIDs[1]}"]`)).toHaveClass(
		/grid__row--selected/,
	);

	const cancelButton = page.locator('.schema-edit__header-cancel-button');
	await cancelButton.locator('button').focus();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator(`.schema-edit .grid__row[data-id="${readOnlyRowIDs[2]}"]`)).toHaveClass(
		/grid__row--selected/,
	);

	const description = page.locator('.property-panel sl-textarea >> textarea[name="description"]');
	const modifiedDots = page.locator('.property-panel .property-form__modified-dot');
	const initialDescription = await description.inputValue();
	await description.focus();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator(`.schema-edit .grid__row[data-id="${readOnlyRowIDs[2]}"]`)).toHaveClass(
		/grid__row--selected/,
	);

	const rowIDsBeforeMove = await editRows.evaluateAll((rows) => rows.map((row: HTMLElement) => row.dataset.id));
	await cancelButton.locator('button').focus();
	await page.keyboard.press('Shift+ArrowDown');
	await expect
		.poll(async () => {
			const rowIDs = await editRows.evaluateAll((rows) => rows.map((row: HTMLElement) => row.dataset.id));
			return rowIDs.indexOf(readOnlyRowIDs[2]);
		})
		.toBe(rowIDsBeforeMove.indexOf(readOnlyRowIDs[2]) + 1);
	const movedRow = page.locator(`.schema-edit .grid__row[data-id="${readOnlyRowIDs[2]}"]`);
	await expect(movedRow.locator('.schema-edit__property-actions')).toHaveText('Reordered');
	await expect(modifiedDots).toHaveCount(0);
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');

	await description.fill(`${initialDescription} updated`);
	await page.locator('.property-panel__save').click();
	await expect(movedRow.locator('.schema-edit__property-actions')).toHaveText('Modified');
	await expect(page.locator('.property-panel sl-textarea .property-form__modified-dot')).toBeVisible();
	await expect(modifiedDots).toHaveCount(1);

	await description.fill(initialDescription);
	await page.locator('.property-panel__save').click();
	await expect(movedRow.locator('.schema-edit__property-actions')).toHaveText('Reordered');
	await expect(modifiedDots).toHaveCount(0);

	await cancelButton.locator('button').focus();
	await page.keyboard.press('Shift+ArrowUp');
	await expect
		.poll(async () => {
			const rowIDs = await editRows.evaluateAll((rows) => rows.map((row: HTMLElement) => row.dataset.id));
			return rowIDs.indexOf(readOnlyRowIDs[2]);
		})
		.toBe(rowIDsBeforeMove.indexOf(readOnlyRowIDs[2]));
	await expect(movedRow.locator('.schema-edit__property-actions')).toBeEmpty();
	await expect(page.locator('.schema-edit__change-count')).toContainText('No pending changes');
});

test(`Keep an object expanded when reordering it`, async ({ page }) => {
	let previewRequests = 0;
	page.on('request', (request) => {
		if (request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT') {
			previewRequests++;
		}
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push(
			{
				name: 'expanded_object',
				prefilled: '',
				role: 'Both',
				type: {
					kind: 'object',
					properties: [
						{
							name: 'child',
							prefilled: '',
							role: 'Both',
							type: { kind: 'string' },
							createRequired: false,
							updateRequired: false,
							readOptional: true,
							nullable: false,
							description: '',
						},
					],
				},
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
			{
				name: 'following_property',
				prefilled: '',
				role: 'Both',
				type: { kind: 'string' },
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
		);
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	const objectRow = page.locator('.schema-edit .grid__row[data-id="expanded_object"]');
	const objectGroup = objectRow.locator('xpath=..');
	await objectRow.locator('xpath=preceding-sibling::*[contains(@class, "grid__row-expand")]').click();
	await expect(objectGroup).toHaveClass(/grid__nested-rows--expanded/);

	await page.keyboard.press('Shift+ArrowDown');
	await expect(objectGroup).toHaveClass(/grid__nested-rows--expanded/);
	await expect(page.locator('.schema-edit .grid__row[data-id="expanded_object.child"]')).toBeVisible();
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('loading');
	await page.waitForTimeout(500);
	expect(previewRequests).toBe(0);

	const previewResponse = page.waitForResponse(
		(response) => response.url().endsWith('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await previewResponse;
	await expect(page.locator('.schema-edit__queries')).toBeVisible();
});

test(`Warn before leaving pending profile schema changes`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	const cancelButton = page.locator('.schema-edit__header-cancel-button');
	await cancelButton.focus();
	await page.keyboard.press('Shift+ArrowDown');
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');

	await page.evaluate(() => window.history.back());
	const dialog = page.locator('.alert-dialog', { hasText: 'Discard unsaved changes?' });
	await expect(dialog).toBeVisible();
	await expect(dialog).toContainText('The pending schema changes will be discarded.');
	await dialog.getByText('Keep editing', { exact: true }).click();
	await expect(page.locator('.schema-edit')).toBeVisible();
	await expect(dialog).not.toBeVisible();

	await page.evaluate(() => window.history.back());
	await expect(dialog).toBeVisible();
	await dialog.getByText('Discard and leave', { exact: true }).click();
	await expect(page.locator('.schema-grid')).toBeVisible();
});

test(`Restore pointer interaction after discarding schema changes with Cancel`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	const cancelButton = page.locator('.schema-edit__header-cancel-button');
	await cancelButton.focus();
	await page.keyboard.press('Shift+ArrowDown');
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');

	await cancelButton.click();
	const dialog = page.locator('.alert-dialog', { hasText: 'Discard unsaved changes?' });
	await expect(dialog).toBeVisible();
	await dialog.getByText('Discard and leave', { exact: true }).click();
	await expect(page.locator('.schema-grid')).toBeVisible();

	const row = page.locator('.schema-grid .grid__row--clickable:visible').nth(1);
	await row.click();
	await expect(row).toHaveClass(/grid__row--selected/);
	await expect(page.locator('.property-details-panel')).toBeVisible();
});

test(`Keep an unsaved property selected when selecting another property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	const applyButton = page.locator('.schema-edit__header-apply-button');

	await openProperty(page, 'email');
	const propertyPanel = page.locator('.property-panel');
	await expect(propertyPanel.locator('.property-panel__header sl-icon-button')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-panel__cancel')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-panel__save')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-panel__remove')).toBeVisible();
	const description = page.locator('.property-panel sl-textarea >> textarea[name="description"]');
	await description.fill('Unsaved description');
	await expect(propertyPanel.locator('.property-panel__cancel')).toHaveText('Cancel');
	await expect(propertyPanel.locator('.property-panel__save')).toHaveText('Confirm');
	await expect(propertyPanel.locator('.property-panel__remove')).toHaveCount(0);
	await page.waitForTimeout(100);
	await openProperty(page, 'phone_numbers');
	await expect(page.locator('.schema-edit .grid__row[data-id="email"]')).toHaveClass(/grid__row--selected/);
	await expect(page.locator('.schema-edit .grid__row[data-id="phone_numbers"]')).not.toHaveClass(
		/grid__row--selected/,
	);
	await expect(propertyPanel.locator('sl-animation')).toHaveJSProperty('play', true);
	await propertyPanel.locator('.property-panel__cancel').click();
	await expect(applyButton).toHaveAttribute('disabled');
});

test(`Keep contextual and form actions in their expected tab order`, async ({ page }) => {
	await page.route('**/v1/connections', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.connections.push({
			id: '9zQ4Tn7B3mS6',
			name: 'Schema test source',
			connector: 'dummy',
			connectorType: 'SDK',
			role: 'Source',
			storage: '',
			compression: '',
			strategy: null,
			sendingMode: null,
			hasSettings: false,
			health: 'Healthy',
			linkedConnections: null,
		});
		await route.fulfill({ response, json: body });
	});
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'email');

	const propertyPanel = page.locator('.property-panel');
	const primarySource = propertyPanel.locator('.property-form__primary-source');
	const primarySourceInput = primarySource.locator('[part="display-input"]');
	const removeButton = propertyPanel.locator('.property-panel__remove');
	await primarySourceInput.focus();
	await expect(primarySourceInput).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(primarySourceInput).not.toBeFocused();
	await expect(removeButton).not.toBeFocused();

	await propertyPanel.locator('sl-textarea textarea[name="description"]').fill('Unsaved description');
	await expect(propertyPanel.locator('.property-panel__cancel')).toBeVisible();
	await primarySourceInput.focus();
	await expect(primarySourceInput).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(propertyPanel.locator('.property-panel__cancel')).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(propertyPanel.locator('.property-panel__save')).toBeFocused();
	await propertyPanel.locator('.property-panel__cancel').click();
});

test(`Keep an unsaved property visible while filtering`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'email');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('sl-textarea textarea[name="description"]').fill('Unsaved description');
	await page.locator('.schema-edit__search-button').click();
	await page.locator('.schema-edit__search >> input').fill('dummy_id');

	await expect(page.locator('.schema-edit .grid__row[data-id="email"]')).toHaveClass(/grid__row--selected/);
	await expect(page.locator('.schema-edit .grid__row[data-id="dummy_id"]')).toBeVisible();
	await expect(page.getByText('Reorder', { exact: true }).locator('..')).toHaveAttribute('aria-disabled', 'true');

	await propertyPanel.locator('.property-panel__cancel').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="email"]')).toHaveCount(0);
	await expect(page.locator('.schema-edit .grid__row[data-id="dummy_id"]')).toHaveClass(/grid__row--selected/);
});

test(`View property details and keep the selection when editing`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'address',
			prefilled: '',
			role: 'Both',
			type: {
				kind: 'object',
				properties: [
					{
						name: 'country',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string', maxLength: 2 },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
				],
			},
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await openProperty(page, 'email');
	let panel = page.locator('.property-details-panel');
	await expect(panel.locator('.property-panel__title')).toHaveText('Property');
	const emailRow = page.locator('.grid__row[data-id="email"]');
	await expect(panel.locator('.property-details-panel__label')).toContainText([
		'Name',
		'Type',
		'Identity resolution',
		'Description',
		'Primary source',
	]);
	const emailCells = await emailRow.locator('.grid__cell-content').allInnerTexts();
	await expect(panel.locator('.property-details-panel__value')).toHaveText([
		emailCells[0],
		emailCells[1],
		'Not an identifier',
		emailCells[3],
		emailCells[4],
	]);
	const gridTypeFont = await emailRow
		.locator('.schema-grid__technical-type')
		.evaluate((type) => getComputedStyle(type).fontFamily);
	expect(
		await panel
			.locator('.property-details-panel__technical-type')
			.evaluate((type) => getComputedStyle(type).fontFamily),
	).toBe(gridTypeFont);
	await expect(emailRow).toHaveClass(/grid__row--selected/);
	await panel.getByLabel('Close property details').click();
	await expect(panel).toHaveCount(0);

	const addressRow = page.locator('.schema-grid .grid__row[data-id="address"]');
	await addressRow.locator('xpath=preceding-sibling::*[contains(@class, "grid__row-expand")]').click();
	await expect(addressRow).toHaveClass(/grid__row--selected/);
	panel = page.locator('.property-details-panel');
	await expect(
		panel.locator('.property-details-panel__detail').first().locator('.property-details-panel__value'),
	).toHaveText('address');
	await page.locator('.schema-grid__page-header h1').hover();
	const addressBackground = await addressRow
		.locator('.grid__cell')
		.first()
		.evaluate((cell) => {
			cell.getAnimations().forEach((animation) => animation.finish());
			return getComputedStyle(cell).backgroundColor;
		});
	await addressRow.hover();
	expect(
		await addressRow
			.locator('.grid__cell')
			.first()
			.evaluate((cell) => {
				cell.getAnimations().forEach((animation) => animation.finish());
				return getComputedStyle(cell).backgroundColor;
			}),
	).toBe(addressBackground);

	await openProperty(page, 'address.country');
	panel = page.locator('.property-details-panel');
	await expect(
		panel.locator('.property-details-panel__detail').first().locator('.property-details-panel__value'),
	).toHaveText('country');
	await expect(panel).toContainText('string, max 2 chars');
	const countryRow = page.locator('.schema-grid .grid__row[data-id="address.country"]');
	await page.locator('.schema-grid__page-header h1').hover();
	const countryBackground = await countryRow
		.locator('.grid__cell')
		.first()
		.evaluate((cell) => {
			cell.getAnimations().forEach((animation) => animation.finish());
			return getComputedStyle(cell).backgroundColor;
		});
	await countryRow.hover();
	expect(
		await countryRow
			.locator('.grid__cell')
			.first()
			.evaluate((cell) => {
				cell.getAnimations().forEach((animation) => animation.finish());
				return getComputedStyle(cell).backgroundColor;
			}),
	).toBe(countryBackground);

	await editSchema(page);
	await expect(page.locator('.schema-grid__workspace')).toHaveClass(/schema-grid__workspace--with-panel/);
	await expect(panel).toHaveCount(1);
	await expect(page.locator('.schema-edit .property-panel .property-panel__title')).toHaveText('Property');
	await expect(page.locator('.property-panel .property-form__name-input')).toHaveJSProperty('value', 'country');
	const selectedRow = page.locator('.schema-edit .grid__row[data-id="address.country"]');
	await expect(selectedRow).toBeVisible();
	await expect(selectedRow).toHaveClass(/grid__row--selected/);
});

test(`Show identifier order and renumber identifiers after removing a property`, async ({ page }) => {
	await page.route('**/v1/workspaces', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		for (const workspace of body.workspaces) {
			workspace.identifiers = ['email', 'first_name', 'last_name'];
		}
		await route.fulfill({ response, json: body });
	});

	const identifierCell = (container, property: string) =>
		container.locator(`.grid__row[data-id="${property}"] .grid__cell-content`).nth(2);

	await page.goto(`${adminURL}/profile-unification/schema`);
	const readOnlyGrid = page.locator('.schema-grid');
	await expect(identifierCell(readOnlyGrid, 'email')).toHaveText('#1');
	await expect(identifierCell(readOnlyGrid, 'first_name')).toHaveText('#2');
	await expect(identifierCell(readOnlyGrid, 'last_name')).toHaveText('#3');
	await expect(identifierCell(readOnlyGrid, 'phone_numbers')).toBeEmpty();

	await openProperty(page, 'email');
	const identifierDetail = page
		.locator('.property-details-panel__detail')
		.filter({ hasText: 'Identity resolution' })
		.locator('.property-details-panel__value');
	await expect(identifierDetail).toHaveText('Identifier #1');
	await expect(identifierDetail.locator('.schema-property-grid__identifier')).toHaveText('#1');
	await openProperty(page, 'phone_numbers');
	await expect(
		page.locator('.property-details-panel__label').filter({ hasText: 'Identity resolution' }),
	).toHaveCount(0);
	await openProperty(page, 'email');

	await editSchema(page);
	const editGrid = page.locator('.schema-edit');
	await expect(identifierCell(editGrid, 'email')).toHaveText('#1');
	await expect(identifierCell(editGrid, 'phone_numbers')).toBeEmpty();
	await expect(
		page.locator('.property-form__read-only-value .schema-property-grid__identifier'),
	).toHaveText('#1');
	await openProperty(page, 'phone_numbers');
	await expect(page.locator('.property-form__label').filter({ hasText: 'Identity resolution' })).toHaveCount(0);

	await removeProperty(page, 'email');
	await expect(identifierCell(editGrid, 'first_name')).toHaveText('#1');
	await expect(identifierCell(editGrid, 'last_name')).toHaveText('#2');

	await page.locator('.schema-edit__add-property').click();
	const propertyPanel = page.locator('.property-panel');
	await expect(propertyPanel.locator('.property-form__label').filter({ hasText: 'Identity resolution' })).toHaveCount(0);
	await propertyPanel.locator('.property-form__name-input input').fill('email');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(identifierCell(editGrid, 'email')).toBeEmpty();
	await expect(propertyPanel.locator('.property-form__label').filter({ hasText: 'Identity resolution' })).toHaveCount(0);
});

test(`Keep object types unchanged after canceling the schema review`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'review_object',
			prefilled: '',
			role: 'Both',
			type: {
				kind: 'object',
				properties: [
					{
						name: 'child',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string' },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
				],
			},
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await expandAllObjects(page);
	await openProperty(page, 'review_object.child');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('sl-textarea textarea[name="description"]').fill('Updated child');
	await propertyPanel.locator('.property-panel__save').click();
	await page.locator('.schema-edit__header-apply-button').click();

	const reviewDialog = page.locator('.schema-edit__queries');
	await expect(reviewDialog).toBeVisible();
	await reviewDialog.getByText('Cancel', { exact: true }).click();
	await expect(reviewDialog).not.toBeVisible();

	await openProperty(page, 'review_object');
	await expect(propertyPanel.locator('.property-form__modified-dot')).toHaveCount(0);
});

test(`Keep the schema review closed when its preview finishes`, async ({ page }) => {
	let previewRequestCount = 0;
	let finishPreview = () => {};
	const previewResponse = new Promise<void>((resolve) => {
		finishPreview = resolve;
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		previewRequestCount++;
		if (previewRequestCount === 1) {
			await route.fulfill({ status: 500, contentType: 'application/json', body: '{}' });
			return;
		}
		await previewResponse;
		await route.fulfill({ json: { queries: ['SELECT 1'] } });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('property_with_delayed_preview');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();

	const reviewDialog = page.locator('.schema-edit__queries');
	const failedPreviewResponse = page.waitForResponse(
		(response) => response.url().endsWith('/profiles/schema/preview') && response.status() === 500,
	);
	await page.locator('.schema-edit__header-apply-button').click();
	await failedPreviewResponse;
	await expect(reviewDialog).toHaveJSProperty('open', false);

	const previewResponsePromise = page.waitForResponse(
		(response) => response.url().endsWith('/profiles/schema/preview') && response.status() === 200,
	);
	await page.locator('.schema-edit__header-apply-button').click();
	await expect(reviewDialog).toHaveJSProperty('open', true);
	await page.keyboard.press('Escape');
	await expect(reviewDialog).toHaveJSProperty('open', false);

	finishPreview();
	await previewResponsePromise;
	await page.waitForTimeout(400);
	await expect(reviewDialog).toHaveJSProperty('open', false);

	await page.locator('.schema-edit__header-cancel-button').click();
	await page.locator('.alert-dialog', { hasText: 'Discard unsaved changes?' }).getByText('Discard and leave').click();
	await expect(page.locator('.schema-edit')).toHaveCount(0);
});

test(`Keep the schema review open while applying changes`, async ({ page }) => {
	let finishAlter = () => {};
	const alterResponse = new Promise<void>((resolve) => {
		finishAlter = resolve;
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		if (route.request().method() !== 'PUT') {
			await route.continue();
			return;
		}
		await alterResponse;
		await route.fulfill({ json: {} });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'email');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('sl-textarea textarea[name="description"]').fill('Updated description');
	await propertyPanel.locator('.property-panel__save').click();
	await page.locator('.schema-edit__header-apply-button').click();

	const reviewDialog = page.locator('.schema-edit__queries');
	await expect(reviewDialog).toBeVisible();
	const alterRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema') && request.method() === 'PUT',
	);
	await page.locator('.schema-edit__apply-alter-button').click();
	await alterRequestPromise;

	await expect(reviewDialog.getByText('Cancel', { exact: true })).toHaveAttribute('disabled');
	await page.keyboard.press('Escape');
	await expect(reviewDialog).toHaveJSProperty('open', true);

	finishAlter();
	await expect(page.locator('.schema-edit')).toHaveCount(0);
});

test(`Preserve create-required on top-level properties in the schema preview`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'create_required_property',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: true,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('preview_trigger');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();
	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await page.locator('.schema-edit__header-apply-button').click();
	const previewRequest = await previewRequestPromise;
	const previewSchema = previewRequest.postDataJSON().schema as ObjectType;
	const property = previewSchema.properties.find((candidate) => candidate.name === 'create_required_property');
	expect(property).toHaveProperty('createRequired', true);
	expect(property).not.toHaveProperty('createRequire');
});

test(`Preview schema changes only when applying them`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
	let previewRequests = 0;
	page.on('request', (request) => {
		if (request.url().includes('/profiles/schema/preview') && request.method() === 'PUT') {
			previewRequests++;
		}
	});
	await openProperty(page, 'email');
	const propertyPanel = page.locator('.property-panel');
	const description = propertyPanel.locator('sl-textarea textarea[name="description"]');
	const originalDescription = await description.inputValue();
	await description.fill('Updated description');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await propertyPanel.locator('.property-panel__save').click();
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('disabled');
	await expect(applyButton).not.toHaveAttribute('loading');
	await page.waitForTimeout(1000);
	expect(previewRequests).toBe(0);

	const metadataPreviewResponse = page.waitForResponse(
		(response) => response.url().includes('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await metadataPreviewResponse;
	const dialog = page.locator('.schema-edit__queries');
	await expect(dialog).toHaveAttribute('label', 'Apply schema changes?');
	await expect(dialog.locator('.schema-edit__no-query')).toHaveText(
		'These changes affect only the schema definition. The data warehouse will not be modified.',
	);
	await expect(dialog.locator('.schema-edit__apply-alter-button')).toHaveAttribute('variant', 'primary');
	await dialog.locator('.schema-edit__queries-buttons sl-button').first().click();

	await openProperty(page, 'email');
	await description.fill(originalDescription);
	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.
	await propertyPanel.locator('.property-panel__save').click();
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
	await page.waitForTimeout(1000);
	expect(previewRequests).toBe(1);

	// Structural changes are also previewed only when the user applies them.
	await page.click('.schema-edit__add-property');
	await page.locator('.property-form__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'temporary_preview_property');
	await selectPropertyType(page, 'string');
	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.
	await propertyPanel.locator('.property-panel__save').click();
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('loading');
	await page.waitForTimeout(1000);
	expect(previewRequests).toBe(1);

	const structuralPreviewResponse = page.waitForResponse(
		(response) => response.url().includes('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await structuralPreviewResponse;
	await expect(dialog).toHaveAttribute('label', 'Review changes');
	await expect(dialog.locator('.schema-edit__apply-alter-button')).toHaveAttribute('variant', 'danger');
});

test(`Add schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
	await page.click('.schema-edit__add-property');

	const panel = page.locator('.property-panel');
	await expect(panel.locator('.property-panel__title')).toHaveText('New property');
	const nameInput = panel.locator('.property-form__name-input');
	await expect(nameInput).toBeFocused();
	await expect(panel.locator('.property-form__change-name')).toHaveCount(0);
	await expect(panel.locator('.property-panel__remove')).toHaveCount(0);
	await expect(panel.locator('.property-panel__cancel')).toHaveText('Cancel');
	await expect(panel.locator('.property-panel__save')).toHaveText('Confirm');

	await nameInput.evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await page.keyboard.press('Tab');
	await expect(panel.locator('.property-type-selector__structure-trigger')).toBeFocused();
	await expect(panel.locator('.property-type-selector__structure-dropdown')).toHaveJSProperty('open', false);
	await page.keyboard.press('ArrowDown');
	await expect(panel.locator('.property-type-selector__structure-dropdown')).toHaveJSProperty('open', true);
	await expect(panel.locator('[data-structure-option="one"]')).toBeFocused();
	await page.keyboard.press('Enter');
	await expect(panel.locator('.property-type-selector__trigger')).toBeFocused();
	await expect(panel.locator('.property-type-selector__dropdown')).toHaveJSProperty('open', false);
	await page.keyboard.press('ArrowDown');
	await expect(panel.locator('.property-type-selector__dropdown')).toHaveJSProperty('open', true);
	await expect(panel.locator('[data-type-option="string"]')).toBeFocused();
	await page.keyboard.press('ArrowDown');
	await expect(panel.locator('[data-type-option="int"]')).toBeFocused();
	await page.keyboard.press('ArrowUp');
	await expect(panel.locator('[data-type-option="string"]')).toBeFocused();
	await page.keyboard.press('Enter');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await panel.locator('.property-panel__save').click();
	await logValidationErrors(page, ['.property-form__control-error']);
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('loading');
	await expect(applyButton).not.toHaveAttribute('disabled');

	const previewResponse = page.waitForResponse(
		(response) => response.url().includes('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await previewResponse;
	await expect(page.locator('.schema-edit__queries')).toHaveAttribute('label', 'Review changes');
	await expect(page.locator('.schema-edit__apply-alter-button')).toHaveAttribute('variant', 'danger');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	let cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeAttached();
	await expect(cell.locator('xpath=../following-sibling::*[1]')).toContainText('string');

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeAttached();
	await expect(cell.locator('xpath=../following-sibling::*[1]')).toContainText('string');
});

test(`Clear the primary source when changing a new property to an array`, async ({ page }) => {
	const sourceID = '9zQ4Tn7B3mS6';
	await page.route('**/v1/connections', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.connections.push({
			id: sourceID,
			name: 'Schema test source',
			connector: 'dummy',
			connectorType: 'SDK',
			role: 'Source',
			storage: '',
			compression: '',
			strategy: null,
			sendingMode: null,
			hasSettings: false,
			health: 'Healthy',
			linkedConnections: null,
		});
		await route.fulfill({ response, json: body });
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		if (route.request().method() === 'PUT') {
			await route.fulfill({ json: {} });
			return;
		}
		await route.continue();
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('array_without_primary_source');
	await selectPropertyType(page, 'string');
	const primarySource = propertyPanel.locator('.property-form__primary-source');
	await primarySource.click();
	await primarySource.locator(`sl-option[value="${sourceID}"]`).click();
	await expect(primarySource).toHaveJSProperty('open', false);
	await primarySource.focus();
	await page.keyboard.press('Tab');
	await expect(propertyPanel.locator('.property-panel__cancel')).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(propertyPanel.locator('.property-panel__save')).toBeFocused();

	await propertyPanel.locator('.property-type-selector__structure-trigger').click();
	await propertyPanel.locator('[data-structure-option="array"]').click();
	await expect(primarySource).toHaveCount(0);
	await expect(propertyPanel.locator('.property-type-selector__trigger')).toBeFocused();
	await expect(propertyPanel.locator('.property-type-selector__dropdown')).toHaveJSProperty('open', false);
	await propertyPanel.locator('.property-panel__save').click();

	const alterRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema') && request.method() === 'PUT',
	);
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).not.toHaveAttribute('disabled');
	const previewResponsePromise = page.waitForResponse(
		(response) => response.url().endsWith('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await previewResponsePromise;
	await expect(page.locator('.schema-edit__queries')).toBeVisible();
	await page.locator('.schema-edit__apply-alter-button').click();
	const alterRequest = await alterRequestPromise;
	expect(alterRequest.postDataJSON().primarySources).not.toHaveProperty('array_without_primary_source');
	await expect(page.locator('.schema-edit')).toHaveCount(0);
});

test(`Only offer user-capable source connections as primary sources`, async ({ page }) => {
	const eventSourceID = '7B3mN9qK2xA';
	const userSourceID = '9zQ4Tn7B3mS6';
	await page.route('**/v1/connections', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.connections.push(
			{
				id: eventSourceID,
				name: 'Event-only source',
				connector: 'kafka',
				connectorType: 'MessageBroker',
				role: 'Source',
				storage: '',
				compression: '',
				strategy: null,
				sendingMode: null,
				hasSettings: true,
				health: 'Healthy',
				linkedConnections: null,
			},
			{
				id: userSourceID,
				name: 'User-capable source',
				connector: 'dummy',
				connectorType: 'SDK',
				role: 'Source',
				storage: '',
				compression: '',
				strategy: null,
				sendingMode: null,
				hasSettings: false,
				health: 'Healthy',
				linkedConnections: null,
			},
		);
		await route.fulfill({ response, json: body });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('property_with_primary_source');
	await selectPropertyType(page, 'string');
	const primarySource = propertyPanel.locator('.property-form__primary-source');
	await primarySource.click();
	await expect(primarySource.locator(`sl-option[value="${userSourceID}"]`)).toHaveCount(1);
	await expect(primarySource.locator(`sl-option[value="${eventSourceID}"]`)).toHaveCount(0);
});

test(`Validate string length constraints before adding a property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('string_with_length_constraints');
	await selectPropertyType(page, 'string');

	const lengthConstraints = propertyPanel.locator('.property-form__constraints--length sl-input');
	const maxCharacters = lengthConstraints.nth(0).locator('input');
	const maxBytes = lengthConstraints.nth(1).locator('input');
	const addedProperty = page.locator('.schema-edit .grid__row[data-id="string_with_length_constraints"]');

	await maxCharacters.fill('0');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(addedProperty).toHaveCount(0);

	await maxCharacters.fill('1');
	await maxBytes.fill('1.5');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(addedProperty).toHaveCount(0);

	await maxBytes.fill('4294967296');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(addedProperty).toHaveCount(0);

	await maxCharacters.fill('');
	await maxBytes.fill('');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(addedProperty).toBeVisible();
});

test(`Validate decimal constraints as they are edited`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	const saveButton = propertyPanel.locator('.property-panel__save');
	const decimalError = propertyPanel.locator(
		'.property-form__constraints--decimal > [data-error-on="decimal-constraints"]',
	);
	const decimalDescription = propertyPanel.locator('.property-form__decimal-description');
	await propertyPanel.locator('.property-form__name-input input').fill('decimal_constraints');
	await selectPropertyType(page, 'decimal');

	const precision = propertyPanel.locator('.property-form__precision input');
	const scale = propertyPanel.locator('.property-form__scale input');

	await precision.fill('');
	await expect(precision).toHaveValue('');
	await expect(decimalError).toContainText('Precision cannot be empty');
	await expect(decimalDescription).toHaveCount(0);
	await expect(saveButton).toHaveAttribute('disabled');

	await precision.fill('10');
	await scale.fill('');
	await expect(scale).toHaveValue('');
	await expect(decimalError).toContainText('Scale cannot be empty');
	await expect(decimalDescription).toHaveCount(0);
	await expect(saveButton).toHaveAttribute('disabled');

	await scale.fill('4');
	await expect(decimalError).toHaveCount(0);
	await expect(decimalDescription).toHaveText('10 digits total, with 4 after the decimal point.');
	await expect(saveButton).not.toHaveAttribute('disabled');
	await saveButton.click();
	const addedProperty = page.locator('.schema-edit .grid__row[data-id="decimal_constraints"]');
	await expect(addedProperty.locator('.schema-edit__property-technical-type')).toHaveText('decimal(10,4)');
});

test(`Edit schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);

	await openProperty(page, 'foo');
	await expect(page.locator('.property-type-selector__structure-trigger')).toHaveJSProperty('caret', false);
	await expect(page.locator('.property-type-selector__trigger')).toHaveJSProperty('caret', false);
	const changeNameButton = page.locator('.property-panel .property-form__change-name');
	const nameInput = page.locator('.property-panel .property-form__name-input');
	await expect(nameInput).toHaveAttribute('readonly', '');
	await nameInput.evaluate((element: any) => {
		const input = element.shadowRoot.querySelector('input');
		input.scrollLeft = input.scrollWidth;
		input.focus();
	});
	await expect(nameInput).not.toBeFocused();
	await expect
		.poll(() => nameInput.evaluate((element: any) => element.shadowRoot.querySelector('input').scrollLeft))
		.toBe(0);
	await changeNameButton.click();
	await expect(nameInput).not.toHaveAttribute('readonly', '');
	await expect(nameInput).toBeFocused();
	await expect
		.poll(() => nameInput.evaluate((element: any) => element.shadowRoot.querySelector('input').selectionStart))
		.toBe(0);
	await expect(changeNameButton).toHaveCount(0);
	const nativeNameInput = nameInput.locator('input');
	await nativeNameInput.fill('temporary_name');
	await nativeNameInput.fill('foo');
	await nativeNameInput.blur();
	await expect(nameInput).toHaveAttribute('readonly', '');
	await expect(changeNameButton).toBeVisible();
	await changeNameButton.click();
	await nativeNameInput.fill('_foo');
	await expect(page.locator('.property-form__control--name .property-form__control-error')).toContainText(
		'Profile schema property names cannot start with an underscore',
	);
	await expect(page.locator('.property-panel__save')).toHaveAttribute('disabled');

	await nameInput.evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'bar');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel__save');
	await logValidationErrors(page, ['.property-form__control-error']);

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Apply changes');
	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	let fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	let barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).not.toBeAttached();
	await expect(barCell).toBeAttached();

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).not.toBeAttached();
	await expect(barCell).toBeAttached();
});

test(`Restore the original property name without leaving pending changes`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'original_property_name',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'original_property_name');
	const firstVisiblePropertyKey = await page
		.locator('.schema-edit .grid__row--clickable:visible')
		.first()
		.getAttribute('data-id');
	expect(firstVisiblePropertyKey).not.toBeNull();

	const propertyPanel = page.locator('.property-panel');
	const nameInput = propertyPanel.locator('.property-form__name-input input');
	const changeNameButton = propertyPanel.locator('.property-form__change-name');

	await changeNameButton.click();
	await nameInput.fill('temporary_property_name');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');
	await page.locator('.schema-edit__filter-button').click();
	const showChanged = page.locator('.schema-edit__show-changed');
	await showChanged.click();
	await page.locator('.schema-edit__filter-button').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="original_property_name"]')).toHaveClass(
		/grid__row--selected/,
	);

	await changeNameButton.click();
	await nameInput.fill('original_property_name');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit__change-count')).toContainText('No pending changes');
	await expect(page.locator('.schema-edit__header-apply-button')).toHaveAttribute('disabled');
	await expect(page.locator('.schema-edit .grid__row')).toHaveCount(0);
	await expect(page.locator('.property-panel--empty')).toBeEmpty();

	await page.locator('.schema-edit__filter-button').click();
	await showChanged.click();
	await expect(page.locator(`.schema-edit .grid__row[data-id="${firstVisiblePropertyKey}"]`)).toHaveClass(
		/grid__row--selected/,
	);
});

test(`Remove a renamed property without sending its stale RePath`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'property_to_rename_and_remove',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'property_to_rename_and_remove');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('temporary_property_name');
	await propertyPanel.locator('.property-panel__save').click();
	await propertyPanel.locator('.property-panel__remove').click();
	await page.click('.schema-edit__confirm-remove-property');

	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await page.locator('.schema-edit__header-apply-button').click();
	const previewRequest = await previewRequestPromise;
	expect(previewRequest.postDataJSON().rePaths).toEqual({});
});

test(`Remove a replacement property without sending its stale RePath`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		if (route.request().method() === 'PUT') {
			await route.fulfill({ json: {} });
			return;
		}
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'property_to_replace_and_remove',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await removeProperty(page, 'property_to_replace_and_remove');

	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__parent').evaluate((select: any) => {
		select.value = '__root__';
		select.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	});
	await propertyPanel.locator('.property-form__name-input input').fill('property_to_replace_and_remove');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();

	await removeProperty(page, 'property_to_replace_and_remove');

	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).not.toHaveAttribute('disabled');
	await applyButton.click();
	await expect(page.locator('.schema-edit__queries')).toBeVisible();
	const alterRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema') && request.method() === 'PUT',
	);
	await page.locator('.schema-edit__apply-alter-button').click();
	const alterRequest = await alterRequestPromise;
	expect(alterRequest.postDataJSON().rePaths).toEqual({});
});

test(`Do not show modified field indicators on a replacement property`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'property_to_replace',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: 'Original description',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await removeProperty(page, 'property_to_replace');

	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('property_to_replace');
	await selectPropertyType(page, 'boolean');
	await propertyPanel.locator('sl-textarea textarea[name="description"]').fill('Replacement description');
	await propertyPanel.locator('.property-panel__save').click();

	await expect(propertyPanel.locator('.schema-edit__property-status')).toHaveText('Added');
	await expect(propertyPanel.locator('.property-form__modified-dot')).toHaveCount(0);
});

test(`Rename an existing property to a deleted property's name`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push(
			{
				name: 'property_to_rename',
				prefilled: '',
				role: 'Both',
				type: { kind: 'string' },
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
			{
				name: 'deleted_property_name',
				prefilled: '',
				role: 'Both',
				type: { kind: 'string' },
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
		);
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await removeProperty(page, 'deleted_property_name');
	await openProperty(page, 'property_to_rename');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('deleted_property_name');
	await propertyPanel.locator('.property-panel__save').click();

	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await page.click('.schema-edit__header-apply-button');
	const previewRequest = await previewRequestPromise;
	expect(previewRequest.postDataJSON().rePaths).toEqual({ deleted_property_name: 'property_to_rename' });
});

test(`Check that RePaths are sent correctly`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);

	await openProperty(page, 'bar');
	await page.locator('.property-panel .property-form__change-name').click();

	await page.locator('.property-panel .property-form__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel__save');
	await logValidationErrors(page, ['.property-form__control-error']);

	await page.click('.schema-edit__add-property');

	await page.locator('.property-panel .property-form__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'bar');

	await selectPropertyType(page, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-panel__save');
	await logValidationErrors(page, ['.property-form__control-error']);

	await page.waitForTimeout(2000); // Add a timeout to ensure that editable schema in the React state is synced with the newly added property.

	let isRequestOK = false;
	page.on('request', async (request) => {
		if (request.url().includes('/profiles/schema') && request.method() === 'PUT') {
			const body = request.postData();
			const parsed = JSON.parse(body);
			isRequestOK = JSON.stringify(parsed.rePaths) === JSON.stringify({ foo: 'bar', bar: null });
		}
	});

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Apply changes');
	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await page.waitForTimeout(5000); // Add a timeout to ensure that the saving was completed.
	expect(isRequestOK).toBe(true);

	let fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	let barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).toBeAttached();
	await expect(barCell).toBeAttached();

	await page.reload();

	fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).toBeAttached();
	await expect(barCell).toBeAttached();
});

test(`Reuse a property name more than once before applying schema changes`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'reused_property_name',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	const renameProperty = async (key: string, name: string) => {
		await openProperty(page, key);
		const propertyPanel = page.locator('.property-panel');
		await propertyPanel.locator('.property-form__change-name').click();
		await propertyPanel.locator('.property-form__name-input input').fill(name);
		await propertyPanel.locator('.property-panel__save').click();
	};
	const addStringProperty = async () => {
		await page.click('.schema-edit__add-property');
		const propertyPanel = page.locator('.property-panel');
		await propertyPanel.locator('.property-form__name-input input').fill('reused_property_name');
		await selectPropertyType(page, 'string');
		await propertyPanel.locator('.property-panel__save').click();
	};

	await renameProperty('reused_property_name', 'first_renamed_property');
	await addStringProperty();
	await renameProperty('reused_property_name-2', 'second_renamed_property');
	await addStringProperty();

	const thirdProperty = page.locator('.schema-edit .grid__row[data-id="reused_property_name-3"]');
	await expect(thirdProperty).toBeVisible();
	await expect(thirdProperty.locator('.grid__cell').first()).toContainText('reused_property_name');
});

test(`Remove a replacement property's RePath when renaming it`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'replacement_name',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	const propertyPanel = page.locator('.property-panel');
	await openProperty(page, 'replacement_name');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('renamed_original');
	await propertyPanel.locator('.property-panel__save').click();

	await page.click('.schema-edit__add-property');
	await propertyPanel.locator('.property-form__name-input input').fill('replacement_name');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();

	await openProperty(page, 'replacement_name-2');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('renamed_replacement');
	await propertyPanel.locator('.property-panel__save').click();

	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await page.click('.schema-edit__header-apply-button');
	const previewRequest = await previewRequestPromise;
	expect(previewRequest.postDataJSON().rePaths).toEqual({ renamed_original: 'replacement_name' });
});

test(`Allow matching property names under different object parents`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		const createProperty = (name: string, type: Property['type']): Property => ({
			name,
			prefilled: '',
			role: 'Both',
			type,
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		schema.properties.push(
			createProperty('duplicate_scope', {
				kind: 'object',
				properties: [
					createProperty('billing', {
						kind: 'object',
						properties: [
							createProperty('matching_add_name', { kind: 'string' }),
							createProperty('matching_rename_name', { kind: 'string' }),
						],
					}),
					createProperty('shipping', {
						kind: 'object',
						properties: [createProperty('shipping_child', { kind: 'string' })],
					}),
				],
			}),
		);
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__expand-all-button');

	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__parent').evaluate((select: any) => {
		select.value = 'duplicate_scope.billing';
		select.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	});
	await propertyPanel.locator('.property-form__name-input input').fill('matching_add_name');
	const nameError = propertyPanel.locator('.property-form__control--name .property-form__control-error');
	await expect(nameError).toHaveText(
		'A property named “matching_add_name” already exists in duplicate_scope › billing.',
	);
	await expect(propertyPanel.locator('.property-panel__save')).toHaveAttribute('disabled');
	await propertyPanel.locator('.property-form__parent').evaluate((select: any) => {
		select.value = 'duplicate_scope.shipping';
		select.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	});
	await expect(nameError).not.toBeAttached();
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.grid__row[data-id="duplicate_scope.shipping.matching_add_name"]')).toBeVisible();

	await openProperty(page, 'duplicate_scope.shipping.shipping_child');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('matching_rename_name');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(
		page.locator('.schema-edit .grid__row[data-id="duplicate_scope.shipping.shipping_child"]'),
	).toContainText('matching_rename_name');
});

test(`Support hasOwnProperty as a profile schema property name`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		const createProperty = (name: string, type: Property['type']): Property => ({
			name,
			prefilled: '',
			role: 'Both',
			type,
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		schema.properties.push(
			createProperty('hasOwnProperty', { kind: 'string' }),
			createProperty('property_container', {
				kind: 'object',
				properties: [createProperty('existing_child', { kind: 'string' })],
			}),
		);
		await route.fulfill({ response, json: schema });
	});
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await expect(page.locator('.schema-edit .grid__row[data-id="hasOwnProperty"]')).toBeVisible();

	await openProperty(page, 'property_container');
	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('hasOwnProperty');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="property_container.hasOwnProperty"]')).toBeVisible();

	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).not.toHaveAttribute('disabled');
	const previewResponsePromise = page.waitForResponse(
		(response) => response.url().endsWith('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await applyButton.click();
	await previewResponsePromise;
	await expect(page.locator('.schema-edit__queries')).toBeVisible();
});

test(`Ignore inherited primary sources for prototype property names`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'toString',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'toString');

	const propertyPanel = page.locator('.property-panel');
	const description = propertyPanel.locator('sl-textarea textarea[name="description"]');
	await description.fill('Updated description');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');

	await description.fill('');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit__change-count')).toContainText('No pending changes');
});

test(`Add schema object property with sub-property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);
	await page.click('.schema-edit__add-property');

	await page.locator('.property-panel .property-form__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_obj');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-type-selector__structure-trigger').click();
	await propertyPanel.locator('[data-structure-option="object"]').click();
	await expect(propertyPanel.locator('.property-type-selector__structure-trigger')).toContainText('object');
	await expect(propertyPanel.locator('.property-type-selector__dropdown')).toHaveCount(0);

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-panel__save');
	await logValidationErrors(page, ['.property-form__control-error']);

	const objectRow = page.locator('.grid__row[data-id="test_obj"]');
	await expect(objectRow).toBeVisible();
	await page.click('.schema-edit__add-property');
	await expect(propertyPanel.locator('.property-type-selector__structure-trigger')).toContainText('one value');
	await expect(propertyPanel.locator('.property-type-selector__dropdown')).toHaveCount(1);
	await page.locator('.property-panel .property-form__parent').evaluate((select: any) => {
		select.value = 'test_obj';
		select.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	});

	await page.locator('.property-panel .property-form__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_sub_prop_1');

	await selectPropertyType(page, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-panel__save');
	await logValidationErrors(page, ['.property-form__control-error']);

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Apply changes');
	await page.click('.schema-edit__header-apply-button');

	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_obj',
		}),
	).toBeAttached();

	await page.waitForTimeout(8000); // Ensures that the Admin console has had enough time to poll the server to know if the update is completed (polling happens every 3 seconds) and to refetch the schema.
	await expandAllObjects(page);
	await expect(
		page.locator('.grid__row--children > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_sub_prop_1',
		}),
	).toBeAttached();

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_obj',
		}),
	).toBeAttached();
	await expandAllObjects(page);
	await expect(
		page.locator('.grid__row--children > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_sub_prop_1',
		}),
	).toBeAttached();
});

test(`Remove nested properties when changing a new object to another type`, async ({ page }) => {
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		await route.fulfill({ json: { queries: [] } });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('new_object');
	await propertyPanel.locator('.property-type-selector__structure-trigger').click();
	await propertyPanel.locator('[data-structure-option="object"]').click();
	await propertyPanel.locator('.property-panel__save').click();

	await page.click('.schema-edit__add-property');
	await propertyPanel.locator('.property-form__name-input input').fill('nested_property');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="new_object.nested_property"]')).toBeVisible();

	await openProperty(page, 'new_object');
	await propertyPanel.locator('.property-type-selector__structure-trigger').click();
	await propertyPanel.locator('[data-structure-option="one"]').click();
	await expect(propertyPanel.locator('.property-type-selector__trigger')).toBeFocused();
	await expect(propertyPanel.locator('.property-type-selector__dropdown')).toHaveJSProperty('open', false);
	await propertyPanel.locator('.property-type-selector__trigger').click();
	await propertyPanel.locator('[data-type-option="string"]').click();
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="new_object.nested_property"]')).toHaveCount(0);
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).not.toHaveAttribute('disabled');
	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await applyButton.click();
	const previewRequest = await previewRequestPromise;
	const previewSchema = previewRequest.postDataJSON().schema as ObjectType;
	const property = previewSchema.properties.find((candidate) => candidate.name === 'new_object');
	expect(property?.type).toEqual({ kind: 'string' });
	await expect(page.locator('.schema-edit__queries')).toBeVisible();
});

test(`Reject descendant changes while renaming an object property`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'object_to_rename',
			prefilled: '',
			role: 'Both',
			type: {
				kind: 'object',
				properties: [
					{
						name: 'child',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string' },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
				],
			},
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});
	let previewRequests = 0;
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		previewRequests++;
		await route.abort();
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'object_to_rename');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__change-name').click();
	await propertyPanel.locator('.property-form__name-input input').fill('renamed_object');
	await propertyPanel.locator('.property-panel__save').click();

	await expandAllObjects(page);
	await openProperty(page, 'object_to_rename.child');
	await propertyPanel.locator('textarea[name="description"]').fill('Changed description');
	await propertyPanel.locator('.property-panel__save').click();
	await page.locator('.schema-edit__header-apply-button').click();

	await expect(page.locator('.toast')).toContainText(
		'Object property "object_to_rename" cannot be renamed while its nested properties are being changed',
	);
	expect(previewRequests).toBe(0);
});

test(`Reject a new object property without sub-properties`, async ({ page }) => {
	let previewRequests = 0;
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		previewRequests++;
		await route.abort();
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__add-property');

	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input').evaluate((input: any) => {
		input.value = 'empty_object';
		input.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	});
	await propertyPanel.locator('.property-type-selector__structure-trigger').click();
	await propertyPanel.locator('[data-structure-option="object"]').click();
	await propertyPanel.locator('.property-panel__save').click();
	await expect(page.locator('.schema-edit .grid__row[data-id="empty_object"]')).toBeVisible();

	await page.locator('.schema-edit__header-apply-button').click();
	await expect(page.locator('.toast')).toContainText(
		'Object property "empty_object" must contain at least one property',
	);
	expect(previewRequests).toBe(0);
});

test(`Reject an existing object property after removing all its sub-properties`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push(
			{
				name: 'object_to_empty',
				prefilled: '',
				role: 'Both',
				type: {
					kind: 'object',
					properties: [
						{
							name: 'child',
							prefilled: '',
							role: 'Both',
							type: { kind: 'string' },
							createRequired: false,
							updateRequired: false,
							readOptional: true,
							nullable: false,
							description: '',
						},
					],
				},
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
			{
				name: 'object_to_empty_sibling',
				prefilled: '',
				role: 'Both',
				type: { kind: 'string' },
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				description: '',
			},
		);
		await route.fulfill({ response, json: schema });
	});
	let previewRequests = 0;
	await page.route('**/v1/profiles/schema/preview', async (route) => {
		previewRequests++;
		await route.abort();
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__expand-all-button');
	await removeProperty(page, 'object_to_empty.child');
	await expect(page.locator('.schema-edit .grid__row[data-id="object_to_empty.child"]')).toHaveCount(0);

	await page.locator('.schema-edit__header-apply-button').click();
	await expect(page.locator('.toast')).toContainText(
		'Object property "object_to_empty" must contain at least one property',
	);
	expect(previewRequests).toBe(0);
});

test(`Count an object removal once after changing its children`, async ({ page }) => {
	await page.route('**/v1/profiles/schema', async (route) => {
		const response = await route.fetch();
		const schema = (await response.json()) as ObjectType;
		schema.properties.push({
			name: 'object_with_replaced_child',
			prefilled: '',
			role: 'Both',
			type: {
				kind: 'object',
				properties: [
					{
						name: 'child',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string' },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
					{
						name: 'removed_child',
						prefilled: '',
						role: 'Both',
						type: { kind: 'string' },
						createRequired: false,
						updateRequired: false,
						readOptional: true,
						nullable: false,
						description: '',
					},
				],
			},
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		});
		await route.fulfill({ response, json: schema });
	});

	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await page.click('.schema-edit__expand-all-button');
	await removeProperty(page, 'object_with_replaced_child.child');

	await page.click('.schema-edit__add-property');
	const propertyPanel = page.locator('.property-panel');
	await propertyPanel.locator('.property-form__name-input input').fill('child');
	await selectPropertyType(page, 'string');
	await propertyPanel.locator('.property-panel__save').click();

	await removeProperty(page, 'object_with_replaced_child.removed_child');
	await removeProperty(page, 'object_with_replaced_child');
	await expect(page.locator('.schema-edit__change-count')).toContainText('1 pending change');

	const previewRequestPromise = page.waitForRequest(
		(request) => request.url().endsWith('/profiles/schema/preview') && request.method() === 'PUT',
	);
	await page.click('.schema-edit__header-apply-button');
	const previewRequest = await previewRequestPromise;
	expect(previewRequest.postDataJSON().rePaths).toEqual({});
});

test(`Clear the property selection after deleting a filtered property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);

	await page.locator('.schema-edit__search-button').click();
	const searchInput = page.locator('.schema-edit__search >> input');
	await searchInput.fill('foo string');
	await removeProperty(page, 'foo');

	await expect(page.locator('.property-panel--empty')).toBeVisible();
	await expect(page.locator('.schema-edit .grid__row--selected')).toHaveCount(0);
	await expect(searchInput).toHaveValue('foo string');
});

test(`Do not restore focus to the delete action after closing its dialog`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await editSchema(page);
	await openProperty(page, 'foo');

	const propertyPanel = page.locator('.property-panel');
	const removeButton = propertyPanel.locator('.property-panel__remove');
	const deleteTooltip = propertyPanel
		.locator('.schema-edit__toolbar-tooltip')
		.filter({ has: page.locator('.property-panel__remove') });
	const grid = page.locator('.schema-edit .grid');
	const removeDialog = page
		.locator('.alert-dialog')
		.filter({ has: page.locator('.schema-edit__confirm-remove-property') });
	await expect(deleteTooltip).toHaveCount(1);
	await removeButton.click();
	await expect(removeDialog).toBeVisible();
	await deleteTooltip.evaluate((tooltip) => {
		tooltip.dataset.focusReturnCount = '0';
		tooltip.addEventListener('focusin', () => {
			tooltip.dataset.focusReturnCount = String(Number(tooltip.dataset.focusReturnCount) + 1);
		});
	});
	await removeDialog.evaluate((dialog) => {
		dialog.dataset.afterHideSettled = 'false';
		dialog.addEventListener(
			'sl-after-hide',
			() => {
				// Shoelace queues focus restoration immediately before emitting sl-after-hide.
				setTimeout(() => {
					dialog.dataset.afterHideSettled = 'true';
				});
			},
			{ once: true },
		);
	});
	await removeDialog.locator('sl-button').filter({ hasText: 'Cancel' }).click();
	await expect(removeDialog).toHaveAttribute('data-after-hide-settled', 'true');
	await expect(grid).toBeFocused();
	await expect(deleteTooltip).toHaveAttribute('data-focus-return-count', '0');
	await expect(deleteTooltip).not.toHaveAttribute('open');

	await removeButton.click();
	await expect(removeDialog).toBeVisible();
	await deleteTooltip.evaluate((tooltip) => {
		tooltip.dataset.focusReturnCount = '0';
	});
	await removeDialog.evaluate((dialog) => {
		dialog.dataset.afterHideSettled = 'false';
		dialog.addEventListener(
			'sl-after-hide',
			() => {
				setTimeout(() => {
					dialog.dataset.afterHideSettled = 'true';
				});
			},
			{ once: true },
		);
	});
	await removeDialog.locator('.schema-edit__confirm-remove-property').click();
	await expect(removeDialog).toHaveAttribute('data-after-hide-settled', 'true');
	await expect(grid).toBeFocused();
	await expect(deleteTooltip).toHaveAttribute('data-focus-return-count', '0');
	await expect(deleteTooltip).not.toHaveAttribute('open');
});

test(`Remove schema properties`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);

	await removeProperty(page, 'foo');
	await removeProperty(page, 'bar');
	await removeProperty(page, 'test_obj');

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Apply changes');
	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^test_obj$/ }),
	).not.toBeAttached();

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^test_obj$/ }),
	).not.toBeAttached();
});

test(`Check that the property name is correctly validated`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await editSchema(page);
	await page.click('.schema-edit__add-property');

	let error = page.locator('.property-form__control--name .property-form__control-error');
	let saveButton = page.locator('.property-panel__save');

	// Name cannot be empty.
	await page.locator('sl-input >> input[name="name"]').fill('test');
	await page.locator('sl-input >> input[name="name"]').fill('');
	await expect(error).toBeAttached();
	await expect(error).toContainText('Name cannot be empty');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name cannot contain spaces.
	await page.locator('sl-input >> input[name="name"]').fill('my property');
	await expect(error).toBeAttached();
	await expect(error).toContainText('Name cannot contain spaces');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name cannot start with a number.
	await page.locator('sl-input >> input[name="name"]').fill('3foo');
	await expect(error).toBeAttached();
	await expect(error).toContainText('Name cannot start with a number');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name must start with an ASCII alphabet character or an
	// underscore.
	await page.locator('sl-input >> input[name="name"]').fill('$foo');
	await expect(error).toBeAttached();
	await expect(error).toContainText('Name must start with an ASCII alphabet character or an underscore');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name must contain only ASCII alphabet characters, digits and
	// underscores.
	await page.locator('sl-input >> input[name="name"]').fill('foo_3bar');
	await expect(error).not.toBeAttached();
	await expect(saveButton).not.toHaveAttribute('disabled');
	await page.locator('sl-input >> input[name="name"]').fill('foo$bar');
	await expect(error).toBeAttached();
	await expect(error).toContainText('Name must contain only ASCII alphabet characters, digits and underscores');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name must be unique within the selected parent.
	await page.locator('sl-input >> input[name="name"]').fill('email');
	await expect(error).toContainText('A property named “email” already exists in Profile (top level).');
	await expect(saveButton).toHaveAttribute('disabled');
});
