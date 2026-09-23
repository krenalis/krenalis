import { test, expect } from '@playwright/test';
import { login, logout, adminURL } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Show consent management in the settings menu`, async ({ page }) => {
	await page.goto(`${adminURL}/settings/privacy`);
	await expect(
		page.locator('.sidebar__sub-item-text').getByText('Consent management', { exact: true }),
	).toBeVisible();
});

test(`Select the profile property of a consent purpose`, async ({ page }) => {
	const purposes = [
		{
			id: 'existing-purpose',
			name: 'Existing purpose',
			eventConsentLocations: [],
			profileConsentLocation: { property: 'accepted' },
		},
		{
			id: 'existing-json-purpose',
			name: 'Existing JSON purpose',
			eventConsentLocations: [],
			profileConsentLocation: { property: 'attributes', jsonKey: 'marketing' },
		},
	];
	await page.route('**/v1/consent-purposes', async (route) => {
		if (route.request().method() === 'POST') {
			purposes.push({ id: 'created-purpose', ...route.request().postDataJSON() });
			await route.fulfill({ json: {} });
			return;
		}
		await route.fulfill({
			json: { purposes },
		});
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({
			json: {
				kind: 'object',
				properties: [
					{ name: 'accepted', type: { kind: 'boolean' } },
					{
						name: 'nested',
						type: {
							kind: 'object',
							properties: [
								{ name: 'ignored', type: { kind: 'string' } },
								{ name: 'metadata', type: { kind: 'json' } },
								{ name: 'allowed', type: { kind: 'boolean' } },
							],
						},
					},
					{ name: 'attributes', type: { kind: 'json' } },
				],
			},
		});
	});

	await page.goto(`${adminURL}/settings/privacy`);
	await page.getByRole('button', { name: 'Add consent purpose' }).click();

	const dialog = page.locator('.privacy__dialog[open]');
	await dialog.locator('[part="overlay"]').dispatchEvent('click');
	await expect(dialog).toHaveJSProperty('open', true);
	await expect(dialog.locator('.privacy__dialog-name')).toHaveJSProperty('size', 'medium');
	await expect(dialog.locator('.privacy__dialog-purpose-code')).toHaveJSProperty('size', 'medium');
	await expect(dialog.locator('.privacy__dialog-name [part="form-control-label"]')).toHaveCSS('font-size', '14px');
	await expect(dialog.locator('.privacy__dialog-profile-path-dropdown')).toHaveCSS('margin-top', '17px');

	const profilePath = dialog.locator('.privacy__dialog-profile-path');
	await expect(profilePath).toHaveJSProperty('size', 'medium');
	await expect(profilePath.locator('input')).toHaveCSS('caret-color', 'rgba(0, 0, 0, 0)');
	await expect(profilePath.locator('input')).toHaveCSS('cursor', 'default');
	await expect(dialog.locator('.privacy__dialog-path-label').nth(0)).toHaveCSS('font-size', '14px');
	await expect(dialog.locator('.privacy__dialog-path-label').nth(1)).toHaveCSS('font-size', '14px');
	await expect(dialog.getByRole('img', { name: 'About profile property' })).toBeVisible();
	const pathDescriptions = dialog.locator('.schema-property-grid__tooltip-content');
	await expect(pathDescriptions.nth(0).locator('p')).toHaveText([
		'In incoming events, consent for this purpose is read from a property in context.consents. Enter the code used for this purpose in the platform you use to manage consent.',
		'If you add multiple locations, they are checked from top to bottom. The first location with a consent value is used.',
	]);
	await expect(pathDescriptions.nth(1).locator('p')).toHaveText([
		'Consent for this purpose is represented in profiles using a profile property. Select a boolean property, or select a json property and specify the key.',
	]);
	await profilePath.click();
	const options = dialog.locator('.privacy__dialog-profile-path-menu sl-menu-item');
	await expect(options).toHaveCount(4);
	await expect(options.locator('.privacy__dialog-profile-path-option')).toHaveText([
		'accepted',
		'nested.metadata',
		'nested.allowed',
		'attributes',
	]);
	await expect(options.locator('.privacy__dialog-profile-path-option-type')).toHaveText([
		'boolean',
		'json',
		'boolean',
		'json',
	]);
	await expect(options.locator('.privacy__dialog-profile-path-option-type').nth(0)).toHaveCSS(
		'font-family',
		/monospace/,
	);

	await options.nth(0).click();
	await expect(profilePath).toHaveJSProperty('readonly', true);
	await expect(profilePath.locator('input')).toHaveValue('accepted');
	await profilePath.locator('input').pressSequentially('changed');
	await expect(profilePath.locator('input')).toHaveValue('accepted');
	await expect(profilePath.locator('input')).toHaveCSS('caret-color', 'rgba(0, 0, 0, 0)');
	await expect(profilePath.locator('input')).toHaveCSS('cursor', 'default');
	await expect(profilePath.locator('input')).toHaveCSS('transform', 'matrix(1, 0, 0, 1, 0, 1)');
	await expect(options.nth(0)).not.toBeVisible();
	const profilePathMessage = dialog.locator('.privacy__dialog-profile-path-message');
	await expect(profilePathMessage).toHaveText('This property is also used by other purposes.');
	await expect(profilePathMessage.locator('sl-icon')).toHaveAttribute('name', 'exclamation-triangle');
	const profilePathDropdown = dialog.locator('.privacy__dialog-profile-path-dropdown');
	await profilePathDropdown.evaluate((dropdown) => {
		dropdown.setAttribute('data-show-count', '0');
		dropdown.addEventListener('sl-show', () => {
			const count = Number(dropdown.getAttribute('data-show-count'));
			dropdown.setAttribute('data-show-count', String(count + 1));
		});
	});

	await profilePath.getByRole('button', { name: 'Clear profile property' }).click();
	await expect(profilePath.locator('input')).toHaveValue('');
	await expect(profilePath.locator('input')).toHaveAttribute('placeholder', 'Select a profile property');
	await expect(profilePath.getByRole('button', { name: 'Clear profile property' })).toHaveCount(0);
	await expect(options.nth(0)).not.toBeVisible();
	await expect(profilePathDropdown).toHaveAttribute('data-show-count', '0');

	await profilePath.click();
	await options.nth(1).click();
	await expect(profilePath).toHaveJSProperty('readonly', false);
	await expect(profilePath.locator('[slot="prefix"]')).toHaveText('nested.metadata.');
	await expect(profilePath.locator('input')).toHaveCSS('transform', 'matrix(1, 0, 0, 1, 0, 1)');
	const purposeCodePrefix = dialog.locator('.privacy__dialog-purpose-code [part="prefix"]');
	const purposeCodePrefixBackground = await purposeCodePrefix.evaluate(
		(prefix) => getComputedStyle(prefix).backgroundColor,
	);
	await expect(profilePath.locator('[part="prefix"]')).toHaveCSS('background-color', purposeCodePrefixBackground);
	await expect(profilePath.locator('input')).toHaveAttribute('placeholder', 'key');
	await expect(profilePath.locator('input')).toBeFocused();
	await expect(profilePath.getByRole('button', { name: 'Clear profile property' })).toBeVisible();
	await profilePath.locator('input').click();
	await expect(options.nth(0)).not.toBeVisible();
	await profilePath.locator('input').pressSequentially('purpose code');
	await expect(profilePath.locator('input')).toHaveValue('purpose code');
	await expect(options.nth(0)).not.toBeVisible();
	await profilePath.getByRole('button', { name: 'Clear profile property' }).click();
	await expect(profilePath.locator('input')).toHaveValue('');
	await expect(profilePath.locator('[slot="prefix"]')).toHaveCount(0);
	await expect(options.nth(0)).not.toBeVisible();
	await expect(profilePathMessage).not.toBeVisible();
	await expect(profilePathDropdown).toHaveAttribute('data-show-count', '1');

	await profilePath.click();
	await options.nth(3).click();
	await dialog.locator('.privacy__dialog-name input').fill('Marketing');
	await profilePath.locator('input').fill('marketing');
	await expect(profilePathMessage).not.toBeVisible();
	await profilePath.locator('input').blur();
	await expect(profilePathMessage).toHaveText('This property and key are also used by other purposes.');
	await expect(profilePathMessage.locator('sl-icon')).toHaveAttribute('name', 'exclamation-triangle');
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes') && request.method() === 'POST',
	);
	await dialog.locator('.privacy__dialog-save').click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing',
		eventConsentLocations: [],
		profileConsentLocation: { property: 'attributes', jsonKey: 'marketing' },
	});
	await expect(dialog).toHaveCount(0);
	await page
		.locator('.grid__row')
		.filter({ hasText: 'Existing JSON purpose' })
		.getByRole('button', { name: 'Edit...' })
		.click();
	await expect(profilePathMessage).toHaveText('This property and key are also used by other purposes.');
});

test(`Add and remove purpose codes of a consent purpose`, async ({ page }) => {
	await page.route('**/v1/consent-purposes', async (route) => {
		await route.fulfill({ json: { purposes: [] } });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({ json: { kind: 'object', properties: [] } });
	});
	await page.goto(`${adminURL}/settings/privacy`);
	await page.getByRole('button', { name: 'Add consent purpose' }).click();

	const dialog = page.locator('.privacy__dialog[open]');
	const rows = dialog.locator('.privacy__dialog-purpose-code-row');
	const inputs = rows.locator('input');
	await dialog.locator('.privacy__dialog-name input').fill('Marketing');
	await expect(rows.locator('.privacy__dialog-purpose-code-remove')).toHaveCount(0);
	await inputs.nth(0).fill('marketing');

	// A blank row is removable, but the only non-empty row is not.
	await rows.nth(0).locator('.privacy__dialog-purpose-code-add sl-button').click();
	await expect(inputs).toHaveCount(2);
	await expect(inputs.nth(1)).toBeFocused();
	await expect.poll(() => rows.nth(1).evaluate((row) => row.getAnimations().length)).toBe(0);
	await expect(inputs.nth(1)).toBeFocused();
	await expect(rows.nth(0).locator('.privacy__dialog-purpose-code-remove')).toHaveCount(0);
	await rows.nth(1).locator('.privacy__dialog-purpose-code-remove sl-button').click();
	await expect(inputs).toHaveCount(1);
	await expect(inputs.nth(0)).toHaveValue('marketing');

	// The first row can be removed when another row has a value.
	await rows.nth(0).locator('.privacy__dialog-purpose-code-add sl-button').click();
	await expect(inputs.nth(1)).toBeFocused();
	await inputs.nth(1).fill('oldmarketing');
	await expect(rows.locator('.privacy__dialog-purpose-code-remove')).toHaveCount(2);
	await rows.nth(0).locator('.privacy__dialog-purpose-code-add sl-button').click();
	await expect(inputs).toHaveCount(3);
	await expect(inputs.nth(1)).toHaveValue('');
	await expect(inputs.nth(1)).toBeFocused();
	await expect(inputs.nth(2)).toHaveValue('oldmarketing');
	await rows.nth(1).locator('.privacy__dialog-purpose-code-remove sl-button').click();
	await expect(inputs).toHaveCount(2);
	await expect(inputs.nth(0)).toHaveValue('marketing');
	await expect(inputs.nth(1)).toHaveValue('oldmarketing');
	const firstPurposeCodeRemove = rows.nth(0).locator('.privacy__dialog-purpose-code-remove');
	await firstPurposeCodeRemove.hover();
	await expect(firstPurposeCodeRemove).toHaveJSProperty('open', true);
	await firstPurposeCodeRemove.locator('sl-button').dblclick();
	await expect(inputs).toHaveCount(1);
	await expect(inputs.nth(0)).toHaveValue('oldmarketing');
	await expect(dialog.locator('.privacy__dialog-path-label').nth(0)).toContainText('Consent in events');

	// Keys are literal strings, and no more than five inputs can be added.
	const purposeCodes = ['oldmarketing', '#CFK567', 'purpose code', 'a.b', 'last'];
	for (let i = 1; i < purposeCodes.length; i++) {
		await rows
			.nth(i - 1)
			.locator('.privacy__dialog-purpose-code-add sl-button')
			.click();
		await expect(inputs).toHaveCount(i + 1);
		await expect(inputs.nth(i)).toBeFocused();
		await inputs.nth(i).fill(purposeCodes[i]);
	}
	const addButtons = rows.locator('.privacy__dialog-purpose-code-add sl-button button');
	await expect(addButtons).toHaveCount(5);
	for (const button of await addButtons.all()) {
		await expect(button).toBeDisabled();
	}
	await inputs.nth(4).fill(purposeCodes[0]);
	await expect(dialog.locator('.privacy__dialog-error')).toHaveCount(0);
	await inputs.nth(4).blur();
	await expect(dialog.locator('.privacy__dialog-error')).toHaveText('Purpose code "oldmarketing" is duplicated');
	await dialog.locator('.privacy__dialog-save').click();
	await expect(dialog.locator('.privacy__dialog-error')).toHaveText('Purpose code "oldmarketing" is duplicated');
	await inputs.nth(4).fill(purposeCodes[4]);
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes') && request.method() === 'POST',
	);
	await dialog.locator('.privacy__dialog-save').click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing',
		eventConsentLocations: purposeCodes.map((purposeCode) => ({ purposeCode })),
		profileConsentLocation: null,
	});
	await expect(dialog).toHaveCount(0);
});

test(`Edit a consent purpose without setting the initial focus`, async ({ page }) => {
	await page.route('**/v1/consent-purposes', async (route) => {
		await route.fulfill({
			json: {
				purposes: [
					{
						id: 'purpose',
						name: 'Marketing',
						eventConsentLocations: [{ purposeCode: 'marketing' }],
						profileConsentLocation: { property: 'accepted' },
					},
				],
			},
		});
	});
	await page.route('**/v1/consent-purposes/purpose', async (route) => {
		await route.fulfill({ json: {} });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({
			json: { kind: 'object', properties: [{ name: 'accepted', type: { kind: 'boolean' } }] },
		});
	});
	await page.goto(`${adminURL}/settings/privacy`);
	const dialog = page.locator('.privacy__dialog').nth(1);
	await dialog.evaluate((dialog) => {
		dialog.setAttribute('data-focus-count', '0');
		dialog.addEventListener('focusin', () => {
			const count = Number(dialog.getAttribute('data-focus-count'));
			dialog.setAttribute('data-focus-count', String(count + 1));
		});
		dialog.addEventListener('sl-after-show', (event) => {
			if (event.target === dialog) {
				dialog.setAttribute('data-initial-focus', String(dialog.matches(':focus-within')));
			}
		});
	});
	await page.getByRole('button', { name: 'Edit...' }).click();
	await expect(dialog).toHaveAttribute('data-initial-focus', 'false');
	await expect(dialog).toHaveAttribute('data-focus-count', '0');
	await expect(dialog.locator('.privacy__dialog-purpose-code input')).toHaveValue('marketing');
	await expect(dialog.locator('.privacy__dialog-profile-path input')).toHaveValue('accepted');
	await dialog.locator('.privacy__dialog-name input').fill('Marketing communications');
	await expect(dialog).toHaveAttribute('data-focus-count', /^[1-9]\d*$/);
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes/purpose') && request.method() === 'PUT',
	);
	await dialog.locator('.privacy__dialog-save').click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing communications',
		eventConsentLocations: [{ purposeCode: 'marketing' }],
		profileConsentLocation: { property: 'accepted' },
	});
	await expect(dialog).toHaveJSProperty('open', false);
});

test(`Validate existing consent locations and allow clearing an invalid profile location`, async ({ page }) => {
	let puts = 0;
	await page.route('**/v1/consent-purposes', async (route) => {
		await route.fulfill({
			json: {
				purposes: [
					{
						id: 'purpose',
						name: 'Marketing',
						eventConsentLocations: [{ purposeCode: 'mar\nketing' }],
						profileConsentLocation: { property: 'consents', jsonKey: 'a\t' },
					},
				],
			},
		});
	});
	await page.route('**/v1/consent-purposes/purpose', async (route) => {
		puts++;
		await route.fulfill({ json: {} });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({
			json: { kind: 'object', properties: [{ name: 'consents', type: { kind: 'json' } }] },
		});
	});
	await page.goto(`${adminURL}/settings/privacy`);
	await page.getByRole('button', { name: 'Edit...' }).click();
	const dialog = page.locator('.privacy__dialog[open]');
	const save = dialog.locator('.privacy__dialog-save');
	await save.click();
	await expect(dialog.locator('.privacy__dialog-error')).toHaveText(
		'Consent keys must not contain invisible characters',
	);
	expect(puts).toBe(0);
	await dialog.locator('.privacy__dialog-purpose-code input').fill('marketing');
	await save.click();
	await expect(dialog.locator('.privacy__dialog-error')).toHaveText(
		'Consent keys must not contain invisible characters',
	);
	expect(puts).toBe(0);
	await dialog.getByRole('button', { name: 'Clear profile property' }).click();
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes/purpose') && request.method() === 'PUT',
	);
	await save.click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing',
		eventConsentLocations: [{ purposeCode: 'marketing' }],
		profileConsentLocation: null,
	});
	await expect(dialog).toHaveCount(0);
});

test(`Validate authored consent keys and save literal JSON keys separately`, async ({ page }) => {
	let posts = 0;
	await page.route('**/v1/consent-purposes', async (route) => {
		if (route.request().method() === 'POST') posts++;
		await route.fulfill({ json: { purposes: [] } });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({ json: { kind: 'object', properties: [{ name: 'consents', type: { kind: 'json' } }] } });
	});
	await page.goto(`${adminURL}/settings/privacy`);
	await page.getByRole('button', { name: 'Add consent purpose' }).click();
	const dialog = page.locator('.privacy__dialog[open]');
	await dialog.locator('.privacy__dialog-name input').fill('Marketing');
	const purposeCodeInput = dialog.locator('.privacy__dialog-purpose-code input');
	const save = dialog.locator('.privacy__dialog-save');
	for (const key of [' ', 'a\t', 'a\u200b', '😀'.repeat(1025)]) {
		await purposeCodeInput.fill(key);
		await save.click();
		await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
		expect(posts).toBe(0);
	}
	await purposeCodeInput.fill(' purpose.a.b ');
	const profilePath = dialog.locator('.privacy__dialog-profile-path');
	await profilePath.click();
	await dialog.locator('.privacy__dialog-profile-path-option').click();
	await save.click();
	await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
	expect(posts).toBe(0);
	const profileKey = profilePath.locator('input');
	await profileKey.fill('a\t');
	await profileKey.blur();
	await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
	expect(posts).toBe(0);
	for (const key of [' ', 'a\t', '😀'.repeat(1025)]) {
		await profileKey.fill(key);
		await save.click();
		await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
		expect(posts).toBe(0);
	}
	const prefix = ' purpose.a["b"]\\say "yes" ';
	const key = prefix + '😀'.repeat(1024 - Array.from(prefix).length);
	await profileKey.fill(key);
	await expect(dialog.locator('.privacy__dialog-error:visible')).toHaveCount(0);
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes') && request.method() === 'POST',
	);
	await save.click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing',
		eventConsentLocations: [{ purposeCode: ' purpose.a.b ' }],
		profileConsentLocation: { property: 'consents', jsonKey: key },
	});
	await expect(dialog).toHaveCount(0);
});

test(`Validate changed API JSON keys and clear the profile property`, async ({ page }) => {
	const originalProfileLocation = { property: 'consents', jsonKey: 'a\t' };
	let puts = 0;
	await page.route('**/v1/consent-purposes', async (route) => {
		await route.fulfill({
			json: {
				purposes: [
					{
						id: 'purpose',
						name: 'Marketing',
						eventConsentLocations: [],
						profileConsentLocation: originalProfileLocation,
					},
				],
			},
		});
	});
	await page.route('**/v1/consent-purposes/purpose', async (route) => {
		puts++;
		await route.fulfill({ json: {} });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({
			json: {
				kind: 'object',
				properties: ['consents', 'other'].map((name) => ({ name, type: { kind: 'json' } })),
			},
		});
	});
	await page.goto(`${adminURL}/settings/privacy`);
	await page.getByRole('button', { name: 'Edit...' }).click();
	const dialog = page.locator('.privacy__dialog[open]');
	const profilePath = dialog.locator('.privacy__dialog-profile-path');
	await profilePath.locator('sl-icon[name="chevron-down"]').click();
	const otherProfilePathOption = dialog
		.locator('.privacy__dialog-profile-path-option')
		.getByText('other', { exact: true });
	await otherProfilePathOption.click();
	await expect(otherProfilePathOption).not.toBeVisible();
	await profilePath.locator('input').fill('a\t');
	await dialog.locator('.privacy__dialog-save').click();
	await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
	expect(puts).toBe(0);
	await profilePath.getByRole('button', { name: 'Clear profile property' }).click();
	await expect(dialog.locator('.privacy__dialog-error:visible')).toHaveCount(0);
	await profilePath.locator('sl-icon[name="chevron-down"]').click();
	await dialog.locator('.privacy__dialog-profile-path-option').getByText('consents', { exact: true }).click();
	await profilePath.locator('input').fill('a\t');
	await dialog.locator('.privacy__dialog-save').click();
	await expect(dialog.locator('.privacy__dialog-error:visible')).toBeVisible();
	expect(puts).toBe(0);
	await profilePath.getByRole('button', { name: 'Clear profile property' }).click();
	const savedPurpose = page.waitForRequest(
		(request) => request.url().endsWith('/v1/consent-purposes/purpose') && request.method() === 'PUT',
	);
	await dialog.locator('.privacy__dialog-save').click();
	expect((await savedPurpose).postDataJSON()).toEqual({
		name: 'Marketing',
		eventConsentLocations: [],
		profileConsentLocation: null,
	});
	await expect(dialog).toHaveCount(0);
});

test(`Change the workspace name`, async ({ page }) => {
	await page.goto(`${adminURL}/settings/general`);
	await page.locator('.general-settings__name >> input').fill('Test workspace');
	await page.click('.general-settings__save-workspace-button');
	await page.reload();
	await expect(page.locator('.general-settings__name >> input')).toHaveValue('Test workspace');
	await expect(page.locator('.workspace-selector__value')).toContainText('Test workspace');
	await page.locator('.general-settings__name >> input').fill('Workspace');
	await page.click('.general-settings__save-workspace-button');
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();
	await expect(page.locator('.general-settings__name >> input')).toHaveValue('Workspace');
	await expect(page.locator('.workspace-selector__value')).toContainText('Workspace');
});

test(`Change the UI user profile properties`, async ({ page }) => {
	await page.goto(`${adminURL}/settings/general`);

	const userProfileFirstName = page.locator('.general-settings__user-profile-first-name sl-input >> input');
	const userProfileLastName = page.locator('.general-settings__user-profile-last-name sl-input >> input');
	const userProfileAdditionalLine = page.locator('.general-settings__user-profile-extra sl-input >> input');
	const userProfileImage = page.locator('.general-settings__profile-image sl-input >> input');

	await userProfileFirstName.fill('first_name');
	await userProfileLastName.fill('last_name');
	await userProfileAdditionalLine.fill('email');
	await userProfileImage.fill('dummy_id'); // Currently in the default schema we don't have any property for the image.

	await page.click('.general-settings__save-workspace-button');

	await expect(userProfileFirstName).toHaveValue('first_name');
	await expect(userProfileLastName).toHaveValue('last_name');
	await expect(userProfileAdditionalLine).toHaveValue('email');
	await expect(userProfileImage).toHaveValue('dummy_id');

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	await expect(userProfileFirstName).toHaveValue('first_name');
	await expect(userProfileLastName).toHaveValue('last_name');
	await expect(userProfileAdditionalLine).toHaveValue('email');
	await expect(userProfileImage).toHaveValue('dummy_id');
});

test(`Change the automatic execution of the identity resolution`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/rules`);

	const automaticExecution = page.locator('.identifiers__automatic-execution');
	const automaticExecutionLabel = page.locator('.identifiers__automatic-execution >> label');

	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await automaticExecution.click();

	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await page.click('.identifiers__save-button');
	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();
	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await automaticExecution.click();

	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await page.click('.identifiers__save-button');
	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();
	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);
});

