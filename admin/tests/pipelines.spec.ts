import { test, expect } from '@playwright/test';
import {
	addDummyDestination,
	addDummySource,
	addFileSystemDestination,
	addFileSystemSource,
	addJavascriptSource,
	addPostgreSQLDestination,
	addPostgreSQLSource,
	deepComparePipelineSchema,
	fillUserPipelineFilters,
	login,
	logout,
	adminURL,
} from './utils';
import { join } from 'path';
import { writeFile } from 'fs';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Add "Import customers" pipeline on Dummy`, async ({ page }) => {
	const id = await addDummySource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import Dummy customers',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	let email = page.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let dummyId = page.locator('.combobox[data-id="dummy_id"]');
	await dummyId.locator('sl-input').click();
	await dummyId.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'dummyId' }).click();
	await page.keyboard.press('Escape');

	let firstName = page.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'firstName' }).click();
	await page.keyboard.press('Escape');

	let lastName = page.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'lastName' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
	{
		"target": "User",
		"eventType": null,
		"name": "Import Dummy customers",
		"enabled": true,
		"filter": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{ "name": "email", "type": { "kind": "string" }, "description": "Email", "nullable": true },
				{ "name": "dummyId", "type": { "kind": "string" }, "description": "Dummy ID" },
				{ "name": "firstName", "type": { "kind": "string" }, "description": "First name", "nullable": true },
				{ "name": "lastName", "type": { "kind": "string" }, "description": "Last name", "nullable": true }
			]
		},
		"outSchema": {
			"kind": "object",
			"properties": [
				{ "name": "email", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true, "description": "" },
				{ "name": "dummy_id", "type": { "kind": "string" }, "readOptional": true, "description": "" },
				{ "name": "first_name", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true, "description": "" },
				{ "name": "last_name", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true, "description": "" }
			]
		},
		"incremental": false,
		"transformation": {
			"mapping": {
				"email": "email",
				"dummy_id": "dummyId",
				"first_name": "firstName",
				"last_name": "lastName"
			}
		}
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Export customers" pipeline on Dummy`, async ({ page }) => {
	const id = await addDummyDestination(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Export customers',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// Export mode.
	let exportMode = page.locator('.pipeline__export-mode');
	await exportMode.locator('sl-select').click();
	await exportMode.locator('sl-option[value="UpdateOnly"]').click();

	// Matching.
	let matching = page.locator('.pipeline__matching-properties');
	await matching.locator('[data-id="in"] sl-input >> input').click();
	await matching.locator('[data-id="in"] sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	await matching.locator('[data-id="out"] sl-input >> input').click();
	await matching.locator('[data-id="out"] sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();
	// Selected out matching property should not be visible in the
	// mapping.
	await expect(
		page.locator('.pipeline__transformation-mappings .pipeline__transformation-output-property >> input', {
			hasText: 'email',
		}),
	).not.toBeAttached();

	// Update on duplicates.
	await page.locator('.pipeline__update-on-duplicates sl-checkbox').click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let firstName = mappings.locator('.combobox[data-id="firstName"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="lastName"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
	{
		"target": "User",
		"eventType": null,
		"name": "Export customers",
		"enabled": true,
		"filter": {
			"logical": "or",
			"conditions": [
				{
					"property": "email",
					"operator": "is one of",
					"values": [
						"acme@krenalis.com",
						"test@krenalis.com"
					]
				},
				{
					"property": "dummy_id",
					"operator": "is between",
					"values": [
						"1200",
						"1800"
					]
				}
			]
		},
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "dummy_id",
					"type": {
						"kind": "string"
					},
					"readOptional": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "firstName",
					"type": {
						"kind": "string"
					},
					"nullable": true,
					"description": "First name"
				},
				{
					"name": "lastName",
					"type": {
						"kind": "string"
					},
					"nullable": true,
					"description": "Last name"
				},
				{
					"name": "email",
					"type": {
						"kind": "string"
					},
					"nullable": true,
					"description": "Email"
				}
			]
		},
		"transformation": {
			"mapping": {
				"firstName": "first_name",
				"lastName": "last_name"
			}
		},
		"exportMode": "UpdateOnly",
		"matching": {
			"in": "email",
			"out": "email"
		},
		"updateOnDuplicates": true
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Send Add to Cart" pipeline on Dummy`, async ({ page }) => {
	const id = await addDummyDestination(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Send Add to Cart',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'traits' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
	{
		"target": "Event",
		"eventType": "send_add_to_cart",
		"name": "Send Add to Cart",
		"enabled": false,
		"filter": null,
		"inSchema": null,
		"outSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "email",
					"type": {
						"kind": "string"
					},
					"createRequired": true,
					"description": "Email"
				}
			]
		},
		"transformation": {
			"mapping": {
				"email": "traits"
			}
		}
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Import users" pipeline on PostgreSQL`, async ({ page }) => {
	const id = await addPostgreSQLSource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Query.
	await page.click('.monaco-editor');
	await page.keyboard.press('Control+A');
	await page.keyboard.press('Backspace');
	await page.keyboard.type('SELECT email, first_name, last_name FROM users WHERE ${updated_at} LIMIT ${limit}');
	await page.click('.pipeline__query-confirm');
	await expect(page.locator('.pipeline__transformation')).toBeAttached();

	// Identity column.
	const identity = page.locator('.pipeline__transformation-identity-column');
	await identity.locator('sl-input').click();
	await identity.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'last_name' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
	{
		"target": "User",
		"eventType": null,
		"name": "Import users",
		"enabled": true,
		"filter": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				}
			]
		},
		"transformation": {
			"mapping": {
				"first_name": "first_name",
				"last_name": "last_name"
			}
		},
		"query": "SELECT email, first_name, last_name FROM users WHERE \${updated_at} LIMIT \${limit}",
		"incremental": false,
		"userIDColumn": "email",
		"updatedAtColumn": "",
		"updatedAtFormat": ""
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Export users" pipeline on PostgreSQL`, async ({ page }) => {
	const id = await addPostgreSQLDestination(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Export users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// Table.
	await page.locator('.pipeline__destination_table sl-input >> input').fill('users');
	await page.locator('.pipeline__destination_table sl-button').click();

	await expect(page.locator('.pipeline__destination_table-key-section')).toBeAttached();
	await expect(page.locator('.pipeline__transformation')).toBeAttached();

	// Table key.
	let tableKey = page.locator('.pipeline__destination_table-key-property');
	await tableKey.locator('sl-input >> input').click();
	await tableKey.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
	{
		"target": "User",
		"eventType": null,
		"name": "Export users",
		"enabled": true,
		"filter": {
			"logical": "or",
			"conditions": [
				{
					"property": "email",
					"operator": "is one of",
					"values": [
						"acme@krenalis.com",
						"test@krenalis.com"
					]
				},
				{
					"property": "dummy_id",
					"operator": "is between",
					"values": [
						"1200",
						"1800"
					]
				}
			]
		},
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "dummy_id",
					"type": {
						"kind": "string"
					},
					"readOptional": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"createRequired": true,
					"updateRequired": false,
					"nullable": false,
					"description": ""
				},
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true,
					"description": ""
				}
			]
		},
		"transformation": {
			"mapping": {
				"email": "email",
				"first_name": "first_name",
				"last_name": "last_name"
			}
		},
		"tableName": "users",
		"tableKey": "email"
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Import users" pipeline on CSV file on File System`, async ({ page }) => {
	const connectionID = await addFileSystemSource(page);

	const tempDir = process.env.KRENALIS_TEST_FS_TEMP_DIR;
	if (!tempDir) {
		throw new Error('Missing environment variable: KRENALIS_TEST_FS_TEMP_DIR');
	}

	// Create a temporary file.
	const fileName = 'test.csv';
	const tempFilePath = join(tempDir, fileName);
	writeFile(tempFilePath, 'first_name, last_name, email\nJohn, Doe, example@krenalis.com', (err) => {
		if (err) throw err;
	});

	await page.goto(`${adminURL}/connectors?role=Source`);
	await page.click(`[data-code="csv"]`);
	await page.click('.connectors-list__documentation-add');

	await page.click('.file-connector__storage sl-select');
	await page.locator(`.file-connector__storage sl-select sl-option[value="${connectionID}"]`).click();

	let name = page.locator('.file-connector__pipeline-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// File
	await page.locator('.pipeline__file-path >> input').fill(fileName);
	await page.click('.connector-ui .connector-checkbox:last-child sl-checkbox');

	await page.click('.pipeline__file-confirm');

	// Identity column.
	const identity = page.locator('.pipeline__transformation-identity-column');
	await identity.locator('sl-input').click();
	await identity.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();
	await page.keyboard.press('Escape');

	const expectedBody = `
		{
			"target": "User",
			"eventType": null,
			"name": "Import users",
			"enabled": true,
			"filter": null,
			"inSchema": {
				"kind": "object",
				"properties": [
					{
						"name": "email",
						"type": {
							"kind": "string"
						},
						"description": ""
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string"
						},
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string"
						},
						"description": ""
					}
				]
			},
			"outSchema": {
				"kind": "object",
				"properties": [
					{
						"name": "email",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					}
				]
			},
			"transformation": {
				"mapping": {
					"email": "email",
					"first_name": "first_name",
					"last_name": "last_name"
				}
			},
			"path": "test.csv",
			"sheet": null,
			"userIDColumn": "email",
			"incremental": false,
			"updatedAtColumn": "",
			"updatedAtFormat": "",
			"compression": "",
			"format": "csv",
			"formatSettings": {
				"separator": ",",
				"numberOfColumns": 0,
				"hasColumnNames": true,
				"lazyQuotes": false,
				"trimLeadingSpace": false,
				"useCRLF": false
			}
		}`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = connectionID;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Export users" pipeline on CSV file on File System`, async ({ page }) => {
	const connectionID = await addFileSystemDestination(page);

	const tempDir = process.env.KRENALIS_TEST_FS_TEMP_DIR;
	if (!tempDir) {
		throw new Error('Missing environment variable: KRENALIS_TEST_FS_TEMP_DIR');
	}

	// Create a temporary file.
	const fileName = 'test.csv';

	const tempFilePath = join(tempDir, fileName);
	writeFile(tempFilePath, '', (err) => {
		if (err) throw err;
	});

	await page.goto(`${adminURL}/connectors?role=Destination`);
	await page.click(`[data-code="csv"]`);
	await page.click('.connectors-list__documentation-add');

	await page.click('.file-connector__storage sl-select');
	await page.locator(`.file-connector__storage sl-select sl-option[value="${connectionID}"]`).click();

	let name = page.locator('.file-connector__pipeline-types .list-tile__name', {
		hasText: 'Export users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// File
	await page.locator('.pipeline__file-format').click();
	await page.locator('.pipeline__file-format sl-option[value="csv"]').click();

	await page.locator('.pipeline__file-path >> input').fill(fileName);

	const expectedBody = `
		{
			"target": "User",
			"eventType": null,
			"name": "Export users",
			"enabled": true,
			"filter": {
				"logical": "or",
				"conditions": [
					{
						"property": "email",
						"operator": "is one of",
						"values": [
							"acme@krenalis.com",
							"test@krenalis.com"
						]
					},
					{
						"property": "dummy_id",
						"operator": "is between",
						"values": [
							"1200",
							"1800"
						]
					}
				]
			},
			"inSchema": {
				"kind": "object",
				"properties": [
					{
						"name": "email",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "dummy_id",
						"type": {
							"kind": "string"
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "android",
						"type": {
							"kind": "object",
							"properties": [
								{
									"name": "id",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "idfa",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "push_token",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								}
							]
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "ios",
						"type": {
							"kind": "object",
							"properties": [
								{
									"name": "id",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "idfa",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "push_token",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								}
							]
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "gender",
						"type": {
							"kind": "string"
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "food_preferences",
						"type": {
							"kind": "object",
							"properties": [
								{
									"name": "drink",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "fruit",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								}
							]
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "phone_numbers",
						"type": {
							"kind": "array",
							"elementType": {
								"kind": "string",
								"maxLength": 300
							}
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "favorite_movie",
						"type": {
							"kind": "object",
							"properties": [
								{
									"name": "title",
									"type": {
										"kind": "string"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "length",
									"type": {
										"kind": "float",
										"bitSize": 64
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "soundtrack",
									"type": {
										"kind": "object",
										"properties": [
											{
												"name": "title",
												"type": {
													"kind": "string"
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "author",
												"type": {
													"kind": "string"
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "length",
												"type": {
													"kind": "float",
													"bitSize": 64
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "genre",
												"type": {
													"kind": "string"
												},
												"readOptional": true,
												"description": ""
											}
										]
									},
									"readOptional": true,
									"description": ""
								}
							]
						},
						"readOptional": true,
						"description": ""
					}
				]
			},
			"outSchema": null,
			"transformation": null,
			"path": "test.csv",
			"sheet": null,
			"userIDColumn": "",
			"updatedAtColumn": "",
			"updatedAtFormat": "",
			"compression": "",
			"orderBy": "email",
			"format": "csv",
			"formatSettings": {
				"separator": ",",
				"numberOfColumns": 0,
				"hasColumnNames": false,
				"lazyQuotes": false,
				"trimLeadingSpace": false,
				"useCRLF": false
			}
		}`;
	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = connectionID;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Import events" pipeline on JavaScript`, async ({ page }) => {
	const id = await addJavascriptSource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import events',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await page.locator('.pipeline__filters-add-condition').click();
	await page.locator('.pipeline__filters-add-condition').click();
	await page.locator('.pipeline__filters-add-condition').click();

	let filters = page.locator('.pipeline__filters-filter');

	await filters.nth(0).locator('.pipeline__filters-property sl-input').click();
	await filters
		.nth(0)
		.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^type$/ })
		.click();
	await filters.nth(0).locator('.pipeline__filters-operator sl-option[value="0"]').click(); // option is "is".

	await filters.nth(0).locator('.pipeline__filters-value-input sl-option[value="track"]').click(); // value select should open automatically after selecting the operator

	const expectedBody = `
	{
		"target": "Event",
		"eventType": null,
		"name": "Import events into warehouse",
		"enabled": false,
		"filter": {
			"logical": "and",
			"conditions": [
				{
					"property": "type",
					"operator": "is",
					"values": [
						"track"
					]
				}
			]
		},
		"inSchema": null,
		"outSchema": null,
		"transformation": null
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});

test(`Add "Import users" pipeline on JavaScript`, async ({ page }) => {
	const id = await addJavascriptSource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	const expectedBody = `
	{
		"target": "User",
		"eventType": null,
		"name": "Import users into warehouse",
		"enabled": false,
		"filter": {
			"logical": "or",
			"conditions": [
				{
					"property": "type",
					"operator": "is",
					"values": [
						"identify"
					]
				},
				{
					"property": "traits",
         			"operator": "is not empty",
         			"values": null
       			}
			]
		},
		"inSchema": null,
		"outSchema": null,
		"transformation": null
	}
	`;

	let saveButton = page.locator('.pipeline__header-save >> button');
	const [response] = await Promise.all([
		page.waitForResponse((response) => {
			return response.url().includes('/pipelines') && response.request().method() === 'POST';
		}),
		saveButton.click(),
	]);

	const status = response.status();
	if (status !== 200) {
		throw new Error(`Unexpected response status while adding the pipeline: ${status}`);
	}

	const got = JSON.parse(response.request().postData());
	let expected = JSON.parse(expectedBody);
	expected.connection = id;
	deepComparePipelineSchema(got, expected);

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();

	await page.reload();

	await expect(page.locator('.connection-pipelines__grid')).toBeAttached();
});
