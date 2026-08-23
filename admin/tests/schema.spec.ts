import { test, expect } from '@playwright/test';
import { login, logout, adminURL, logValidationErrors } from './utils';
import { ObjectType } from '../src/lib/api/types/types';

const selectPropertyType = async (page, option: string) => {
	const panel = page.locator('.property-panel');
	await panel.locator('.property-type-selector__trigger').click();
	await panel.locator(`[data-type-option="${option}"]`).click();
};

const openProperty = async (page, property: string) => {
	const isEditing = new URL(page.url()).pathname.endsWith('/schema/edit');
	const schema = page.locator(isEditing ? '.schema-edit' : '.schema-grid');
	await schema.locator(`.grid__row[data-id="${property}"]`).click();
};

const expandAllObjects = async (page) => {
	await page.click('.schema-grid__expand-all-button');
};

const removeProperty = async (page, property: string) => {
	await openProperty(page, property);
	await page.locator('.property-panel__remove').click();
	await page.click('.schema-edit__confirm-remove-property');
};

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
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
	await expect(page.locator('.grid__row[data-id="birth_date"]')).toHaveCount(0);
	await searchInput.fill('');
	await searchInput.blur();
	await expect(searchInput).toHaveCount(0);
	await expect(searchButton).toBeVisible();

	await page.click('.schema-grid__alter-button');
	const editSearchButton = page.locator('.schema-edit__search-button');
	await editSearchButton.click();
	const editSearchInput = page.locator('.schema-edit__search >> input');
	await expect(editSearchInput).toBeFocused();
	await editSearchInput.fill('string');
	await expect(page.locator('.schema-edit .grid__row[data-id="email"]')).toBeVisible();
	await expect(page.locator('.schema-edit .grid__row[data-id="birth_date"]')).toHaveCount(0);
	await editSearchInput.fill('');
	await editSearchInput.blur();
	await expect(editSearchInput).toHaveCount(0);

	await page.locator('.schema-edit__filter-button').click();
	const showChanged = page.locator('.schema-edit__show-changed');
	await showChanged.click();
	await expect(page.locator('.schema-edit__filter-dot')).toHaveClass(/schema-edit__filter-dot--active/);
	await expect(page.locator('.schema-edit .grid__row')).toHaveCount(0);
	await showChanged.click();
});

test(`Discard unsaved property changes when selecting another property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);
	await page.click('.schema-grid__alter-button');
	const applyButton = page.locator('.schema-edit__header-apply-button');

	await openProperty(page, 'email');
	const propertyPanel = page.locator('.property-panel');
	await expect(propertyPanel.locator('.property-panel__header sl-icon-button')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-panel__cancel')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-dialog__save')).toHaveCount(0);
	await expect(propertyPanel.locator('.property-panel__remove')).toBeVisible();
	const description = page.locator('.property-panel sl-textarea >> textarea[name="description"]');
	await description.fill('Unsaved description');
	await expect(propertyPanel.locator('.property-panel__cancel')).toHaveText('Cancel');
	await expect(propertyPanel.locator('.property-dialog__save')).toHaveText('Confirm');
	await expect(propertyPanel.locator('.property-panel__remove')).toHaveCount(0);
	await page.waitForTimeout(100);
	await openProperty(page, 'phone_numbers');

	const dialog = page.locator('.alert-dialog', { hasText: 'Discard unsaved changes?' });
	await expect(dialog).toBeVisible();
	await dialog.getByText('Discard changes', { exact: true }).click();
	await expect(page.locator('.schema-edit .grid__row[data-id="phone_numbers"]')).toHaveClass(/grid__row--selected/);
	await expect(applyButton).toHaveAttribute('disabled');
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
	await expect(
		panel.locator('.property-details-panel__detail').first().locator('.property-details-panel__value'),
	).toHaveText('email');
	await expect(panel).toContainText('string, max 300 chars');
	await expect(page.locator('.grid__row[data-id="email"]')).toHaveClass(/grid__row--selected/);
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
	await expect(panel).toContainText('Text');
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

	await page.click('.schema-grid__alter-button');
	await expect(page.locator('.property-details-panel')).toHaveCount(0);
	await expect(page.locator('.property-panel .property-panel__title')).toHaveText('Property');
	await expect(page.locator('.property-panel .property-dialog__name-input')).toHaveJSProperty('value', 'country');
	const selectedRow = page.locator('.schema-edit .grid__row[data-id="address.country"]');
	await expect(selectedRow).toBeVisible();
	await expect(selectedRow).toHaveClass(/grid__row--selected/);
});

test(`Add schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	await page.click('.schema-edit__add-property');

	const panel = page.locator('.property-panel');
	await expect(panel.locator('.property-panel__title')).toHaveText('New property');
	const nameInput = panel.locator('.property-dialog__name-input');
	await expect(nameInput).toBeFocused();
	await expect(panel.locator('.property-form__change-name')).toHaveCount(0);
	await expect(panel.locator('.property-panel__remove')).toHaveCount(0);
	await expect(panel.locator('.property-panel__cancel')).toHaveText('Cancel');
	await expect(panel.locator('.property-dialog__save')).toHaveText('Confirm');

	await nameInput.evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await selectPropertyType(page, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await panel.locator('.property-dialog__save').click();
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.schema-edit__header-apply-button');
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

test(`Edit schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');

	await openProperty(page, 'foo');
	const changeNameButton = page.locator('.property-panel .property-form__change-name');
	const nameInput = page.locator('.property-panel .property-dialog__name-input');
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

	await nameInput.evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'bar');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

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

test(`Check that RePaths are sent correctly`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');

	await openProperty(page, 'bar');
	await page.locator('.property-panel .property-form__change-name').click();

	await page.locator('.property-panel .property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.schema-edit__add-property');

	await page.locator('.property-panel .property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'bar');

	await selectPropertyType(page, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.waitForTimeout(2000); // Add a timeout to ensure that editable schema in the React state is synced with the newly added property.

	let isRequestOK = false;
	page.on('request', async (request) => {
		if (request.url().includes('/profiles/schema') && request.method() === 'PUT') {
			const body = request.postData();
			const parsed = JSON.parse(body);
			isRequestOK = JSON.stringify(parsed.rePaths) === JSON.stringify({ foo: 'bar', bar: null });
		}
	});

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

test(`Add schema object property with sub-property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	await page.click('.schema-edit__add-property');

	await page.locator('.property-panel .property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_obj');

	await selectPropertyType(page, 'object');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	const objectRow = page.locator('.grid__row[data-id="test_obj"]');
	await expect(objectRow).toBeVisible();
	await page.click('.schema-edit__add-property');
	await page.locator('.property-panel .property-form__parent').evaluate((select: any) => {
		select.value = 'test_obj';
		select.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	});

	await page.locator('.property-panel .property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_sub_prop_1');

	await selectPropertyType(page, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-panel .property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

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

test(`Remove schema properties`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');

	await removeProperty(page, 'foo');
	await removeProperty(page, 'bar');
	await removeProperty(page, 'test_obj');

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
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
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeAttached();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^test_obj$/ }),
	).not.toBeAttached();
});

test(`Check that the property name is correctly validated`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	await page.click('.schema-edit__add-property');

	let error = page.locator('.property-dialog__control--name .property-dialog__control-error');
	let saveButton = page.locator('.property-dialog__save');

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
});
