import { test, expect } from '@playwright/test';
import { login, logout, adminURL, logValidationErrors } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Add schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	await page.click('.schema-edit__add-property');

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await page.locator('.property-dialog__type-select').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	}, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	let cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeAttached();

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	cell = page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', { hasText: /^foo$/ });
	await expect(cell).toBeAttached();
});

test(`Edit schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');

	await page.click('.grid__row[data-id="foo"] .schema-edit__property-buttons-edit');

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
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

	await page.click('.grid__row[data-id="bar"] .schema-edit__property-buttons-edit');

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'foo');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.schema-edit__add-property');

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'bar');

	await page.locator('.property-dialog__type-select').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	}, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/profiles/schema') && request.method() === 'PUT') {
			const body = request.postData();
			const parsed = JSON.parse(body);
			isRequestDone = JSON.stringify(parsed.rePaths) === JSON.stringify({ foo: 'bar', bar: null });
		}
	});

	await page.click('.schema-edit__header-apply-button');
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	expect(isRequestDone).toBe(true);

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

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_obj');

	await page.locator('.property-dialog__type-select').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	}, 'object');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.grid__row[data-id="test_obj"] .schema-edit__editable-object-cell sl-button');

	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'test_sub_prop_1');

	await page.locator('.property-dialog__type-select').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	}, 'string');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await logValidationErrors(page, ['.property-dialog__control-error']);

	await page.click('.schema-edit__header-apply-button');

	await page.locator('.schema-edit__apply-alter-button').scrollIntoViewIfNeeded();
	await page.click('.schema-edit__apply-alter-button');

	await expect(page.locator('.schema-grid')).toBeAttached();

	await expect(
		page.locator('.grid__row > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_obj',
		}),
	).toBeAttached();

	await page.waitForTimeout(8000); // Ensures that the Admin console has had enough time to poll the server to know if the update is completed (polling happens every 3 seconds) and to refetch the schema.
	await page.click('.schema-grid__expand-all-button');
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
	await page.click('.schema-grid__expand-all-button');
	await expect(
		page.locator('.grid__row--children > .grid__cell:first-child > .grid__cell-content', {
			hasText: 'test_sub_prop_1',
		}),
	).toBeAttached();
});

test(`Remove schema properties`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');

	await page.click('.grid__row[data-id="foo"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

	await page.click('.grid__row[data-id="bar"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

	await page.click('.grid__row[data-id="test_obj"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');

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
