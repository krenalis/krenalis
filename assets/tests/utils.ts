import { expect, Page } from '@playwright/test';
import API from '../src/lib/api/api';
import { Property } from '../src/lib/api/types/types';
import { readFileSync, rmSync, mkdtempSync } from 'fs';
import { tmpdir } from 'os';
import { join, resolve } from 'path';

interface Config {
	baseURL: string;
	workspaceID: number;
	dbHost: string;
	dbPort: string;
	dbUsername: string;
	dbPassword: string;
	dbName: string;
	dbSchema: string;
}

const config: Config = JSON.parse(readFileSync(resolve(__dirname, './test-config.json'), 'utf-8'));
const uiURL = `${config.baseURL}/ui/`;

const login = async (page: Page) => {
	await page.goto(uiURL);
	await page.evaluate(
		async ({ url, workspace }) => {
			localStorage.setItem('meergo_ui_workspace_id', String(workspace));
			const api = new (window as any).API(url, workspace) as API;
			await api.login('acme@open2b.com', 'foopass2');
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
};

const logout = async (page: Page) => {
	await page.goto(uiURL);
	await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			await api.logout();
			localStorage.removeItem('meergo_ui_workspace_id');
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
};

const addDummySource = async (page: Page): Promise<number> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'Dummy',
					role: 'Source',
					connector: 'Dummy',
					strategy: null,
					websiteHost: '',
					sendingMode: null,
					settings: {},
					linkedConnections: null,
				},
				'',
			);
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
	return id;
};

const addDummyDestination = async (page: Page): Promise<number> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'Dummy',
					role: 'Destination',
					connector: 'Dummy',
					strategy: null,
					websiteHost: '',
					sendingMode: 'Cloud',
					settings: { URLForDispatchingEvents: '' },
					linkedConnections: null,
				},
				'',
			);
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
	return id;
};

const addPostgreSQLSource = async (page: Page): Promise<number> => {
	const id = await page.evaluate(
		async ({ config }) => {
			const api = new (window as any).API(config.baseURL, config.workspaceID) as API;
			return await api.workspaces.createConnection(
				{
					name: 'PostgreSQL',
					role: 'Source',
					connector: 'PostgreSQL',
					strategy: null,
					websiteHost: '',
					sendingMode: null,
					settings: {
						Database: config.dbName,
						Host: config.dbHost,
						Password: config.dbPassword,
						Port: config.dbPort,
						Username: config.dbUsername,
						Schema: config.dbSchema,
					},
					linkedConnections: null,
				},
				'',
			);
		},
		{ config: config },
	);
	return id;
};

const addPostgreSQLDestination = async (page: Page): Promise<number> => {
	const id = await page.evaluate(
		async ({ config }) => {
			const api = new (window as any).API(config.baseURL, config.workspaceID) as API;
			return await api.workspaces.createConnection(
				{
					name: 'PostgreSQL',
					role: 'Destination',
					connector: 'PostgreSQL',
					strategy: null,
					websiteHost: '',
					sendingMode: null,
					settings: {
						Database: config.dbName,
						Host: config.dbHost,
						Password: config.dbPassword,
						Port: config.dbPort,
						Username: config.dbUsername,
						Schema: config.dbSchema,
					},
					linkedConnections: null,
				},
				'',
			);
		},
		{ config: config },
	);
	return id;
};

const addFileSystemSource = async (
	page: Page,
	func: (tempDir: string, connectionID: number) => Promise<void>,
): Promise<void> => {
	await createTempDirectory(async (tempDir: string) => {
		const id = await page.evaluate(
			async ({ url, workspace, tempDir }) => {
				const api = new (window as any).API(url, workspace) as API;
				return await api.workspaces.createConnection(
					{
						name: 'Filesystem',
						role: 'Source',
						connector: 'Filesystem',
						strategy: null,
						websiteHost: '',
						sendingMode: null,
						settings: {
							Root: tempDir,
						},
						linkedConnections: null,
					},
					'',
				);
			},
			{ url: config.baseURL, workspace: config.workspaceID, tempDir: tempDir },
		);
		await func(tempDir, id);
	});
};

const addFileSystemDestination = async (
	page: Page,
	func: (tempDir: string, connectionID: number) => Promise<void>,
): Promise<void> => {
	await createTempDirectory(async (tempDir: string) => {
		const id = await page.evaluate(
			async ({ url, workspace, tempDir }) => {
				const api = new (window as any).API(url, workspace) as API;
				return await api.workspaces.createConnection(
					{
						name: 'Filesystem',
						role: 'Destination',
						connector: 'Filesystem',
						strategy: null,
						websiteHost: '',
						sendingMode: null,
						settings: {
							Root: tempDir,
						},
						linkedConnections: null,
					},
					'',
				);
			},
			{ url: config.baseURL, workspace: config.workspaceID, tempDir: tempDir },
		);
		await func(tempDir, id);
	});
};

