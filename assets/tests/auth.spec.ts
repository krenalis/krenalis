import { test, expect } from '@playwright/test';
import { login, logout, adminURL } from './utils';

const LOGIN_BUTTON_CLASS = '.login__button';
const LOGOUT_BUTTON_CLASS = '.sidebar__item-text-logout';

test('Passwordless login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await expect(page.locator('#central-logo')).toBeVisible();
});

test('Update the member email to disable passwordless login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await page.click('.header__passwordless-tooltip-body > a');
	await page.click('.members__member-edit')
	await page.getByRole('textbox', { name: 'email' }).fill('test@open2b.com');
	await page.click('.member__save-button')
	await page.waitForTimeout(2000); // Add a timeout to ensure that the saving was completed.
	await logout(page);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
});

test('Try to access a page that requires authentication and check that it redirects to the login page', async ({
	page,
}) => {
	await page.goto(`${adminURL}/`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${adminURL}/connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${adminURL}/users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
});

test('Login', async ({ page }) => {
	await page.goto(`${adminURL}/`);
	await page.getByRole('textbox', { name: 'email' }).fill('test@open2b.com');
	await page.getByRole('textbox', { name: 'password' }).fill('foopass2');
	await page.click('sl-button');
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeVisible();
	await logout(page);
});

test('Logout', async ({ page }) => {
	await login(page);
	await page.goto(`${adminURL}/`);
	await expect(page.locator(LOGOUT_BUTTON_CLASS)).toBeVisible();
	await page.click(LOGOUT_BUTTON_CLASS);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${adminURL}/connections`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
	await page.goto(`${adminURL}/users`);
	await expect(page.locator(LOGIN_BUTTON_CLASS)).toBeVisible();
});