test(`Change the identifiers`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/rules`);

	expect(await page.locator('.identifiers__identifier').count()).toBe(0);

	const addIdentifierButton = page.locator('.identifiers__add');

	// Add the identifiers.
	await addIdentifierButton.click();
	expect(await page.locator('.identifiers__identifier').count()).toBe(1);
	await addIdentifierButton.click();
	await addIdentifierButton.click();
	expect(await page.locator('.identifiers__identifier').count()).toBe(3);

	// Fill the identifiers.
	const identifiers = page.locator('.identifiers__identifier sl-input');
	await identifiers.nth(0).evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'email');
	await identifiers.nth(1).evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'first_name');
	await identifiers.nth(2).evaluate((el: any, value) => {
		el.value = value;
		el.dispatchEvent(new CustomEvent('sl-input', { bubbles: true, composed: true }));
	}, 'last_name');

	await page.waitForTimeout(1000); // Add a timeout to ensure that the React state is synced with the form controls.

	await page.click('.identifiers__save-button');
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	const identInputs = page.locator('.identifiers__identifier sl-input >> input');
	await expect(identInputs.nth(0)).toHaveValue('email');
	await expect(identInputs.nth(1)).toHaveValue('first_name');
	await expect(identInputs.nth(2)).toHaveValue('last_name');
});

test(`Sort the identifiers`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/rules`);

	const identifiers = page.locator('.identifiers__identifier');
	await identifiers.nth(0).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(0).locator('.identifiers__mapping-down').click();
	await identifiers.nth(2).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(2).locator('.identifiers__mapping-up').click();
	await page.click('.identifiers__save-button');
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();
	const identInputs = page.locator('.identifiers__identifier sl-input >> input');
	await expect(identInputs.nth(0)).toHaveValue('first_name');
	await expect(identInputs.nth(1)).toHaveValue('last_name');
	await expect(identInputs.nth(2)).toHaveValue('email');
});

test(`Remove the identifiers`, async ({ page }) => {
	await page.goto(`${adminURL}/profile-unification/rules`);

	let identifiers = page.locator('.identifiers__identifier');

	await identifiers.nth(2).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(2).locator('.identifiers__mapping-remove').click();

	await identifiers.nth(1).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(1).locator('.identifiers__mapping-remove').click();

	await identifiers.nth(0).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(0).locator('.identifiers__mapping-remove').click();

	await page.click('.identifiers__save-button');
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await page.reload();

	expect(await page.locator('.identifiers__identifier').count()).toBe(0);
});
