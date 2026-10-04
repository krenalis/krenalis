import { test, expect } from '@playwright/test';
import { adminURL, config } from './utils';
import { WORKSPACE_ID_KEY } from '../src/constants/storage';

const account = {
	id: 'TESTACCOUNT01',
	workspace: config.workspaceID,
	name: 'Test account',
	status: 'Available',
	userCount: 10,
	generatedRecordCount: 10,
	sourceConnection: null,
	destinationConnection: null,
	duplicateRecordPercent: 2,
	countries: { IT: 1 },
	createdAt: '2026-01-01T00:00:00Z',
	updatedAt: '2026-01-01T00:00:00Z',
};

test.beforeEach(async ({ page }) => {
	// The Admin fixture has a Production workspace. Expose its Development UI only within these controlled API tests.
	await page.route('**/v1/workspaces', async (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		const response = await route.fetch();
		if (!response.ok()) return route.fulfill({ response });
		const body = await response.json();
		body.workspaces = body.workspaces.map((workspace) =>
			workspace.id === config.workspaceID ? { ...workspace, environment: 'development' } : workspace,
		);
		await route.fulfill({ response, json: body });
	});
	const response = await page.request.post(`${config.baseURL}/v1/members/login`, {
		data: { email: 'test@krenalis.com', password: 'krenalis-password' },
	});
	expect(response.ok()).toBe(true);
	await page.addInitScript(({ workspace, key }) => localStorage.setItem(key, workspace), {
		workspace: config.workspaceID,
		key: WORKSPACE_ID_KEY,
	});
});

test.afterEach(async ({ page }) => {
	await page.request.post(`${config.baseURL}/v1/members/logout`);
});

test('Development links to Simulated world', async ({ page }) => {
	await page.goto(`${adminURL}/development`);
	await expect(page.getByRole('link', { name: /^Simulated world Create simulated applications/ })).toBeVisible();
	await expect(page.getByText('Create simulated applications for development, testing and demos.')).toBeVisible();
	await expect(page.getByText('simulated connector accounts', { exact: false })).toHaveCount(0);
});

test('a deleting account remains listed until cleanup finishes', async ({ page }) => {
	let reads = 0;
	await page.route('**/v1/simulated/systems', (route) => {
		reads += 1;
		return route.fulfill({
			json: { simulatedSystems: reads === 1 ? [{ ...account, status: 'Deleting' }] : [] },
		});
	});
	await page.goto(`${adminURL}/simulated-world`);
	await expect(page.getByText(account.name)).toBeVisible();
	await expect(page.locator('.simulated-systems__status-badge--deleting')).toHaveText('Deleting');
	await expect(page.getByText('No simulated systems yet')).toBeVisible({ timeout: 10000 });
});

test('account detail returns to the list after deletion finishes', async ({ page }) => {
	let reads = 0;
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => {
		reads += 1;
		return reads === 1
			? route.fulfill({ json: { ...account, status: 'Deleting' } })
			: route.fulfill({ status: 404, json: { error: { message: 'Account not found' } } });
	});
	await page.route('**/v1/simulated/systems', (route) => route.fulfill({ json: { simulatedSystems: [] } }));
	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await expect(page.getByText('Deleting simulated system and records')).toBeVisible();
	await expect(page).toHaveURL(`${adminURL}/simulated-world`, { timeout: 10000 });
});

