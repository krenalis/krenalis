import { test, expect } from '@playwright/test';
import { login, logout, adminURL, logValidationErrors } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Preview schema metadata changes`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
	let previewRequests = 0;
	page.on('request', (request) => {
		if (request.url().includes('/profiles/schema/preview') && request.method() === 'PUT') {
			previewRequests++;
		}
	});
	// Record every label so the test also detects changes that are too brief for
	// a final-state assertion.
	await applyButton.evaluate((button: any) => {
		button.observedLabels = [button.textContent.trim()];
		button.labelObserver = new MutationObserver(() => {
			button.observedLabels.push(button.textContent.trim());
		});
		button.labelObserver.observe(button, { childList: true, subtree: true, characterData: true });
	});

	await page.click('.grid__row[data-id="email"] .schema-edit__property-buttons-edit');
	const description = page.locator('sl-textarea >> textarea[name="description"]');
	const originalDescription = await description.inputValue();
	await description.fill('Updated description');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.property-dialog__save');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('disabled');
	await page.waitForTimeout(1000);
	expect(previewRequests).toBe(0);
	const observedLabels = await applyButton.evaluate((button: any) => {
		button.labelObserver.disconnect();
		return button.observedLabels;
	});
	expect(observedLabels).not.toContain('Review and apply changes...');

	await applyButton.click();
	const dialog = page.locator('.schema-edit__queries');
	await expect(dialog).toHaveAttribute('label', 'Apply schema changes?');
	await expect(dialog.locator('.schema-edit__no-query')).toHaveText(
		'These changes affect only the schema definition. The data warehouse will not be modified.',
	);
	await expect(dialog.locator('.schema-edit__apply-alter-button')).toHaveAttribute('variant', 'primary');
	await dialog.locator('.schema-edit__queries-buttons sl-button').first().click();

	await page.click('.grid__row[data-id="email"] .schema-edit__property-buttons-edit');
	await description.fill(originalDescription);
	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.
	await page.click('.property-dialog__save');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
	await page.waitForTimeout(1000);
	expect(previewRequests).toBe(0);

	// Cache a preview for a change that requires warehouse DDL.
	await page.click('.schema-edit__add-property');
	await page.locator('.property-dialog__name-input').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'temporary_preview_property');
	await page.locator('.property-dialog__type-select').evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-change', { bubbles: true, composed: true }));
	}, 'string');
	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.
	await Promise.all([
		page.waitForResponse(
			(response) => response.url().includes('/profiles/schema/preview') && response.request().method() === 'PUT',
		),
		page.click('.property-dialog__save'),
	]);
	await expect(applyButton).toHaveText('Review and apply changes...');

	// Add a metadata change before removing the temporary property.
	await page.click('.grid__row[data-id="email"] .schema-edit__property-buttons-edit');
	await description.fill('Updated description');
	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.
	await page.click('.property-dialog__save');
	await expect(applyButton).toHaveText('Review and apply changes...');

	// A failed preview must not leave the warehouse status of an earlier
	// version of the schema on the apply button.
	await page.route(
		'**/profiles/schema/preview',
		async (route) => {
			await route.fulfill({ status: 500, contentType: 'application/json', body: '{}' });
		},
		{ times: 1 },
	);
	const failedPreviewResponse = page.waitForResponse(
		(response) =>
			response.url().includes('/profiles/schema/preview') &&
			response.request().method() === 'PUT' &&
			response.status() === 500,
	);
	await page.click('.grid__row[data-id="temporary_preview_property"] .schema-edit__property-buttons-remove');
	await page.click('.schema-edit__confirm-remove-property');
	await failedPreviewResponse;
	await expect(applyButton).not.toHaveAttribute('loading');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).not.toHaveAttribute('disabled');
});

test(`Add schema property`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/schema`);

	await page.click('.schema-grid__alter-button');
	const applyButton = page.locator('.schema-edit__header-apply-button');
	await expect(applyButton).toHaveText('Apply changes');
	await expect(applyButton).toHaveAttribute('disabled');
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

	const previewResponse = page.waitForResponse(
		(response) => response.url().includes('/profiles/schema/preview') && response.request().method() === 'PUT',
	);
	await page.click('.property-dialog__save');
	await expect(applyButton).toHaveAttribute('loading');
	await expect(applyButton).toHaveAttribute('disabled');
	await previewResponse;
	await logValidationErrors(page, ['.property-dialog__control-error']);
	await expect(applyButton).toHaveText('Review and apply changes...');
	await expect(applyButton).not.toHaveAttribute('disabled');

	await applyButton.click();
	await expect(page.locator('.schema-edit__queries')).toHaveAttribute('label', 'Review changes');
	await expect(page.locator('.schema-edit__apply-alter-button')).toHaveAttribute('variant', 'danger');
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

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Review and apply changes...');
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

	await page.waitForTimeout(2000); // Add a timeout to ensure that editable schema in the React state is synced with the newly added property.

	let isRequestOK = false;
	page.on('request', async (request) => {
		if (request.url().includes('/profiles/schema') && request.method() === 'PUT') {
			const body = request.postData();
			const parsed = JSON.parse(body);
			isRequestOK = JSON.stringify(parsed.rePaths) === JSON.stringify({ foo: 'bar', bar: null });
		}
	});

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Review and apply changes...');
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

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Review and apply changes...');
	await page.click('.schema-edit__header-apply-button');

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

	await expect(page.locator('.schema-edit__header-apply-button')).toHaveText('Review and apply changes...');
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
