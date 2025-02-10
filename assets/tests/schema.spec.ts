import { test, expect } from '@playwright/test';
import { login, logout, uiURL } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

// TODO(Andrea): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Add schema property`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');
	await page.click('.schema-edit__add-property');

	await page.locator('sl-input >> input[name="name"]').fill('foo');
	await page.click('.property-dialog__type-select');
	await page.locator('sl-option', { hasText: 'text' }).click();
	await page.locator('sl-textarea >> textarea[name="description"]').fill('Foo property');

	await page.click('.property-dialog__save');

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeVisible();

	let cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeVisible();

	await page.reload();

	cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeVisible();
});

// TODO(Gianluca): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Edit schema property`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');

	await page.click('.grid__row[data-id="foo"] .schema-edit__property-buttons-edit');
	await page.locator('sl-input >> input[name="name"]').fill('bar');
	await page.click('.property-dialog__save');

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeVisible();

	let fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	let barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).not.toBeVisible();
	await expect(barCell).toBeVisible();

	await page.reload();

	fooCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
		hasText: /^foo$/,
	});
	barCell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^bar$/ });
	await expect(fooCell).not.toBeVisible();
	await expect(barCell).toBeVisible();
});

// TODO(Gianluca): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Check that RePaths are sent correctly`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');

	await page.click('.grid__row[data-id="bar"] .schema-edit__property-buttons-edit');
	await page.locator('sl-input >> input[name="name"]').fill('foo');
	await page.click('.property-dialog__save');

	await page.click('.schema-edit__add-property');
	await page.locator('sl-input >> input[name="name"]').fill('bar');
	await page.click('.property-dialog__type-select');
	await page.locator('sl-option', { hasText: 'text' }).click();
	await page.click('.property-dialog__save');

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/user-schema') && request.method() === 'PUT') {
			isRequestDone = true;
			const body = request.postData();
			const parsed = JSON.parse(body);
			JSON.stringify(parsed.rePaths) === JSON.stringify({ foo: 'bar', bar: null });
		}
	});

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	expect(isRequestDone).toBe(true);

	await expect(page.locator('.schema-grid')).toBeVisible();
});

// TODO(Andrea): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Add schema object property with sub-property`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');
	await page.click('.schema-edit__add-property');

	await page.locator('sl-input >> input[name="name"]').fill('test_obj');
	await page.click('.property-dialog__type-select');
	await page.locator('sl-option', { hasText: 'object' }).click();

	await page.click('.property-dialog__save');

	await page.click('.grid__row[data-id="test_obj"] .schema-edit__editable-object-cell sl-button');
	await page.locator('sl-input >> input[name="name"]').fill('test_sub_prop_1');
	await page.click('.property-dialog__type-select');
	await page.locator('sl-option', { hasText: 'text' }).click();
	await page.click('.property-dialog__save');
	await expect(page.locator('.grid__row--children[data-id="test_obj.test_sub_prop_1"]')).toBeVisible();

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeVisible();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_obj',
		}),
	).toBeVisible();

	await page.waitForTimeout(1000);
	await page.click('.schema-grid__expand-all-button');
	await expect(
		page.locator('.grid__row--children > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_sub_prop_1',
		}),
	).toBeVisible();

	await page.reload();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_obj',
		}),
	).toBeVisible();
	await page.click('.schema-grid__expand-all-button');
	await expect(
		page.locator('.grid__row--children > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_sub_prop_1',
		}),
	).toBeVisible();
});

// TODO(Gianluca): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Remove schema properties`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');

	await page.click('.grid__row[data-id="foo"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

	await page.click('.grid__row[data-id="bar"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

	await page.click('.grid__row[data-id="test_obj"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeVisible();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeVisible();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeVisible();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^test_obj$/ }),
	).not.toBeVisible();

	await page.reload();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeVisible();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ }),
	).not.toBeVisible();
	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^test_obj$/ }),
	).not.toBeVisible();
});

// TODO(Andrea): see the issue https://github.com/meergo/meergo/issues/1200.
test.skip(`Check that the property name is correctly validated`, async ({ page }) => {
	await page.goto(`${uiURL}schema`);

	await page.click('.schema-grid__edit-button');
	await page.click('.schema-edit__add-property');

	let error = page.locator('.property-dialog__control--name .property-dialog__control-error');
	let saveButton = page.locator('.property-dialog__save');

	// Name cannot be empty.
	await page.locator('sl-input >> input[name="name"]').fill('test');
	await page.locator('sl-input >> input[name="name"]').fill('');
	await expect(error).toBeVisible();
	await expect(error).toContainText('Name cannot be empty');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name cannot contain spaces.
	await page.locator('sl-input >> input[name="name"]').fill('my property');
	await expect(error).toBeVisible();
	await expect(error).toContainText('Name cannot contain spaces');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name cannot start with a number.
	await page.locator('sl-input >> input[name="name"]').fill('3foo');
	await expect(error).toBeVisible();
	await expect(error).toContainText('Name cannot start with a number');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name must start with an ASCII alphabet character or an
	// underscore.
	await page.locator('sl-input >> input[name="name"]').fill('$foo');
	await expect(error).toBeVisible();
	await expect(error).toContainText('Name must start with an ASCII alphabet character or an underscore');
	await expect(saveButton).toHaveAttribute('disabled');

	// Name must contain only ASCII alphabet characters, digits and
	// underscores.
	await page.locator('sl-input >> input[name="name"]').fill('foo_3bar');
	await expect(error).not.toBeVisible();
	await expect(saveButton).not.toHaveAttribute('disabled');
	await page.locator('sl-input >> input[name="name"]').fill('foo$bar');
	await expect(error).toBeVisible();
	await expect(error).toContainText('Name must contain only ASCII alphabet characters, digits and underscores');
	await expect(saveButton).toHaveAttribute('disabled');
});