test('systems grid shows initial records and live connection activity', async ({ page }) => {
	const sourceID = 'SOURCECON001';
	const destinationID = 'DESTCONN0001';
	let connectionReads = 0;
	let isSourceRunning = false;
	let isDestinationRunning = false;
	await page.route('**/v1/connections', (route) => {
		connectionReads += 1;
		return route.fulfill({
			json: {
				connections:
					connectionReads === 1
						? []
						: [
								{ id: sourceID, pipelines: [{ running: isSourceRunning }] },
								{ id: destinationID, pipelines: [{ running: isDestinationRunning }] },
							],
			},
		});
	});
	await page.route('**/v1/simulated/systems', (route) =>
		route.fulfill({
			json: {
				simulatedSystems: [
					{
						...account,
						name: 'CRM Demo',
						userCount: 15000,
						sourceConnection: sourceID,
						destinationConnection: destinationID,
					},
					{ ...account, id: 'TESTACCOUNT02', name: 'No connections' },
					{ ...account, id: 'TESTACCOUNT03', name: 'Source only', sourceConnection: 'ONLYSOURCE001' },
					{
						...account,
						id: 'TESTACCOUNT04',
						name: 'Destination only',
						destinationConnection: 'ONLYDEST0001',
					},
				],
			},
		}),
	);

	await page.goto(`${adminURL}/simulated-world`);
	await expect(page.locator('.simulated-systems__list-content .grid__header-cell')).toHaveText([
		'Name',
		'Type',
		'Records',
		'Linked connections',
	]);
	const connected = page.locator('.simulated-systems__list-content .grid__row[data-id="TESTACCOUNT01"]');
	await expect(connected.locator('.grid__cell').nth(1)).toHaveText('SaaS application');
	await expect(connected.locator('.grid__cell').nth(2)).toHaveText('15,000');
	await expect(connected.locator('.simulated-systems__status-badge')).toHaveCount(0);
	await expect(connected.getByRole('link', { name: 'Source' })).toHaveAttribute(
		'href',
		`/admin/connections/${sourceID}/pipelines`,
	);
	await expect(connected.getByRole('link', { name: 'Destination' })).toHaveAttribute(
		'href',
		`/admin/connections/${destinationID}/pipelines`,
	);
	await expect(connected.getByRole('link', { name: 'Source' }).locator('sl-icon')).toHaveAttribute(
		'name',
		'arrow-down-right-square',
	);
	await expect(connected.getByRole('link', { name: 'Destination' }).locator('sl-icon')).toHaveAttribute(
		'name',
		'arrow-up-right-square',
	);
	await expect(connected.locator('.simulated-systems__connection-links')).toHaveText(/Source\s*Destination/);
	const unconnected = page.locator('.simulated-systems__list-content .grid__row[data-id="TESTACCOUNT02"]');
	await expect(unconnected.locator('.grid__cell').nth(3)).toHaveText('—');
	const sourceOnly = page.locator('.simulated-systems__list-content .grid__row[data-id="TESTACCOUNT03"]');
	await expect(sourceOnly.locator('.simulated-systems__connection-links')).toHaveText('Source');
	await expect(sourceOnly.getByRole('link', { name: 'Source', exact: true })).toHaveAttribute(
		'href',
		'/admin/connections/ONLYSOURCE001/pipelines',
	);
	const destinationOnly = page.locator('.simulated-systems__list-content .grid__row[data-id="TESTACCOUNT04"]');
	await expect(destinationOnly.locator('.simulated-systems__connection-links')).toHaveText('Destination');
	await expect(destinationOnly.getByRole('link', { name: 'Destination', exact: true })).toHaveAttribute(
		'href',
		'/admin/connections/ONLYDEST0001/pipelines',
	);
	await expect.poll(() => connectionReads).toBeGreaterThan(1);
	isSourceRunning = true;
	await expect(connected.getByRole('link', { name: 'Source' }).locator('.simulated-systems__shimmer')).toHaveCount(
		1,
		{
			timeout: 7000,
		},
	);
	isSourceRunning = false;
	isDestinationRunning = true;
	await expect(
		connected.getByRole('link', { name: 'Destination' }).locator('.simulated-systems__shimmer'),
	).toHaveCount(1, {
		timeout: 7000,
	});
	await expect(connected.getByRole('link', { name: 'Source' }).locator('.simulated-systems__shimmer')).toHaveCount(
		0,
	);
	isDestinationRunning = false;
	await expect(connected.locator('.simulated-systems__connection-links .simulated-systems__shimmer')).toHaveCount(
		0,
		{
			timeout: 7000,
		},
	);
});

