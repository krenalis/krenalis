import { test, expect } from '@playwright/test';
import { login, logout, adminURL } from './utils';

const LOGIN_BUTTON_CLASS = '.login__button';
const LOGOUT_BUTTON_CLASS = '#logout-button';

test('Passwordless login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await expect(page.locator('#central-logo')).toBeAttached();
});

test('Update the member email to disable passwordless login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await page.click('.header__passwordless-create-account');
	await page.click('.members__member-edit');
	await page.getByRole('textbox', { name: 'email' }).fill('test@meergo.com');
	// await page.locator('sl-input >> input[name="email"]').fill('test@meergo.com'); // TEST: Try another way to access the input
	// await page.waitForTimeout(2000); // TEST: try with a timeout to ensure React state is updated before saving
	// TEST: use an alternative way to fill the input (ex. focus the input and then -> keyboard.type('test@meergo.com'))
	// TEST: click outside the input (if for some reason `onSlInput` on the OS behaves like `onSlChange`)
	await page.click('.member__save-button');
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await logout(page);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
});

test.skip('Try to access a page that requires authentication and check that it redirects to the login page', async ({
	page,
}) => {
	await page.goto(`${adminURL}/`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
});

test.skip('Login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await page.getByRole('textbox', { name: 'email' }).fill('test@meergo.com');
	await page.getByRole('textbox', { name: 'password' }).fill('meergo-password');
	await page.click('sl-button');
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeAttached();
	await logout(page);
});

test.skip('Logout', async ({ page }) => {
	await login(page);
	await page.goto(`${adminURL}/`);
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeAttached();
	await page.waitForTimeout(2000); // Add a timeout to ensure that the page is fully loaded.
	await page.click('.header__account-avatar');
	await page.click(LOGOUT_BUTTON_CLASS);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
});
