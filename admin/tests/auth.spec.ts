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
	await page.click('.header__account-menu-item--profile'); // Account menu is automatically opened after passwordless login.
	await page.getByRole('textbox', { name: 'email' }).fill('test@krenalis.com');
	await page.click('.member__save-button');
	await expect(page).toHaveURL(`${adminURL}/organization/members`);
	await expect(
		page.locator('wa-toast-item', { hasText: 'Team member information saved successfully' }),
	).toBeVisible();
	await logout(page);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
});

test('An invalid password reset link shows the expiry message on the login page', async ({ page }) => {
	await page.goto(`${adminURL}/reset-password/expired-token`);
	await expect(page).toHaveURL(`${adminURL}/`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await expect(page.locator('wa-toast-item', { hasText: 'This reset password request is expired' })).toBeVisible();
});

test('Try to access a page that requires authentication and check that it redirects to the login page', async ({
	page,
}) => {
	await page.goto(`${adminURL}/`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
	await page.goto(`${adminURL}/users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeAttached();
});

test('Login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await page.getByRole('textbox', { name: 'email' }).fill('test@krenalis.com');
	await page.getByRole('textbox', { name: 'password' }).fill('krenalis-password');
	await page.click('wa-button');
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeAttached();
	await logout(page);
});

test('Logout', async ({ page }) => {
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