for (const view of ['list', 'detail'] as const) {
	test(`Available ${view} refreshes external connection associations without remounting`, async ({ page }) => {
		await page.clock.install();
		let sourceConnection: string | null = null;
		let destinationConnection: string | null = null;
		let reads = 0;
		const endpoint = view === 'list' ? '**/v1/simulated/systems' : `**/v1/simulated/systems/${account.id}`;
		await page.route(endpoint, (route) => {
			reads += 1;
			const current = { ...account, sourceConnection, destinationConnection };
			return route.fulfill({ json: view === 'list' ? { simulatedSystems: [current] } : current });
		});
		await page.route('**/v1/connections', (route) => route.fulfill({ json: { connections: [] } }));
		await page.goto(`${adminURL}/simulated-world${view === 'detail' ? `/${account.id}` : ''}`);
		const content = page.locator(
			view === 'list' ? '.simulated-systems__list-content' : '.simulated-systems--detail',
		);
		await expect(content).toContainText(account.name);
		await expect(
			content.getByRole('link', { name: view === 'list' ? 'Source' : 'View connection', exact: true }),
		).toHaveCount(0);
		const initialReads = reads;

		sourceConnection = 'EXTERNALSOURCE';
		destinationConnection = 'EXTERNALDEST';
		await page.clock.runFor(15000);
		await expect(content.locator('a[href="/admin/connections/EXTERNALSOURCE/pipelines"]')).toBeVisible();
		await expect(content.locator('a[href="/admin/connections/EXTERNALDEST/pipelines"]')).toBeVisible();
		expect(reads).toBe(initialReads + 1);

		sourceConnection = null;
		destinationConnection = null;
		await page.clock.runFor(15000);
		await expect(content.locator('a[href*="/connections/"]')).toHaveCount(0);
		expect(reads).toBe(initialReads + 2);
		await expect(page).toHaveURL(`${adminURL}/simulated-world${view === 'detail' ? `/${account.id}` : ''}`);
	});
}

test('OK shows a newly created system in the list', async ({ page }) => {
	let isCreated = false;
	const newAccount = { ...account, name: 'New system', userCount: 0, generatedRecordCount: 0 };
	await page.route('**/v1/simulated/systems', (route) => {
		if (route.request().method() === 'GET')
			return route.fulfill({ json: { simulatedSystems: isCreated ? [newAccount] : [] } });
		isCreated = true;
		return route.fulfill({ json: { id: account.id } });
	});
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => route.fulfill({ json: newAccount }));

	await page.goto(`${adminURL}/simulated-world`);
	await expect(page.getByText('No simulated systems yet', { exact: false })).toBeVisible();
	await page.getByRole('link', { name: 'Create simulated system' }).click();
	await page.getByLabel('Name', { exact: true }).fill('New system');
	await page.locator('sl-select#simulated-system-user-count').click();
	await page.locator('sl-select#simulated-system-user-count sl-option[value="0"]').click();
	await page.getByLabel('Source connection').uncheck();
	await page.getByLabel('Destination connection').uncheck();
	await page.getByRole('button', { name: 'Create', exact: true }).click();
	await page.getByRole('button', { name: 'OK' }).click();
	await expect(page.locator('.simulated-systems__list-content .grid__row[data-id]')).toContainText('New system');
});

