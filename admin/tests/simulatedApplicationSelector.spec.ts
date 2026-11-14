import { expect, Page, test } from '@playwright/test';
import { adminURL, config } from './utils';
import { WORKSPACE_ID_KEY } from '../src/constants/storage';
import { ConnectionRole } from '../src/lib/api/types/connection';
import { SimulatedSystem } from '../src/lib/api/types/simulatedSystem';

const headers = { 'Krenalis-Workspace': config.simulatedWorkspaceID };

const createSystem = async (page: Page, name: string): Promise<string> => {
	const response = await page.request.post(`${config.baseURL}/v1/simulated/systems`, {
		headers,
		data: { name, userCount: 0 },
	});
	expect(response.ok()).toBe(true);
	return (await response.json()).id;
};

const createConnection = async (page: Page, role: ConnectionRole, account: string): Promise<string> => {
	const response = await page.request.post(`${config.baseURL}/v1/connections`, {
		headers,
		data: {
			name: `Existing ${role}`,
			role,
			connector: 'simulated-application',
			strategy: null,
			sendingMode: role === 'Destination' ? 'Server' : null,
			settings: { account },
			linkedConnections: null,
		},
	});
	expect(response.ok()).toBe(true);
	return (await response.json()).id;
};

const openSelector = async (page: Page, role: ConnectionRole) => {
	await page.goto(`${adminURL}/connectors?role=${role}`);
	await page.locator('[data-code="simulated-application"]').click();
	await page.locator('.connectors-list__documentation-add').click();
	await expect(page.locator('.connector-ui')).toBeVisible();
};

test.beforeEach(async ({ page }) => {
	const response = await page.request.post(`${config.baseURL}/v1/members/login`, {
		data: { email: 'test@krenalis.com', password: 'krenalis-password' },
	});
	expect(response.ok()).toBe(true);
	await page.addInitScript(({ workspace, key }) => localStorage.setItem(key, workspace), {
		workspace: config.simulatedWorkspaceID,
		key: WORKSPACE_ID_KEY,
	});
});

test.afterEach(async ({ page }) => {
	const connections = await page.request.get(`${config.baseURL}/v1/connections`, { headers });
	expect(connections.ok()).toBe(true);
	for (const connection of (await connections.json()).connections) {
		const response = await page.request.delete(`${config.baseURL}/v1/connections/${connection.id}`, { headers });
		expect(response.ok()).toBe(true);
	}
	const systems = await page.request.get(`${config.baseURL}/v1/simulated/systems`, { headers });
	expect(systems.ok()).toBe(true);
	for (const system of (await systems.json()).simulatedSystems as SimulatedSystem[]) {
		if (system.status === 'Failed') continue;
		const response = await page.request.delete(`${config.baseURL}/v1/simulated/systems/${system.id}`, { headers });
		expect(response.ok()).toBe(true);
	}
	await expect
		.poll(
			async () => {
				const response = await page.request.get(`${config.baseURL}/v1/simulated/systems`, { headers });
				expect(response.ok()).toBe(true);
				return (await response.json()).simulatedSystems.filter(
					(system: SimulatedSystem) => system.status !== 'Failed',
				);
			},
			{ timeout: 30000 },
		)
		.toEqual([]);
	await page.request.post(`${config.baseURL}/v1/members/logout`);
});

test('Simulated selector empty state guides creation in Simulated world', async ({ page }) => {
	await openSelector(page, 'Source');
	await expect(page.locator('.connector-ui')).toContainText(
		'There are no available simulated data stores. Create one in the Simulated world section',
	);
	await expect(page.locator('.connector-ui wa-select')).toHaveCount(0);
	for (const obsolete of ['New account name', 'Initial user records', 'Duplicate records', 'Create new account']) {
		await expect(page.getByText(obsolete, { exact: false })).toHaveCount(0);
	}
	await expect(page.locator('.connector-ui button[name="save"]')).toHaveCount(0);
});

