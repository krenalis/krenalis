import { test, expect } from '@playwright/test';
import { adminURL } from './utils';

test('Test fill on native input', async ({ page }) => {
	await page.goto(`${adminURL}`);

	await page.waitForTimeout(2000); // Add a timeout to ensure the passwordless login is completed.
	await page.goto(`${adminURL}/test-fill`);

	// const nativeInput = page.locator('input[name="native-input"]');

	await page.locator('input[name="native-input"]').fill('hello');
	await page.waitForTimeout(5000); // Add a timeout to ensure the saving is completed
	await expect(page.locator('input[name="native-input"]')).toHaveValue('hello');

	await page.locator('input[name="native-input"]').fill('hello2');
	await page.waitForTimeout(5000); // Add a timeout to ensure the saving is completed
	await expect(page.locator('input[name="native-input"]')).toHaveValue('hello2');
});

test('Test fill on Shoelace input', async ({ page }) => {
	await page.goto(`${adminURL}`);

	await page.waitForTimeout(2000); // Add a timeout to ensure the passwordless login is completed.
	await page.goto(`${adminURL}/test-fill`);

	// const slInput = page.locator('sl-input >> input[name="shoelace-input"]');

	await page.locator('sl-input >> input[name="shoelace-input"]').fill('hello');
	await page.waitForTimeout(5000); // Add a timeout to ensure the saving is completed
	await expect(page.locator('sl-input >> input[name="shoelace-input"]')).toHaveValue('hello');

	await page.locator('sl-input >> input[name="shoelace-input"]').fill('hello2');
	await page.waitForTimeout(5000); // Add a timeout to ensure the saving is completed
	await expect(page.locator('sl-input >> input[name="shoelace-input"]')).toHaveValue('hello2');
});