test('custom validation and zero records submit no hidden configuration', async ({ page }) => {
	let submitted;
	const emptyAccount = {
		...account,
		name: 'Empty test',
		userCount: 0,
		generatedRecordCount: 0,
		duplicateRecordPercent: 0,
		countries: {},
	};
	await page.route('**/v1/simulated/systems', async (route) => {
		if (route.request().method() === 'GET')
			return route.fulfill({ json: { simulatedSystems: submitted == null ? [] : [emptyAccount] } });
		submitted = route.request().postDataJSON();
		await route.fulfill({ json: { id: account.id } });
	});
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => route.fulfill({ json: emptyAccount }));

	await page.goto(`${adminURL}/simulated-world`);
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.locator('.simulated-systems > .section--annotated')).toHaveCount(1);
	await expect(page.locator('.simulated-systems__list-content .grid')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Create simulated system' })).toHaveCount(1);
	await page.getByRole('link', { name: 'Create simulated system' }).click();
	await expect(page.getByRole('dialog', { name: 'Create simulated system' })).toBeVisible();
	await expect(
		page.locator('sl-dialog.simulated-systems__dialog').getByRole('button', { name: 'Close' }),
	).toHaveCount(0);
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await page.getByLabel('Name', { exact: true }).fill('Empty test');
	await page.locator('sl-select#simulated-system-user-count').click();
	await page.locator('sl-select#simulated-system-user-count sl-option[value="100000"]').click();
	await expect(page.locator('sl-select#simulated-system-user-count sl-option[value="custom"]')).toBeHidden();
	await page.locator('sl-select#simulated-system-user-count').click();
	await page.locator('sl-select#simulated-system-user-count sl-option[value="custom"]').click();
	await expect(page.getByRole('textbox', { name: 'User records to generate', exact: true })).toBeFocused();
	await expect(page.getByRole('textbox', { name: 'User records to generate', exact: true })).toHaveValue('100,000');
	await expect(page.getByLabel('Duplicate records', { exact: true })).toBeVisible();
	await expect(page.getByText('Country distribution', { exact: true })).toBeVisible();
	await page.getByRole('textbox', { name: 'User records to generate', exact: true }).fill('1000');
	await expect(page.locator('sl-select#simulated-system-user-count')).toHaveCount(0);
	await expect(page.getByRole('textbox', { name: 'User records to generate', exact: true })).toHaveValue('1000');
	await page.locator('sl-select#simulated-system-duplicate-percent').click();
	await page.locator('sl-select#simulated-system-duplicate-percent sl-option[value="custom"]').click();
	const duplicateInput = page.locator('sl-input#simulated-system-duplicate-percent');
	await expect(page.locator('sl-select#simulated-system-duplicate-percent')).toHaveCount(0);
	await expect(duplicateInput).toBeFocused();
	await expect(duplicateInput.locator('input')).toHaveValue('2');
	await expect(duplicateInput.locator('[slot="suffix"]')).toHaveText('%');
	await duplicateInput.locator('input').fill('2.001');
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Create', exact: true })
		.click();
	await expect(page.getByRole('alert')).toContainText('at most two decimal places');
	await expect(duplicateInput.locator('input')).toHaveValue('2.001');
	await duplicateInput.locator('input').fill('12.34');
	await page.getByRole('textbox', { name: 'User records to generate', exact: true }).fill('0');
	await expect(duplicateInput).toHaveCount(0);
	await page.getByLabel('Source connection').uncheck();
	await page.getByLabel('Destination connection').uncheck();
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Create', exact: true })
		.click();
	await expect(page.getByRole('heading', { name: 'Simulated system created' })).toBeVisible();
	await expect(page.getByText('Simulated system created successfully')).toBeVisible();
	await expect(page.locator('.simulated-systems__not-requested li')).toHaveText([
		'Simulated users',
		'Source connection',
		'Destination connection',
	]);
	await expect(page.locator('.simulated-systems__step')).toHaveCount(0);
	await expect(page.locator('.simulated-systems__back')).toHaveCount(0);
	await page.getByRole('button', { name: 'OK' }).click();
	await expect(page.locator('.simulated-systems__list-content .grid__row[data-id]')).toContainText('Empty test');
	expect(submitted).toEqual({
		name: 'Empty test',
		userCount: 0,
		duplicateRecordPercent: 0,
		countries: {},
	});
});

