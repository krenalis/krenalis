import { expect, Page } from '@playwright/test';
import API from '../src/lib/api/api';
import { Property } from '../src/lib/api/types/types';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { IS_PASSWORDLESS_KEY, WORKSPACE_ID_KEY } from '../src/constants/storage';

interface Config {
	baseURL: string;
	workspaceID: string;
	dbHost: string;
	dbPort: string;
	dbUsername: string;
	dbPassword: string;
	dbName: string;
	dbSchema: string;
}

const config: Config = JSON.parse(readFileSync(resolve(__dirname, './test-config.json'), 'utf-8'));
const adminPath = 'admin';
const adminURL = `${config.baseURL}/${adminPath}`;

const login = async (page: Page) => {
	await page.goto(`${adminURL}/`);
	await page.evaluate(
		async ({ url, workspace, workspaceIDKey }) => {
			localStorage.setItem(workspaceIDKey, String(workspace));
			const api = new (window as any).API(url, workspace) as API;
			await api.login('test@krenalis.com', 'krenalis-password');
		},
		{ url: config.baseURL, workspace: config.workspaceID, workspaceIDKey: WORKSPACE_ID_KEY },
	);
};

const logout = async (page: Page) => {
	await page.evaluate(
		async ({ url, workspace, workspaceIDKey, isPasswordlessKey }) => {
			const api = new (window as any).API(url, workspace) as API;
			await api.logout();
			localStorage.removeItem(workspaceIDKey);
			localStorage.removeItem(isPasswordlessKey);
		},
		{
			url: config.baseURL,
			workspace: config.workspaceID,
			workspaceIDKey: WORKSPACE_ID_KEY,
			isPasswordlessKey: IS_PASSWORDLESS_KEY,
		},
	);
	await page.goto(`${adminURL}/`);
};

const addDummySource = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'Dummy',
					role: 'Source',
					connector: 'dummy',
					strategy: null,
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

const addDummyDestination = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'Dummy',
					role: 'Destination',
					connector: 'dummy',
					strategy: null,
					sendingMode: 'Server',
					settings: { urlForDispatchingEvents: '' },
					linkedConnections: null,
				},
				'',
			);
		},
		{ url: config.baseURL, workspace: config.workspaceID },
	);
	return id;
};

