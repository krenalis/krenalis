import { test, expect, type Route } from '@playwright/test';
import { login, logout, config, adminURL, addDummySource } from './utils';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test('A failed refresh preserves unsaved connection settings', async ({ page }) => {
	const id = await addDummySource(page);
	await page.goto(`${adminURL}/connections/${id}/settings`);
	const name = page.getByRole('textbox', { name: 'Name', exact: true });
	await name.fill('Saved name');

	const refresh = Promise.withResolvers<Route>();
	await page.route(`**/v1/connections/${id}/pipeline-types`, (route) => refresh.resolve(route), { times: 1 });
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	const request = await refresh.promise;
	await name.fill('Unsaved name');
	await request.fulfill({ status: 500 });

	await expect(page.locator('wa-toast-item')).toBeVisible();
	await expect(name).toHaveValue('Unsaved name');
	await expect(page.getByRole('button', { name: 'Retry' })).toHaveCount(0);
});

test('Retry cancels a connection error that has not been shown yet', async ({ page }) => {
	const id = await addDummySource(page);
	await page.clock.install({ time: new Date('2026-01-01T00:00:00Z') });
	const firstLoad = Promise.withResolvers<Route>();
	await page.route(`**/v1/connections/${id}/pipeline-types`, (route) => firstLoad.resolve(route), { times: 1 });
	await page.goto(`${adminURL}/connections/${id}/settings`);
	const request = await firstLoad.promise;
	await page.clock.pauseAt(new Date('2026-01-01T00:01:00Z'));
	await request.fulfill({ status: 500 });
	await page.getByRole('button', { name: 'Retry' }).click();
	await page.clock.resume();
	await expect(page.getByRole('textbox', { name: 'Name', exact: true })).toBeVisible();
	await expect(page.locator('wa-toast-item')).toHaveCount(0);
});

test(`Add Dummy source`, async ({ page }) => {
	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="dummy"]`);
	await page.click('.connectors-list__documentation-add');
	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');
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
	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');
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

	await page.locator('wa-input >> input[name="host"]').fill(config.dbHost);
	await page.locator('wa-input >> input[name="port"]').fill(String(config.dbPort));
	await page.locator('wa-input >> input[name="username"]').fill(config.dbUsername);
	await page.locator('wa-input >> input[name="password"]').fill(config.dbPassword);
	await page.locator('wa-input >> input[name="database"]').fill(config.dbName);

	await page.click('.connector-ui__buttons .feedback-button >> button[name="test"]');
	await expect(page.locator('.feedback-button.feedback-button--confirm')).toBeAttached();

	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');

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

	await page.locator('wa-input >> input[name="host"]').fill(config.dbHost);
	await page.locator('wa-input >> input[name="port"]').fill(String(config.dbPort));
	await page.locator('wa-input >> input[name="username"]').fill(config.dbUsername);
	await page.locator('wa-input >> input[name="password"]').fill(config.dbPassword);
	await page.locator('wa-input >> input[name="database"]').fill(config.dbName);

	await page.click('.connector-ui__buttons .feedback-button >> button[name="test"]');
	await expect(page.locator('.feedback-button.feedback-button--confirm')).toBeAttached();

	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');

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
	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');
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
	await page.click('.connector-ui__buttons .feedback-button >> button[name="save"]');
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