test('creation shows user progress and creates both selected connections', async ({ page }) => {
	let accountReads = 0;
	const connections: any[] = [];
	await page.route('**/v1/connectors', (route) =>
		route.fulfill({
			json: {
				connectors: [
					{
						code: 'simulated-application',
						label: 'Simulated Application',
						type: 'Application',
						categories: [],
						asSource: { targets: ['User'], hasSettings: true, sampleQuery: '', summary: '' },
						asDestination: { targets: ['User'], hasSettings: true, sendingMode: 'Server', summary: '' },
						hasSheets: false,
						fileExtension: '',
						oauth: { configured: false, disallow127_0_0_1: false, disallowLocalhost: false },
						terms: { user: 'user', users: 'users', userID: 'id' },
						strategies: false,
					},
				],
			},
		}),
	);
	await page.route('**/v1/simulated/systems', (route) => route.fulfill({ json: { id: account.id } }));
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => {
		accountReads += 1;
		return route.fulfill({
			json: {
				...account,
				status: accountReads === 1 ? 'Preparing' : 'Available',
				generatedRecordCount: accountReads === 1 ? 4 : 10,
			},
		});
	});
	await page.route('**/v1/connections', (route) => {
		if (route.request().method() === 'GET')
			return route.fulfill({
				json: {
					connections: connections.map((connection, index) => ({
						id: `CONNECTION0${index + 1}`,
						name: connection.name,
						connector: connection.connector,
						connectorType: 'Application',
						role: connection.role,
						storage: '',
						compression: '',
						strategy: null,
						sendingMode: connection.sendingMode,
						hasSettings: true,
						health: 'NoRecentData',
						pipelines: [],
					})),
				},
			});
		connections.push(route.request().postDataJSON());
		return route.fulfill({ json: { id: `CONNECTION0${connections.length}` } });
	});

	await page.goto(`${adminURL}/simulated-world/create`);
	await expect(page.getByLabel('Source connection')).toBeChecked();
	await expect(page.getByLabel('Destination connection')).toBeChecked();
	await page.getByLabel('Name', { exact: true }).fill('Test account');
	await page.locator('sl-select#simulated-system-user-count').click();
	await page.locator('sl-select#simulated-system-user-count sl-option[value="10"]').click();
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Create', exact: true })
		.click();
	await expect(page.getByRole('progressbar', { name: 'Simulated users created' })).toHaveAttribute('value', '4');
	await page.keyboard.press('Escape');
	await expect(page.getByRole('dialog', { name: 'Create simulated system' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Simulated system created' })).toBeVisible();
	await expect(page.locator('.simulated-systems__step--done')).toHaveCount(3);
	const usersStep = page.locator('.simulated-systems__step').filter({ hasText: 'Add simulated users' });
	await expect(usersStep).toContainText('10 records');
	await expect(usersStep.getByRole('progressbar')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'OK' })).toBeVisible();
	await page.getByRole('button', { name: 'OK' }).click();
	await page.getByRole('link', { name: 'Sources' }).click();
	await expect(page.locator('.connections-list')).toContainText('Test account');
	await page.getByRole('link', { name: 'Destinations' }).click();
	await expect(page.locator('.connections-list')).toContainText('Test account');
	expect(connections).toEqual([
		{
			name: 'Test account',
			role: 'Source',
			connector: 'simulated-application',
			sendingMode: null,
			settings: { account: account.id },
			linkedConnections: null,
			authToken: '',
		},
		{
			name: 'Test account',
			role: 'Destination',
			connector: 'simulated-application',
			sendingMode: 'Server',
			settings: { account: account.id },
			linkedConnections: null,
			authToken: '',
		},
	]);
});

for (const selectedRole of ['Source', 'Destination'] as const) {
	test(`creation can add only the ${selectedRole.toLowerCase()} connection`, async ({ page }) => {
		const connections: any[] = [];
		await page.route('**/v1/simulated/systems', (route) => route.fulfill({ json: { id: account.id } }));
		await page.route(`**/v1/simulated/systems/${account.id}`, (route) =>
			route.fulfill({ json: { ...account, userCount: 0, generatedRecordCount: 0 } }),
		);
		await page.route('**/v1/connections', (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			connections.push(route.request().postDataJSON());
			return route.fulfill({ json: { id: 'CONNECTION01' } });
		});

		await page.goto(`${adminURL}/simulated-world/create`);
		await page.getByLabel('Name', { exact: true }).fill('Test account');
		await page.locator('sl-select#simulated-system-user-count').click();
		await page.locator('sl-select#simulated-system-user-count sl-option[value="0"]').click();
		await page.getByLabel(selectedRole === 'Source' ? 'Destination connection' : 'Source connection').uncheck();
		await page
			.locator('sl-dialog.simulated-systems__dialog')
			.getByRole('button', { name: 'Create', exact: true })
			.click();
		await expect(page.getByRole('heading', { name: 'Simulated system created' })).toBeVisible();
		await expect(page.locator('.simulated-systems__step--done')).toHaveCount(1);
		await expect(page.locator('.simulated-systems__not-requested li')).toHaveText([
			'Simulated users',
			selectedRole === 'Source' ? 'Destination connection' : 'Source connection',
		]);
		expect(connections).toHaveLength(1);
		expect(connections[0].role).toBe(selectedRole);
	});
}

