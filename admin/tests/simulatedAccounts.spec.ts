import { test, expect } from '@playwright/test';
import { adminURL, config } from './utils';
import { WORKSPACE_ID_KEY } from '../src/constants/storage';

const account = {
	id: 'TESTACCOUNT01',
	workspace: config.workspaceID,
	name: 'Test account',
	status: 'Ready',
	userCount: 10,
	duplicateRecordPercent: 2,
	countries: { IT: 1 },
	generatedRecordCount: 10,
	generationError: '',
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
		data: { email: 'acme@krenalis.com', password: 'krenalis-password' },
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

test('custom validation and zero records submit no hidden configuration', async ({ page }) => {
	let submitted;
	await page.route('**/v1/simulated-accounts', async (route) => {
		if (route.request().method() === 'GET') return route.fulfill({ json: { simulatedAccounts: [] } });
		submitted = route.request().postDataJSON();
		await route.fulfill({ json: { id: account.id } });
	});
	await page.route(`**/v1/simulated-accounts/${account.id}`, (route) =>
		route.fulfill({
			json: {
				...account,
				name: 'Empty test',
				userCount: 0,
				duplicateRecordPercent: 0,
				countries: {},
				generatedRecordCount: 0,
			},
		}),
	);

	await page.goto(`${adminURL}/simulated-accounts`);
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Create simulated account' })).toHaveCount(2);
	await page.getByRole('link', { name: 'Create simulated account' }).first().click();
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await page.getByLabel('Account name').fill('Empty test');
	await page.getByLabel('Initial user records').selectOption('custom');
	await expect(page.getByLabel('Custom initial user records')).toBeFocused();
	await page.getByLabel('Custom initial user records').fill('1000');
	await expect(page.getByLabel('Custom initial user records')).toBeVisible();
	await page.getByLabel('Duplicate records').selectOption('custom');
	await expect(page.getByLabel('Custom duplicate records')).toBeFocused();
	await page.getByLabel('Custom duplicate records').fill('2.001');
	await page.getByRole('button', { name: 'Create simulated account' }).click();
	await expect(page.getByRole('alert')).toContainText('at most two decimal places');
	await expect(page.getByLabel('Custom duplicate records')).toHaveValue('2.001');
	await page.getByLabel('Custom duplicate records').fill('12.34');
	await page.getByLabel('Custom initial user records').fill('0');
	await expect(page.getByLabel('Custom duplicate records')).toHaveCount(0);
	await page.getByRole('button', { name: 'Create simulated account' }).click();
	await expect(page.getByRole('heading', { name: 'Empty test' })).toBeVisible();
	expect(submitted).toEqual({
		name: 'Empty test',
		userCount: 0,
		duplicateRecordPercent: 0,
		countries: {},
	});
});

test('preparation follows backend progress, survives network errors, and reaches terminal states', async ({ page }) => {
	let requests = 0;
	await page.route(`**/v1/simulated-accounts/${account.id}`, async (route) => {
		requests += 1;
		if (requests === 2)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Controlled interruption' } },
			});
		await route.fulfill({
			json: {
				...account,
				status: requests >= 4 ? 'Ready' : 'Preparing',
				generatedRecordCount: requests >= 3 ? 10 : 2,
			},
		});
	});
	await page.route('**/v1/simulated-accounts/TESTFAILED01', (route) =>
		route.fulfill({
			json: {
				...account,
				id: 'TESTFAILED01',
				status: 'Failed',
				generatedRecordCount: 2,
				generationError: 'Photo catalog could not be read.',
			},
		}),
	);
	await page.goto(`${adminURL}/simulated-accounts/${account.id}`);
	await expect(page.getByText('2 of 10 initial user records · 20%')).toBeVisible();
	await expect(page.getByText('Unable to refresh account: Controlled interruption')).toBeVisible();
	await expect(page.getByText('2 of 10 initial user records · 20%')).toBeVisible();
	await expect(page.getByText('Completing preparation…')).toBeVisible();
	await expect(page.getByText('Initial preparation complete')).toBeVisible();
	await expect(page.getByText('Ready', { exact: true })).toBeVisible();
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.getByText('S3', { exact: true })).toHaveCount(0);
	await page.goto(`${adminURL}/simulated-accounts/TESTFAILED01`);
	await expect(page.getByText('Failed', { exact: true })).toBeVisible();
	await expect(page.getByText('Photo catalog could not be read.')).toBeVisible();
});