const addJavascriptSource = async (page: Page): Promise<number> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'JavaScript',
					role: 'Source',
					connector: 'JavaScript',
					strategy: 'Conversion',
					websiteHost: '',
					sendingMode: null,
					settings: null,
					linkedConnections: null,
				},
				'',
			);
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
	return id;
};

const fillUserActionFilters = async (page: Page): Promise<void> => {
	await page.waitForTimeout(1000);

	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();

	let filters = page.locator('.action__filters-filter');

	await filters.nth(0).locator('.action__filters-property sl-input').click();
	await filters.nth(0).locator('sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	await filters.nth(0).locator('.action__filters-operator sl-option[value="10"]').click(); // option is "is one of".
	await filters.nth(0).locator('.action__filters-add-value').click();
	await filters.nth(0).locator('.action__filters-add-value').click();

	await page.waitForTimeout(1000);

	await filters
		.nth(0)
		.locator('.action__filters-is-one-of-values > .action__filters-value-input:nth-child(1) >> input')
		.fill('acme@open2b.com');
	await filters
		.nth(0)
		.locator('.action__filters-is-one-of-values > .action__filters-value:nth-child(2) >> input')
		.fill('test@open2b.com');
	await filters
		.nth(0)
		.locator('.action__filters-is-one-of-values > .action__filters-value:nth-child(3) >> input')
		.fill('foo@open2b.com');
	await filters
		.nth(0)
		.locator(
			'.action__filters-is-one-of-values > .action__filters-value:nth-child(3) .action__filters-value-remove',
		)
		.click(); // remove the last value.

	await page.locator('.action__filters-logical sl-button:nth-child(2)').click(); // set the logical to 'or'.

	await filters.nth(1).locator('.action__filters-property sl-input').click();
	await filters.nth(1).locator('sl-menu-item .schema-combobox-item__name', { hasText: 'dummy_id' }).click();
	await filters.nth(1).locator('.action__filters-operator sl-option[value="6"]').click(); // option is "is between".
	await filters.nth(1).locator('.action__filters-value-input:nth-child(2) >> input').fill('1200');
	await filters.nth(1).locator('.action__filters-value-input:nth-child(4) >> input').fill('1800');

	await filters.nth(2).locator('.action__filters-remove-condition').click(); // remove the last filter.
};

const deepCompareActionSchema = (actual: object, expected: object) => {
	const actualCopy = JSON.parse(JSON.stringify(actual));
	const expectedCopy = JSON.parse(JSON.stringify(expected));

	// sort the properties of the action schemas.
	if (actualCopy.inSchema && actualCopy.inSchema.properties) {
		actualCopy.inSchema.properties.sort((a: Property, b: Property) =>
			a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1,
		);
	}
	if (actualCopy.outSchema && actualCopy.outSchema.properties) {
		actualCopy.outSchema.properties.sort((a: Property, b: Property) =>
			a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1,
		);
	}
	if (expectedCopy.inSchema && expectedCopy.inSchema.properties) {
		expectedCopy.inSchema.properties.sort((a: Property, b: Property) =>
			a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1,
		);
	}
	if (expectedCopy.outSchema && expectedCopy.outSchema.properties) {
		expectedCopy.outSchema.properties.sort((a: Property, b: Property) =>
			a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1,
		);
	}

	expect(actualCopy).toEqual(expectedCopy);
};

const createTempDirectory = async (func: (tempDir: string) => Promise<void>) => {
	let tempDir: string;
	const appPrefix = 'meergo-ui-test';
	try {
		tempDir = mkdtempSync(join(tmpdir(), appPrefix));
		await func(tempDir);
	} catch (err) {
		throw err;
	} finally {
		try {
			if (tempDir) {
				rmSync(tempDir, { recursive: true });
			}
		} catch (e) {
			console.error(
				`An error has occurred while removing the temp folder at ${tempDir}. Please remove it manually. Error: ${e}`,
			);
		}
	}
};

export {
	config,
	uiURL,
	login,
	logout,
	addDummySource,
	addDummyDestination,
	addPostgreSQLSource,
	addPostgreSQLDestination,
	addFileSystemSource,
	addFileSystemDestination,
	addJavascriptSource,
	fillUserActionFilters,
	deepCompareActionSchema,
	createTempDirectory,
};