test('a failed source connection does not prevent the destination connection', async ({ page }) => {
	const roles: string[] = [];
	await page.route('**/v1/simulated/systems', (route) => route.fulfill({ json: { id: account.id } }));
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) =>
		route.fulfill({ json: { ...account, userCount: 0, generatedRecordCount: 0 } }),
	);
	await page.route('**/v1/connections', (route) => {
		if (route.request().method() !== 'POST') return route.continue();
		const role = route.request().postDataJSON().role;
		roles.push(role);
		if (role === 'Source')
			return route.fulfill({
				status: 422,
				json: { error: { code: 'InvalidSettings', message: 'Source connection could not be created' } },
			});
		return route.fulfill({ json: { id: 'CONNECTION01' } });
	});

	await page.goto(`${adminURL}/simulated-world/create`);
	await page.getByLabel('Name', { exact: true }).fill('Test account');
	await page.locator('sl-select#simulated-system-user-count').click();
	await page.locator('sl-select#simulated-system-user-count sl-option[value="0"]').click();
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Create', exact: true })
		.click();
	await expect(page.locator('.simulated-systems__step--failed')).toContainText(
		'Source connection could not be created',
	);
	await expect(page.locator('.simulated-systems__step--done')).toHaveCount(2);
	await expect(page.getByRole('button', { name: 'OK' })).toBeVisible();
	expect(roles).toEqual(['Source', 'Destination']);
});

test('preparation follows backend progress, survives network errors, and reaches terminal states', async ({ page }) => {
	let requests = 0;
	await page.route(`**/v1/simulated/systems/${account.id}`, async (route) => {
		requests += 1;
		if (requests === 2)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Controlled interruption' } },
			});
		await route.fulfill({
			json: {
				...account,
				status: requests >= 4 ? 'Available' : 'Preparing',
			},
		});
	});
	await page.route('**/v1/simulated/systems/TESTFAILED01', (route) =>
		route.fulfill({
			json: {
				...account,
				id: 'TESTFAILED01',
				status: 'Failed',
			},
		}),
	);
	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await expect(page.getByText('Preparation in progress')).toBeVisible();
	await expect(page.getByText('Unable to refresh simulated system: Controlled interruption')).toBeVisible();
	await expect(page.getByText('Preparation complete')).toBeVisible({ timeout: 12000 });
	await expect(page.getByText('Available', { exact: true })).toBeVisible();
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.getByText('S3', { exact: true })).toHaveCount(0);
	await page.goto(`${adminURL}/simulated-world/TESTFAILED01`);
	await expect(page.getByText('Failed', { exact: true })).toBeVisible();
	await expect(page.getByText('Preparation failed.')).toBeVisible();
});

test('an initial detail error offers manual retry', async ({ page }) => {
	let requests = 0;
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => {
		requests += 1;
		if (requests === 1)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Initial interruption' } },
			});
		return route.fulfill({ json: account });
	});
	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await expect(page.getByText('Unable to refresh simulated system: Initial interruption')).toBeVisible();
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.getByRole('heading', { name: 'Test account' })).toBeVisible();
	await expect(page.getByText('Italy', { exact: true })).toBeVisible();
	expect(requests).toBe(2);
});

test('account detail shows read-only sections and links to its connections', async ({ page }) => {
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) =>
		route.fulfill({
			json: { ...account, sourceConnection: 'SOURCE01', destinationConnection: 'DESTINATION01' },
		}),
	);
	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await expect(page.locator('.simulated-systems--detail > .section--annotated')).toHaveCount(4);
	await expect(page.getByText('Initial user records')).toBeVisible();
	await expect(page.getByRole('link', { name: 'View connection' })).toHaveCount(2);
	await expect(page.getByRole('link', { name: 'View connection' }).first()).toHaveAttribute(
		'href',
		/admin\/connections\/SOURCE01\/pipelines/,
	);
});

