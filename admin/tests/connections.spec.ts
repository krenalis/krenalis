import { test, expect } from '@playwright/test';
import { login, logout, config, adminURL } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Add Dummy source`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="dummy"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-settings__save-button');
	await expect(page.locator('.connection-wrapper__name')).toContainText('Dummy');
	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];
	await page.goto(`${adminURL}/connections/sources`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'Dummy' }),
	).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'Dummy' }),
	).toBeAttached();
});

test(`Add Dummy destination`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Destination`);
	await page.click(`[data-code="dummy"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-settings__save-button');
	await expect(page.locator('.connection-wrapper__name')).toContainText('Dummy');
	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];
	await page.goto(`${adminURL}/connections/destinations`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'Dummy' }),
	).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'Dummy' }),
	).toBeAttached();
});

test(`Add PostgreSQL source`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="postgresql"]`);
	await page.click('.connectors-list__documentation-add');

	await page.locator('sl-input >> input[name="host"]').fill(config.dbHost);
	await page.locator('sl-input >> input[name="port"]').fill(String(config.dbPort));
	await page.locator('sl-input >> input[name="username"]').fill(config.dbUsername);
	await page.locator('sl-input >> input[name="password"]').fill(config.dbPassword);
	await page.locator('sl-input >> input[name="database"]').fill(config.dbName);

	await page.click('.feedback-button');
	await expect(page.locator('.feedback-button.feedback-button--confirm')).toBeAttached();

	await page.click('.connector-settings__save-button');

	await expect(page.locator('.connection-wrapper__name')).toContainText('PostgreSQL');

	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];

	await page.goto(`${adminURL}/connections/sources`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'PostgreSQL' }),
	).toBeAttached();

	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'PostgreSQL' }),
	).toBeAttached();
});

test(`Add PostgreSQL destination`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Destination`);
	await page.click(`[data-code="postgresql"]`);
	await page.click('.connectors-list__documentation-add');

	await page.locator('sl-input >> input[name="host"]').fill(config.dbHost);
	await page.locator('sl-input >> input[name="port"]').fill(String(config.dbPort));
	await page.locator('sl-input >> input[name="username"]').fill(config.dbUsername);
	await page.locator('sl-input >> input[name="password"]').fill(config.dbPassword);
	await page.locator('sl-input >> input[name="database"]').fill(config.dbName);

	await page.click('.feedback-button');
	await expect(page.locator('.feedback-button.feedback-button--confirm')).toBeAttached();

	await page.click('.connector-settings__save-button');

	await expect(page.locator('.connection-wrapper__name')).toContainText('PostgreSQL');

	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];

	await page.goto(`${adminURL}/connections/destinations`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'PostgreSQL' }),
	).toBeAttached();

	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'PostgreSQL' }),
	).toBeAttached();
});

test(`Add File System source`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="filesystem"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-settings__save-button');
	await expect(page.locator('.connection-wrapper__name')).toContainText('File System');
	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];
	await page.goto(`${adminURL}/connections/sources`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'File System' }),
	).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'File System' }),
	).toBeAttached();
});

test(`Add File System destination`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Destination`);
	await page.click(`[data-code="filesystem"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-settings__save-button');
	await expect(page.locator('.connection-wrapper__name')).toContainText('File System');
	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];
	await page.goto(`${adminURL}/connections/destinations`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'File System' }),
	).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'File System' }),
	).toBeAttached();
});

test(`Add JavaScript source`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="javascript"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-settings__save-button');
	await expect(page.locator('.connection-wrapper__name')).toContainText('JavaScript');
	const url = page.url();
	const fragments = url.split('/');
	const id = fragments[fragments.length - 2];
	await page.goto(`${adminURL}/connections/sources`);
	await expect(
		page.locator(`.grid__row[data-id="${id}"] .connections-list__name-cell`, { hasText: 'JavaScript' }),
	).toBeAttached();
	await page.goto(`${adminURL}/connections`);
	await expect(
		page.locator(`.connection-block[id="${id}"] .connection-block__name`, { hasText: 'JavaScript' }),
	).toBeAttached();
});