test('Simulated selector offers only Available systems and enforces each role independently', async ({ page }) => {
	const free = await createSystem(page, 'Free system');
	const source = await createSystem(page, 'Source occupied');
	const destination = await createSystem(page, 'Destination occupied');
	await createConnection(page, 'Source', source);
	await createConnection(page, 'Destination', destination);
	const response = await page.request.get(`${config.baseURL}/v1/simulated/systems`, { headers });
	const systems: SimulatedSystem[] = (await response.json()).simulatedSystems;
	expect(systems.some((system) => system.status === 'Failed')).toBe(true);

	for (const role of ['Source', 'Destination'] as const) {
		await openSelector(page, role);
		const selector = page.locator('.connector-ui wa-select');
		await expect(selector.locator('wa-option')).toHaveCount(3);
		await expect(selector.locator(`wa-option[value="${free}"]`)).toHaveJSProperty('disabled', false);
		await expect(selector.locator(`wa-option[value="${source}"]`)).toHaveJSProperty('disabled', role === 'Source');
		await expect(selector.locator(`wa-option[value="${destination}"]`)).toHaveJSProperty(
			'disabled',
			role === 'Destination',
		);
		for (const system of systems.filter((item) => item.status !== 'Available')) {
			await expect(selector.locator(`wa-option[value="${system.id}"]`)).toHaveCount(0);
		}
		await selector.click();
		await selector.locator(`wa-option[value="${role === 'Source' ? source : destination}"]`).click({ force: true });
		await expect(selector).toHaveJSProperty('value', null);
		await selector.locator(`wa-option[value="${role === 'Source' ? destination : source}"]`).click();
		await expect(selector).toHaveJSProperty('value', role === 'Source' ? destination : source);
	}
});

test('Simulated selector refresh discovers systems, preserves valid selection and clears occupied selection', async ({
	page,
}) => {
	await openSelector(page, 'Source');
	const account = await createSystem(page, 'Created elsewhere');
	await page.getByRole('button', { name: 'Refresh simulated data stores' }).click();
	const selector = page.locator('.connector-ui wa-select');
	await expect(selector.locator(`wa-option[value="${account}"]`)).toBeAttached();
	await selector.click();
	await selector.locator(`wa-option[value="${account}"]`).click();
	await expect(selector).toHaveJSProperty('value', account);

	await createSystem(page, 'Another system');
	await page.getByRole('button', { name: 'Refresh simulated data stores' }).click();
	await expect(selector.locator('wa-option')).toHaveCount(2);
	await expect(selector).toHaveJSProperty('value', account);
	await createConnection(page, 'Source', account);
	await page.getByRole('button', { name: 'Refresh simulated data stores' }).click();
	await expect(selector.locator(`wa-option[value="${account}"]`)).toHaveJSProperty('disabled', true);
	await expect(selector).toHaveJSProperty('value', null);
});

test('Simulated selector saves exactly the selected binding and saved connection cannot rebind', async ({ page }) => {
	const account = await createSystem(page, 'Selected system');
	await createSystem(page, 'Other system');
	let systemCreates = 0;
	page.on('request', (request) => {
		if (request.method() === 'POST' && new URL(request.url()).pathname === '/v1/simulated/systems')
			systemCreates += 1;
	});
	await openSelector(page, 'Destination');
	const selector = page.locator('.connector-ui wa-select');
	await selector.click();
	await selector.locator(`wa-option[value="${account}"]`).click();
	const saved = page.waitForResponse(
		(response) => response.request().method() === 'POST' && new URL(response.url()).pathname === '/v1/connections',
	);
	await page.locator('.connector-ui button[name="save"]').click();
	const response = await saved;
	expect(response.ok()).toBe(true);
	expect(response.request().postDataJSON().settings).toEqual({ account });
	const { id } = await response.json();
	await expect(page).toHaveURL(new RegExp(`/connections/${id}/pipelines`));
	const detail = await page.request.get(`${config.baseURL}/v1/simulated/systems/${account}`, { headers });
	expect((await detail.json()).destinationConnection).toBe(id);
	expect(systemCreates).toBe(0);

	await page.goto(`${adminURL}/connections/${id}/settings`);
	await page.getByRole('tab', { name: 'Application settings', exact: true }).click();
	const application = page.getByRole('tabpanel', { name: 'Application settings', exact: true });
	await expect(application).toContainText('Selected system');
	await expect(application.locator('wa-select')).toHaveCount(0);
	await expect(application.getByRole('button', { name: 'Save', exact: true })).toHaveCount(0);
	await expect(application.getByRole('button', { name: 'Refresh simulated data stores' })).toHaveCount(0);
	await page.getByRole('tab', { name: 'General', exact: true }).click();
	const general = page.getByRole('tabpanel', { name: 'General', exact: true });
	await expect(general.getByRole('combobox', { name: 'Simulated data store', exact: true })).toHaveCount(0);
	await general.getByRole('textbox', { name: 'Name', exact: true }).fill('Renamed connection');
	const updated = page.waitForResponse(
		(response) =>
			response.request().method() === 'PUT' && new URL(response.url()).pathname === `/v1/connections/${id}`,
	);
	await general.getByRole('button', { name: 'Save', exact: true }).click();
	const update = await updated;
	expect(update.ok()).toBe(true);
	expect(update.request().postDataJSON()).toEqual({
		name: 'Renamed connection',
		strategy: null,
		sendingMode: 'Server',
	});
	const bound = await page.request.get(`${config.baseURL}/v1/simulated/systems/${account}`, { headers });
	expect((await bound.json()).destinationConnection).toBe(id);
});