test('switching workspaces does not show a late Available metadata poll from the previous workspace', async ({
	page,
}) => {
	await page.clock.install();
	let firstReads = 0;
	const secondWorkspaceID = 'TESTWORKSPACE02';
	let firstRequested!: () => void;
	let releaseFirst!: () => void;
	let firstFinished!: () => void;
	const firstRequest = new Promise<void>((resolve) => {
		firstRequested = resolve;
	});
	const firstResponse = new Promise<void>((resolve) => {
		releaseFirst = resolve;
	});
	const firstCompletion = new Promise<void>((resolve) => {
		firstFinished = resolve;
	});

	await page.route('**/v1/**', async (route) => {
		if (route.request().headers()['krenalis-workspace'] !== secondWorkspaceID) return route.fallback();
		const response = await route.fetch({
			headers: { ...route.request().headers(), 'Krenalis-Workspace': config.workspaceID },
		});
		await route.fulfill({ response });
	});
	await page.route('**/v1/workspaces', async (route) => {
		if (route.request().method() !== 'GET') return route.fallback();
		const response = await route.fetch();
		const body = await response.json();
		const workspace = body.workspaces.find((item) => item.id === config.workspaceID);
		body.workspaces = [
			{ ...workspace, environment: 'development' },
			{ ...workspace, id: secondWorkspaceID, name: 'Second Development', environment: 'development' },
		];
		await route.fulfill({ response, json: body });
	});
	await page.route('**/v1/simulated/systems', async (route) => {
		if (route.request().headers()['krenalis-workspace'] === secondWorkspaceID) {
			return route.fulfill({
				json: { simulatedSystems: [{ ...account, workspace: secondWorkspaceID, name: 'Second account' }] },
			});
		}
		firstReads += 1;
		if (firstReads === 1) return route.fulfill({ json: { simulatedSystems: [account] } });
		firstRequested();
		await firstResponse;
		await route.fulfill({ json: { simulatedSystems: [{ ...account, name: 'First account' }] } }).catch(() => {});
		firstFinished();
	});

	await page.goto(`${adminURL}/simulated-world`);
	await expect(page.getByText(account.name, { exact: true })).toBeVisible();
	await page.clock.runFor(15000);
	await firstRequest;
	await page.locator('.workspace-selector__text').click();
	await page.getByText('Second Development', { exact: true }).click();
	await expect(page.locator('.simulated-systems__list-content .grid__row[data-id]')).toContainText('Second account');
	releaseFirst();
	await firstCompletion;
	await expect(page.locator('.simulated-systems__list-content .grid__row[data-id]')).toContainText('Second account');
	await expect(page.getByText('First account')).toHaveCount(0);
});

test('rename and delete retain the account after errors, then update the list', async ({ page }) => {
	let name = account.name;
	let renameRequests = 0;
	let deleteRequests = 0;
	let isDeleted = false;
	await page.route('**/v1/simulated/systems', (route) =>
		route.fulfill({ json: { simulatedSystems: isDeleted ? [] : [{ ...account, name }] } }),
	);
	await page.route(`**/v1/simulated/systems/${account.id}`, async (route) => {
		if (route.request().method() === 'GET')
			return isDeleted
				? route.fulfill({ status: 404, json: { error: { message: 'Account not found' } } })
				: route.fulfill({ json: { ...account, name } });
		if (route.request().method() === 'PUT') {
			renameRequests += 1;
			if (renameRequests === 1)
				return route.fulfill({
					status: 503,
					json: { error: { code: 'Unavailable', message: 'Rename interrupted' } },
				});
			name = route.request().postDataJSON().name;
			return route.fulfill({ json: null });
		}
		deleteRequests += 1;
		if (deleteRequests === 1)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Delete interrupted' } },
			});
		isDeleted = true;
		await route.fulfill({ json: null });
	});
	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await page.getByRole('button', { name: 'Rename simulated system' }).click();
	await page.getByLabel('Simulated system name').fill('Renamed account');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('Rename interrupted')).toBeVisible();
	await expect(page.getByLabel('Simulated system name')).toHaveValue('Renamed account');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('heading', { name: 'Renamed account' })).toBeVisible();
	await page.getByRole('link', { name: '← Simulated world' }).click();
	await expect(page.locator('.simulated-systems__list-content .grid__row[data-id]')).toContainText(
		'Renamed account',
	);
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.getByText('S3', { exact: true })).toHaveCount(0);
	await page.locator('.simulated-systems__list-content .grid__row[data-id]').click();
	await expect(page.getByRole('heading', { name: 'Renamed account' })).toBeVisible();
	await page.getByRole('button', { name: 'Delete simulated system' }).click();
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Delete simulated system' })
		.click();
	await expect(page.getByText('Delete interrupted')).toBeVisible();
	await expect(page.locator('.simulated-systems__step--skipped')).toHaveCount(2);
	await expect(page.getByRole('progressbar')).toHaveCount(0);
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.getByRole('heading', { name: 'Simulated system deleted' })).toBeVisible();
	await page.getByRole('button', { name: 'OK' }).click();
	await expect(page.getByText('No simulated systems yet')).toBeVisible();
	expect(renameRequests).toBe(2);
	expect(deleteRequests).toBe(2);
});

