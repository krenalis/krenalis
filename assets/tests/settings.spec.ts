import { test, expect } from '@playwright/test';
import { login, logout, uiURL } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Change the workspace name`, async ({ page }) => {
	await page.goto(`${uiURL}settings/general`);
	await page.locator('.general-settings__name >> input').fill('Test workspace');
	await page.click('.general-settings__save-workspace-button');
	await page.reload();
	await expect(page.locator('.general-settings__name >> input')).toHaveValue('Test workspace');
	await expect(page.locator('.workspace-selector__value')).toContainText('Test workspace');
	await page.locator('.general-settings__name >> input').fill('Workspace');
	await page.click('.general-settings__save-workspace-button');
	await page.reload();
	await expect(page.locator('.general-settings__name >> input')).toHaveValue('Workspace');
	await expect(page.locator('.workspace-selector__value')).toContainText('Workspace');
});

test(`Change the displayed properties`, async ({ page }) => {
	await page.goto(`${uiURL}settings/general`);

	const displayedFirstName = page.locator('.general-settings__displayed-first-name sl-input >> input');
	const displayedLastName = page.locator('.general-settings__displayed-last-name sl-input >> input');
	const displayedAdditionalLine = page.locator('.general-settings__displayed-information sl-input >> input');
	const displayedImage = page.locator('.general-settings__displayed-image sl-input >> input');

	await displayedFirstName.fill('first_name');
	await displayedLastName.fill('last_name');
	await displayedAdditionalLine.fill('email');
	await displayedImage.fill('dummy_id'); // Currently in the default schema we don't have any property for the image.

	await page.click('.general-settings__save-workspace-button');

	await expect(displayedFirstName).toHaveValue('first_name');
	await expect(displayedLastName).toHaveValue('last_name');
	await expect(displayedAdditionalLine).toHaveValue('email');
	await expect(displayedImage).toHaveValue('dummy_id');

	await page.reload();

	await expect(displayedFirstName).toHaveValue('first_name');
	await expect(displayedLastName).toHaveValue('last_name');
	await expect(displayedAdditionalLine).toHaveValue('email');
	await expect(displayedImage).toHaveValue('dummy_id');
});

test(`Change the automatic execution of the identity resolution`, async ({ page }) => {
	await page.goto(`${uiURL}settings/identity-resolution`);

	const automaticExecution = page.locator('.identifiers__automatic-execution');
	const automaticExecutionLabel = page.locator('.identifiers__automatic-execution >> label');

	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await automaticExecution.click();

	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await page.click('.identifiers__save-button');
	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await page.reload();
	await expect(automaticExecutionLabel).not.toHaveClass(/checkbox--checked/);

	await automaticExecution.click();

	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await page.click('.identifiers__save-button');
	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);

	await page.reload();
	await expect(automaticExecutionLabel).toHaveClass(/checkbox--checked/);
});

test(`Change the identifiers`, async ({ page }) => {
	await page.goto(`${uiURL}settings/identity-resolution`);

	expect(await page.locator('.identifiers__identifier').count()).toBe(0);

	const addIdentifierButton = page.locator('.identifiers__add');

	// Add the identifiers.
	await addIdentifierButton.click();
	expect(await page.locator('.identifiers__identifier').count()).toBe(1);
	await addIdentifierButton.click();
	await addIdentifierButton.click();
	expect(await page.locator('.identifiers__identifier').count()).toBe(3);

	// Fill the identifiers.
	const identInputs = page.locator('.identifiers__identifier sl-input >> input');
	await identInputs.nth(0).fill('email');
	await identInputs.nth(1).fill('first_name');
	await identInputs.nth(2).fill('last_name');
	await page.click('.identifiers__save-button');
	await page.reload();
	await expect(identInputs.nth(0)).toHaveValue('email');
	await expect(identInputs.nth(1)).toHaveValue('first_name');
	await expect(identInputs.nth(2)).toHaveValue('last_name');
});

test(`Sort the identifiers`, async ({ page }) => {
	await page.goto(`${uiURL}settings/identity-resolution`);

	const identifiers = page.locator('.identifiers__identifier');
	await identifiers.nth(0).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(0).locator('.identifiers__mapping-down').click();
	await identifiers.nth(2).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(2).locator('.identifiers__mapping-up').click();
	await page.click('.identifiers__save-button');
	await page.reload();
	const identInputs = page.locator('.identifiers__identifier sl-input >> input');
	await expect(identInputs.nth(0)).toHaveValue('first_name');
	await expect(identInputs.nth(1)).toHaveValue('last_name');
	await expect(identInputs.nth(2)).toHaveValue('email');
});

test(`Remove the identifiers`, async ({ page }) => {
	await page.goto(`${uiURL}settings/identity-resolution`);

	let identifiers = page.locator('.identifiers__identifier');

	await identifiers.nth(2).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(2).locator('.identifiers__mapping-remove').click();

	await identifiers.nth(1).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(1).locator('.identifiers__mapping-remove').click();

	await identifiers.nth(0).locator('.identifiers__identifier-menu').click();
	await identifiers.nth(0).locator('.identifiers__mapping-remove').click();

	await page.click('.identifiers__save-button');
	await page.reload();

	expect(await page.locator('.identifiers__identifier').count()).toBe(0);
});
