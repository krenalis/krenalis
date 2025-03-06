import { test, expect } from '@playwright/test';
import { config, login, logout, uiURL } from './utils';

const LOGIN_BUTTON_CLASS = '.login__button';
const LOGOUT_BUTTON_CLASS = '.sidebar__item-text-logout';

test('Try to access a page that requires authentication and check that it redirects to the login page', async ({
	page,
}) => {
	await page.goto(`${uiURL}`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${uiURL}connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${uiURL}users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
});

test('Login', async ({ page }) => {
	await page.goto(`${uiURL}`);
	await page.getByRole('textbox', { name: 'email' }).fill('acme@open2b.com');
	await page.getByRole('textbox', { name: 'password' }).fill('foopass2');
	await page.click('sl-button');
	try {
		await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeVisible();
	} catch {
		// The user must first select a workspace, because the
		// organization has more than one.
		const workspaceList = page.locator('.workspace-list__workspaces');
		const firstWorkspaceTile = workspaceList
			.locator(`.workspace-list__workspace[data-id="${String(config.workspaceID)}"]`)
			.nth(0);
		await firstWorkspaceTile.click();
		await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeVisible();
	}
	await logout(page);
});

test('Logout', async ({ page }) => {
	await login(page);
	await page.goto(`${uiURL}`);
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeVisible();
	await page.click(LOGOUT_BUTTON_CLASS);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${uiURL}connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${uiURL}users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
});