test('deletion removes connections before account records and can retry a failed step', async ({ page }) => {
	const operations: string[] = [];
	let sourceRequests = 0;
	let hasStartedAccountDeletion = false;
	let readsAfterDeletion = 0;
	await page.route(`**/v1/connections/SOURCE01`, (route) => {
		operations.push('source');
		sourceRequests += 1;
		if (sourceRequests === 1)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Source deletion interrupted' } },
			});
		return route.fulfill({ json: null });
	});
	await page.route(`**/v1/connections/DESTINATION01`, (route) => {
		operations.push('destination');
		return route.fulfill({ json: null });
	});
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => {
		if (route.request().method() === 'DELETE') {
			operations.push('account');
			hasStartedAccountDeletion = true;
			return route.fulfill({ json: null });
		}
		if (!hasStartedAccountDeletion)
			return route.fulfill({
				json: { ...account, sourceConnection: 'SOURCE01', destinationConnection: 'DESTINATION01' },
			});
		readsAfterDeletion += 1;
		return readsAfterDeletion === 1
			? route.fulfill({ json: { ...account, status: 'Deleting' } })
			: route.fulfill({ status: 404, json: { error: { message: 'Account not found' } } });
	});
	await page.route('**/v1/simulated/systems', (route) =>
		route.fulfill({ json: { simulatedSystems: hasStartedAccountDeletion ? [] : [account] } }),
	);

	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await page.getByRole('button', { name: 'Delete simulated system' }).click();
	await expect(page.locator('sl-dialog.simulated-systems__dialog')).toContainText(
		'its connections, and its records.',
	);
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Delete simulated system' })
		.click();
	await expect(page.getByText('Source deletion interrupted')).toBeVisible();
	await expect(page.locator('.simulated-systems__step--failed')).toContainText('Delete source connection');
	await expect(page.locator('.simulated-systems__step--skipped')).toHaveCount(0);
	await expect(page.getByRole('progressbar')).toHaveCount(0);
	expect(operations).toEqual(['source']);
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.locator('.simulated-systems__step--done')).toHaveCount(4, { timeout: 10000 });
	await expect(page.getByRole('heading', { name: 'Simulated system deleted' })).toBeVisible();
	expect(operations).toEqual(['source', 'source', 'destination', 'account']);
	await page.getByRole('button', { name: 'OK' }).click();
	await expect(page.getByText('No simulated systems yet')).toBeVisible();
});

test('deletion retry picks up a connection added after the account was opened', async ({ page }) => {
	const operations: string[] = [];
	let accountDeleteRequests = 0;
	await page.route(`**/v1/connections/SOURCE02`, (route) => {
		operations.push('source');
		return route.fulfill({ json: null });
	});
	await page.route(`**/v1/simulated/systems/${account.id}`, (route) => {
		if (route.request().method() === 'GET') {
			if (accountDeleteRequests === 2)
				return route.fulfill({ status: 404, json: { error: { message: 'Account not found' } } });
			return route.fulfill({
				json: { ...account, sourceConnection: accountDeleteRequests === 0 ? null : 'SOURCE02' },
			});
		}
		operations.push('account');
		accountDeleteRequests += 1;
		if (accountDeleteRequests === 1)
			return route.fulfill({
				status: 422,
				json: { error: { code: 'SimulatedSystemInUse', message: 'Account is in use' } },
			});
		return route.fulfill({ json: null });
	});

	await page.goto(`${adminURL}/simulated-world/${account.id}`);
	await page.getByRole('button', { name: 'Delete simulated system' }).click();
	await page
		.locator('sl-dialog.simulated-systems__dialog')
		.getByRole('button', { name: 'Delete simulated system' })
		.click();
	await expect(page.locator('.simulated-systems__step--failed')).toContainText(
		'Delete simulated system records',
	);
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.locator('.simulated-systems__step--done')).toHaveCount(3);
	await expect(page.getByRole('heading', { name: 'Simulated system deleted' })).toBeVisible();
	expect(operations).toEqual(['account', 'source', 'account']);
});