const addPostgreSQLSource = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ config }) => {
			const api = new (window as any).API(config.baseURL, config.workspaceID) as API;
			return await api.workspaces.createConnection(
				{
					name: 'PostgreSQL',
					role: 'Source',
					connector: 'postgresql',
					strategy: null,
					sendingMode: null,
					settings: {
						database: config.dbName,
						host: config.dbHost,
						password: config.dbPassword,
						port: config.dbPort,
						username: config.dbUsername,
						schema: config.dbSchema,
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

const addPostgreSQLDestination = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ config }) => {
			const api = new (window as any).API(config.baseURL, config.workspaceID) as API;
			return await api.workspaces.createConnection(
				{
					name: 'PostgreSQL',
					role: 'Destination',
					connector: 'postgresql',
					strategy: null,
					sendingMode: null,
					settings: {
						database: config.dbName,
						host: config.dbHost,
						password: config.dbPassword,
						port: config.dbPort,
						username: config.dbUsername,
						schema: config.dbSchema,
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

const addFileSystemSource = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'File System',
					role: 'Source',
					connector: 'filesystem',
					strategy: null,
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

const addFileSystemDestination = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'File System',
					role: 'Destination',
					connector: 'filesystem',
					strategy: null,
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

const addJavascriptSource = async (page: Page): Promise<string> => {
	const id = await page.evaluate(
		async ({ url, workspace }) => {
			const api = new (window as any).API(url, workspace) as API;
			return await api.workspaces.createConnection(
				{
					name: 'JavaScript',
					role: 'Source',
					connector: 'javascript',
					strategy: 'Conversion',
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

const fillUserPipelineFilters = async (page: Page): Promise<void> => {
	await page.waitForTimeout(1000);

	await page.locator('.pipeline__filters-add-condition').click();

	const rootFilter = page.locator('.filter-editor__group--root');
	const rootRemoveConditionButton = rootFilter.getByRole('button', { name: 'Remove condition 1', exact: true });
	const rootRemoveConditionTooltip = rootFilter.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule:not(.filter-editor__rule--group) .filter-editor__remove-condition-wrapper > wa-tooltip',
	);
	await expect(rootRemoveConditionButton).toHaveJSProperty('disabled', false);
	await expect(rootRemoveConditionTooltip).toHaveCount(1);
	await expect(rootRemoveConditionTooltip).toHaveText('Remove condition 1');

	await page.locator('.filter-editor__add-condition').click();
	await page.locator('.filter-editor__add-condition').click();

	await expect(rootFilter.locator(':scope > .filter-editor__group-header .filter-editor__group-subject')).toHaveText(
		'Profiles matching',
	);
	const connectors = rootFilter.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule > .filter-editor__connector',
	);
	await expect(connectors).toHaveText(['and', 'and']);
	const rootActions = rootFilter.locator(':scope > .filter-editor__group-actions');
	await rootActions.locator('.filter-editor__add-group').click();

	const nestedGroup = rootFilter.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule--group > .filter-editor__rule-content > .filter-editor__group',
	);
	const nestedRemoveGroupTooltip = nestedGroup.locator(
		':scope > .filter-editor__group-header .filter-editor__remove-group-wrapper > wa-tooltip',
	);
	const nestedRemoveConditionTooltips = nestedGroup.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule:not(.filter-editor__rule--group) .filter-editor__remove-condition-wrapper > wa-tooltip',
	);
	await expect(nestedGroup.locator('.filter-editor__logical')).toHaveJSProperty('value', 'or');
	await expect(nestedRemoveGroupTooltip).toHaveCount(1);
	await expect(nestedRemoveGroupTooltip).toHaveText('Remove group');

	const nestedRemoveConditionButtons = nestedGroup.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule:not(.filter-editor__rule--group) .filter-editor__remove-condition',
	);
	await expect(nestedRemoveConditionButtons).toHaveCount(1);
	await expect(nestedRemoveConditionButtons.first()).toHaveJSProperty('disabled', true);
	await expect(nestedRemoveConditionTooltips).toHaveCount(0);

	await nestedGroup.locator('.filter-editor__add-condition').click();
	await expect(nestedRemoveConditionButtons).toHaveCount(2);
	await expect(nestedRemoveConditionButtons.first()).toHaveJSProperty('disabled', false);
	await expect(nestedRemoveConditionButtons.nth(1)).toHaveJSProperty('disabled', false);
	await expect(nestedRemoveConditionTooltips).toHaveCount(2);
	await expect(nestedRemoveConditionButtons.first()).toHaveAccessibleName('Remove condition 4');
	await expect(nestedRemoveConditionButtons.nth(1)).toHaveAccessibleName('Remove condition 5');
	await expect(nestedRemoveConditionTooltips.first()).toHaveText('Remove condition 4');
	await expect(nestedRemoveConditionTooltips.nth(1)).toHaveText('Remove condition 5');

	await nestedRemoveConditionButtons.nth(1).click();
	await expect(nestedRemoveConditionButtons).toHaveCount(1);
	await expect(nestedRemoveConditionButtons.first()).toHaveJSProperty('disabled', true);
	await expect(nestedGroup.locator('.filter-editor__property wa-input')).toBeFocused();
	await nestedGroup.locator('.filter-editor__add-group').click();
	await expect(nestedRemoveGroupTooltip).toHaveText('Remove groups');
	await expect(nestedRemoveConditionButtons.first()).toHaveJSProperty('disabled', false);
	await nestedRemoveConditionButtons.first().click();

	const childGroup = nestedGroup.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule--group > .filter-editor__rule-content > .filter-editor__group',
	);
	await childGroup
		.locator(':scope > .filter-editor__group-header')
		.getByRole('button', { name: 'Remove group' })
		.click();
	await expect(childGroup).toHaveCount(0);
	await expect(nestedRemoveConditionButtons).toHaveCount(1);
	await expect(nestedRemoveConditionButtons.first()).toHaveJSProperty('disabled', true);
	await expect(nestedGroup.locator('.filter-editor__property wa-input')).toHaveJSProperty('value', '');
	await expect(nestedRemoveGroupTooltip).toHaveText('Remove group');

	await nestedGroup.locator('.filter-editor__add-group').click();
	await expect(nestedRemoveGroupTooltip).toHaveText('Remove groups');
	await nestedGroup
		.first()
		.locator(':scope > .filter-editor__group-header')
		.getByRole('button', { name: 'Remove group' })
		.click();

	const filters = page.locator('.filter-editor__filter');

	await filters.nth(0).locator('.filter-editor__property wa-input').click();
	await filters.nth(0).locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'email' }).click();
	await filters.nth(0).locator('.filter-editor__operator wa-option[value="10"]').click(); // option is "is one of".
	await filters.nth(0).locator('.filter-editor__add-value').click();
	await filters.nth(0).locator('.filter-editor__add-value').click();

	await page.waitForTimeout(1000);

	await filters
		.nth(0)
		.locator('.filter-editor__is-one-of-values > .filter-editor__value-input:nth-child(1) >> input')
		.fill('acme@krenalis.com');
	await filters
		.nth(0)
		.locator('.filter-editor__is-one-of-values > .filter-editor__value:nth-child(2) >> input')
		.fill('test@krenalis.com');
	await filters
		.nth(0)
		.locator('.filter-editor__is-one-of-values > .filter-editor__value:nth-child(3) >> input')
		.fill('foo@krenalis.com');
	await filters
		.nth(0)
		.locator('.filter-editor__is-one-of-values > .filter-editor__value:nth-child(3) .filter-editor__value-remove')
		.click(); // remove the last value.

	await page.locator('.filter-editor__logical').click();
	await page.locator('.filter-editor__logical wa-option[value="or"]').click();
	await expect(connectors).toHaveText(['or', 'or']);

	await filters.nth(1).locator('.filter-editor__property wa-input').click();
	await filters.nth(1).locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'dummy_id' }).click();
	await filters.nth(1).locator('.filter-editor__operator wa-option[value="6"]').click(); // option is "is between".
	await filters.nth(1).locator('.filter-editor__value-input:nth-child(2) >> input').fill('1200');
	await page.waitForTimeout(2000);
	const lastBetweenValue = filters.nth(1).locator('.filter-editor__value-input:nth-child(4) >> input');
	await lastBetweenValue.fill('1800');
	await lastBetweenValue.focus();
	await page.keyboard.press('Tab');
	await expect(filters.nth(1).getByRole('button', { name: 'Remove condition 2', exact: true })).toBeFocused();

	await filters.nth(2).locator('.filter-editor__remove-condition').click(); // remove the last filter.
	await expect(filters).toHaveCount(2);
	await expect(filters.nth(1).locator('.filter-editor__property wa-input')).toBeFocused();
	await expect(page.locator('.filter-editor__announcement')).toHaveText('Condition removed');

	await expect(rootActions.locator('.filter-editor__add-condition')).toContainText('Add a condition');
	await expect(rootActions.locator('.filter-editor__add-group')).toContainText('Add a group');
	await rootActions.locator('.filter-editor__add-group').click();

	await expect(nestedGroup.locator('.filter-editor__group-subject')).toHaveText('matching');
	await expect(nestedGroup.locator('.filter-editor__logical')).toHaveJSProperty('value', 'and');
	const removeGroupButton = nestedGroup.getByRole('button', { name: 'Remove group' });
	await expect(removeGroupButton).toBeAttached();
	await removeGroupButton.click();
};

const deepComparePipelineSchema = (actual: object, expected: object) => {
	const actualCopy: any = structuredClone(actual);
	const expectedCopy: any = structuredClone(expected);

	// sort the properties of the pipeline schemas.
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

const logValidationErrors = async (page: Page, selectors: string[]): Promise<void> => {
	const errors = await page.evaluate((selectors: string) => {
		const containers = document.querySelectorAll(selectors);
		const e = [];
		for (const el of Array.from(containers)) {
			const message = el.textContent;
			if (message !== '') {
				const errorOn = (el as any).dataset['errorOn'];
				e.push({
					errorOn,
					message,
				});
			}
		}
		return e;
	}, selectors.join(','));

	if (errors.length > 0) {
		for (const error of errors) {
			console.log(`Validation error on the "${error.errorOn}" field: ${error.message}`);
		}
	}
};

export {
	config,
	adminPath,
	adminURL,
	login,
	logout,
	addDummySource,
	addDummyDestination,
	addPostgreSQLSource,
	addPostgreSQLDestination,
	addFileSystemSource,
	addFileSystemDestination,
	addJavascriptSource,
	fillUserPipelineFilters,
	deepComparePipelineSchema,
	logValidationErrors,
};