test('an initial detail error offers manual retry', async ({ page }) => {
	let requests = 0;
	await page.route(`**/v1/simulated-accounts/${account.id}`, (route) => {
		requests += 1;
		if (requests === 1)
			return route.fulfill({
				status: 503,
				json: { error: { code: 'Unavailable', message: 'Initial interruption' } },
			});
		return route.fulfill({ json: account });
	});
	await page.goto(`${adminURL}/simulated-accounts/${account.id}`);
	await expect(page.getByText('Unable to refresh account: Initial interruption')).toBeVisible();
	await page.getByRole('button', { name: 'Retry' }).click();
	await expect(page.getByRole('heading', { name: 'Test account' })).toBeVisible();
	await expect(page.getByText('Italy', { exact: true })).toBeVisible();
	expect(requests).toBe(2);
});

test('switching workspaces does not show a late account list from the previous workspace', async ({ page }) => {
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
	await page.route('**/v1/simulated-accounts', async (route) => {
		if (route.request().headers()['krenalis-workspace'] === secondWorkspaceID) {
			return route.fulfill({
				json: { simulatedAccounts: [{ ...account, workspace: secondWorkspaceID, name: 'Second account' }] },
			});
		}
		firstRequested();
		await firstResponse;
		await route.fulfill({ json: { simulatedAccounts: [{ ...account, name: 'First account' }] } }).catch(() => {});
		firstFinished();
	});

	await page.goto(`${adminURL}/simulated-accounts`);
	await firstRequest;
	await page.locator('.workspace-selector__text').click();
	await page.getByText('Second Development', { exact: true }).click();
	await expect(page.locator('.simulated-accounts__row')).toContainText('Second account');
	releaseFirst();
	await firstCompletion;
	await page.waitForTimeout(100);
	await expect(page.locator('.simulated-accounts__row')).toContainText('Second account');
	await expect(page.getByText('First account')).toHaveCount(0);
});

test('rename and delete retain the account after errors, then update the list', async ({ page }) => {
	let name = account.name;
	let renameRequests = 0;
	let deleteRequests = 0;
	let isDeleted = false;
	await page.route('**/v1/simulated-accounts', (route) =>
		route.fulfill({ json: { simulatedAccounts: isDeleted ? [] : [{ ...account, name }] } }),
	);
	await page.route(`**/v1/simulated-accounts/${account.id}`, async (route) => {
		if (route.request().method() === 'GET') return route.fulfill({ json: { ...account, name } });
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
	await page.goto(`${adminURL}/simulated-accounts/${account.id}`);
	await page.getByRole('button', { name: 'Rename account' }).click();
	await page.getByLabel('Account name').fill('Renamed account');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('Rename interrupted')).toBeVisible();
	await expect(page.getByLabel('Account name')).toHaveValue('Renamed account');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('heading', { name: 'Renamed account' })).toBeVisible();
	await page.getByRole('link', { name: '← Simulated accounts' }).click();
	await expect(page.locator('.simulated-accounts__row')).toContainText('Renamed account');
	await expect(page.getByText('Connector', { exact: true })).toHaveCount(0);
	await expect(page.getByText('S3', { exact: true })).toHaveCount(0);
	await page.locator('.simulated-accounts__row').click();
	await expect(page.getByRole('heading', { name: 'Renamed account' })).toBeVisible();
	await page.locator('.simulated-accounts__danger sl-button').click();
	await page.locator('.alert-dialog sl-button').last().click();
	await expect(page.getByText('Delete interrupted')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Renamed account' })).toBeVisible();
	await page.locator('.simulated-accounts__danger sl-button').click();
	await page.locator('.alert-dialog sl-button').last().click();
	await expect(page.getByRole('heading', { name: 'No simulated accounts yet' })).toBeVisible();
	expect(renameRequests).toBe(2);
	expect(deleteRequests).toBe(2);
});
