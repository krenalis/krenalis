import { test, expect } from '@playwright/test';
import {
	addDummyDestination,
	addDummySource,
	addFileSystemDestination,
	addFileSystemSource,
	addJavascriptSource,
	addPostgreSQLDestination,
	addPostgreSQLSource,
	deepCompareActionSchema,
	fillUserActionFilters,
	login,
	logout,
	uiURL,
} from './utils';
import { join } from 'path';
import { writeFile } from 'fs';

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Add "Import users" action on Dummy`, async ({ page }) => {
	const id = await addDummySource(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	let email = page.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	let dummyId = page.locator('.combobox[data-id="dummy_id"]');
	await dummyId.locator('sl-input').click();
	await dummyId.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'dummyId' }).click();

	let firstName = page.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'firstName' }).click();

	let lastName = page.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'lastName' }).click();

	const expectedBody = `
	{
		"target": "Users",
		"eventType": null,
		"name": "Import users",
		"enabled": true,
		"filter": null,
		"inSchema": {
			"name": "Object",
			"properties": [
				{ "name": "email", "type": { "name": "Text" }, "description": "" },
				{ "name": "dummyId", "type": { "name": "Text" }, "description": "" },
				{ "name": "firstName", "type": { "name": "Text" }, "description": "" },
				{ "name": "lastName", "type": { "name": "Text" }, "description": "" }
			]
		},
		"outSchema": {
			"name": "Object",
			"properties": [
				{ "name": "email", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "description": "" },
				{ "name": "dummy_id", "type": { "name": "Text" }, "readOptional": true, "description": "" },
				{ "name": "first_name", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "description": "" },
				{ "name": "last_name", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "description": "" }
			]
		},
		"incremental": true,
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

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const got = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(got, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Export users" action on Dummy`, async ({ page }) => {
	const id = await addDummyDestination(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Export users',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Filters.
	await fillUserActionFilters(page);

	// Export mode.
	let exportMode = page.locator('.action__export-mode');
	await exportMode.locator('sl-select').click();
	await exportMode.locator('sl-option[value="UpdateOnly"]').click();

	// Matching.
	let matching = page.locator('.action__matching-properties');
	await matching.locator('[data-id="in"] sl-input >> input').click();
	await matching.locator('[data-id="in"] sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	await matching.locator('[data-id="out"] sl-input >> input').click();
	await matching.locator('[data-id="out"] sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();
	// Selected out matching property should not be visible in the
	// mapping.
	await expect(
		page.locator('.action__transformation-mappings .action__transformation-output-property >> input', {
			hasText: 'email',
		}),
	).not.toBeVisible();

	// Export on duplicates.
	await page.locator('.action__export-on-duplicates sl-checkbox').click();

	// Mappings.
	let mappings = page.locator('.action__transformation');
	let firstName = mappings.locator('.combobox[data-id="firstName"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();

	let lastName = mappings.locator('.combobox[data-id="lastName"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();

	const expectedBody = `
	{
		"target": "Users",
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
						"acme@open2b.com",
						"test@open2b.com"
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
			"name": "Object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "email",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "dummy_id",
					"type": {
						"name": "Text"
					},
					"readOptional": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"name": "Object",
			"properties": [
				{
					"name": "firstName",
					"type": {
						"name": "Text"
					},
					"description": ""
				},
				{
					"name": "lastName",
					"type": {
						"name": "Text"
					},
					"description": ""
				},
				{
					"name": "email",
					"type": {
						"name": "Text"
					},
					"description": ""
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
		"exportOnDuplicates": true
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const got = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(got, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Send Add to Cart" action on Dummy`, async ({ page }) => {
	const id = await addDummyDestination(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Send Add to Cart',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Mappings.
	let mappings = page.locator('.action__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'traits' }).click();

	const expectedBody = `
	{
		"target": "Events",
		"eventType": "send_add_to_cart",
		"name": "Send Add to Cart",
		"enabled": false,
		"filter": null,
		"inSchema": null,
		"outSchema": {
			"name": "Object",
			"properties": [
				{
					"name": "email",
					"type": {
						"name": "Text"
					},
					"createRequired": true,
					"description": ""
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

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const got = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(got, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Import users" action on PostgreSQL`, async ({ page }) => {
	const id = await addPostgreSQLSource(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Query.
	await page.click('.monaco-editor');
	await page.keyboard.press('Control+A');
	await page.keyboard.press('Backspace');
	await page.keyboard.type('SELECT email, first_name, last_name FROM users WHERE ${last_change_time} LIMIT ${limit}');
	await page.click('.action__query-preview');
	await expect(page.locator('.action__query-preview-drawer')).toBeVisible();
	await page.locator('.action__query-preview-drawer >> [part="close-button"]').click();
	await page.click('.action__query-confirm');
	await expect(page.locator('.action__transformation')).toBeVisible();

	// Identity column.
	const identity = page.locator('.action__transformation-identity-column');
	await identity.locator('sl-input').click();
	await identity.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.action__transformation');
	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'first_name' }).click();
	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'last_name' }).click();

	const expectedBody = `
	{
		"target": "Users",
		"eventType": null,
		"name": "Import users",
		"enabled": true,
		"filter": null,
		"inSchema": {
			"name": "Object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "email",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"nullable": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"name": "Object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"name": "Text",
						"charLen": 300
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
		"query": "SELECT email, first_name, last_name FROM users WHERE \${last_change_time} LIMIT \${limit}",
		"identityColumn": "email",
		"lastChangeTimeColumn": "",
		"lastChangeTimeFormat": ""
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const got = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(got, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Export users" action on PostgreSQL`, async ({ page }) => {
	const id = await addPostgreSQLDestination(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Export users',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Filters.
	await fillUserActionFilters(page);

	// Table.
	await page.locator('.action__table sl-input >> input').fill('users');
	await page.locator('.action__table sl-button').click();

	await expect(page.locator('.action__table-key-section')).toBeVisible();
	await expect(page.locator('.action__transformation')).toBeVisible();

	// Table key.
	let tableKey = page.locator('.action__table-key-property');
	await tableKey.locator('sl-input >> input').click();
	await tableKey.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.action__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('sl-input').click();
	await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('sl-input').click();
	await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('sl-input').click();
	await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();

	const expectedBody = `
	{
		"target": "Users",
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
						"acme@open2b.com",
						"test@open2b.com"
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
			"name": "Object",
			"properties": [
				{
					"name": "email",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "first_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"readOptional": true,
					"description": ""
				},
				{
					"name": "dummy_id",
					"type": {
						"name": "Text"
					},
					"readOptional": true,
					"description": ""
				}
			]
		},
		"outSchema": {
			"name": "Object",
			"properties": [
				{
					"name": "email",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"createRequired": true,
					"updateRequired": false,
					"nullable": false,
					"description": ""
				},
				{
					"name": "first_name",
					"type": {
						"name": "Text",
						"charLen": 300
					},
					"nullable": true,
					"description": ""
				},
				{
					"name": "last_name",
					"type": {
						"name": "Text",
						"charLen": 300
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

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const got = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(got, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Import users" action on CSV file on Filesystem`, async ({ page }) => {
	await addFileSystemSource(page, async (tempDir: string, connectionID: number) => {
		// Create a temporary file.
		const fileName = 'test.csv';
		const tempFilePath = join(tempDir, fileName);
		writeFile(tempFilePath, 'first_name, last_name, email\nJohn, Doe, example@open2b.com', (err) => {
			if (err) throw err;
		});

		await page.goto(`${uiURL}connectors?role=Source`);
		await page.click('a[href="/ui/connectors/file/CSV?role=Source"]');

		await page.click('.file-connector__storage sl-select');
		await page.locator(`.file-connector__storage sl-select sl-option[value="${connectionID}"]`).click();

		let name = page.locator('.file-connector__action-types .list-tile__name', {
			hasText: 'Import users',
		});

		await expect(name).toBeVisible();

		let button = name.locator('..').locator('..').locator('sl-button');
		await button.click();
		await expect(page.locator('.action__header')).toBeVisible();

		// Filters
		//
		// TODO: currently there is an unhandled error when using the
		// filters of this type of action (see issue #1139).

		// File
		await page.locator('.action__file-path >> input').fill(fileName);
		await page.click('.connector-ui .connector-checkbox:last-child sl-checkbox');

		await page.click('.action__file-preview');

		const preview = page.locator('.action__file-preview-drawer');
		await expect(preview).toBeVisible();
		await expect(
			preview.locator('.grid__header-row .grid__header-cell').nth(0).locator('.grid__cell-content'),
		).toHaveText('first_name');
		await expect(
			preview.locator('.grid__row:nth-child(2) .grid__cell').nth(0).locator('.grid__cell-content'),
		).toHaveText('John');

		await page.click('.action__file-preview-drawer >> .drawer__close');
		await page.click('.action__file-confirm');

		// Identity column.
		const identity = page.locator('.action__transformation-identity-column');
		await identity.locator('sl-input').click();
		await identity.locator('sl-menu-item .schema-combobox-item__text', { hasText: 'email' }).click();

		// Mappings.
		let mappings = page.locator('.action__transformation');
		let email = mappings.locator('.combobox[data-id="email"]');
		await email.locator('sl-input').click();
		await email.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'email' }).click();
		let firstName = mappings.locator('.combobox[data-id="first_name"]');
		await firstName.locator('sl-input').click();
		await firstName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'first_name' }).click();
		let lastName = mappings.locator('.combobox[data-id="last_name"]');
		await lastName.locator('sl-input').click();
		await lastName.locator('sl-menu-item .schema-combobox-item__name', { hasText: 'last_name' }).click();

		const expectedBody = `
		{
			"target": "Users",
			"eventType": null,
			"name": "Import users",
			"enabled": true,
			"filter": null,
			"inSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "email",
						"type": {
							"name": "Text"
						},
						"description": ""
					},
					{
						"name": "first_name",
						"type": {
							"name": "Text"
						},
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"name": "Text"
						},
						"description": ""
					}
				]
			},
			"outSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "email",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "first_name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"name": "Text",
							"charLen": 300
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
			"identityColumn": "email",
			"lastChangeTimeColumn": "",
			"lastChangeTimeFormat": "",
			"compression": "",
			"format": "CSV",
			"formatSettings": {
				"Comma": ",",
				"Comment": "",
				"FieldsPerRecord": 0,
				"HasColumnNames": true,
				"LazyQuotes": false,
				"TrimLeadingSpace": false,
				"UseCRLF": false
			}
		}`;

		let isRequestDone = false;
		page.on('request', async (request) => {
			if (request.url().includes('/actions') && request.method() === 'POST') {
				isRequestDone = true;
				const body = JSON.parse(request.postData());
				let expected = JSON.parse(expectedBody);
				expected.connection = connectionID;
				deepCompareActionSchema(body, expected);
			}
		});

		let saveButton = page.locator('.action__header-save >> button');
		await saveButton.click();

		await expect(page.locator('.connection-actions__grid')).toBeVisible();
		expect(isRequestDone).toBe(true);

		await page.reload();

		await expect(page.locator('.connection-actions__grid')).toBeVisible();
	});
});

test(`Add "Export users" action on CSV file on Filesystem`, async ({ page }) => {
	await addFileSystemDestination(page, async (tempDir: string, connectionID: number) => {
		// Create a temporary file.
		const fileName = 'test.csv';

		const tempFilePath = join(tempDir, fileName);
		writeFile(tempFilePath, '', (err) => {
			if (err) throw err;
		});

		await page.goto(`${uiURL}connectors?role=Destination`);
		await page.click('a[href="/ui/connectors/file/CSV?role=Destination"]');

		await page.click('.file-connector__storage sl-select');
		await page.locator(`.file-connector__storage sl-select sl-option[value="${connectionID}"]`).click();

		let name = page.locator('.file-connector__action-types .list-tile__name', {
			hasText: 'Import users',
		});

		await expect(name).toBeVisible();

		let button = name.locator('..').locator('..').locator('sl-button');
		await button.click();
		await expect(page.locator('.action__header')).toBeVisible();

		// Filters.
		await fillUserActionFilters(page);

		// File
		await page.locator('.action__file-format').click();
		await page.locator('.action__file-format sl-option[value="CSV"]').click();

		await page.locator('.action__file-path >> input').fill(fileName);

		const expectedBody = `
		{
			"target": "Users",
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
							"acme@open2b.com",
							"test@open2b.com"
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
				"name": "Object",
				"properties": [
					{
						"name": "email",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "dummy_id",
						"type": {
							"name": "Text"
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "android",
						"type": {
							"name": "Object",
							"properties": [
								{
									"name": "id",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "idfa",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "push_token",
									"type": {
										"name": "Text"
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
							"name": "Object",
							"properties": [
								{
									"name": "id",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "idfa",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "push_token",
									"type": {
										"name": "Text"
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
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "last_name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "gender",
						"type": {
							"name": "Text"
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "food_preferences",
						"type": {
							"name": "Object",
							"properties": [
								{
									"name": "drink",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "fruit",
									"type": {
										"name": "Text"
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
							"name": "Array",
							"elementType": {
								"name": "Text",
								"charLen": 300
							}
						},
						"readOptional": true,
						"description": ""
					},
					{
						"name": "favorite_movie",
						"type": {
							"name": "Object",
							"properties": [
								{
									"name": "title",
									"type": {
										"name": "Text"
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "length",
									"type": {
										"name": "Float",
										"bitSize": 64
									},
									"readOptional": true,
									"description": ""
								},
								{
									"name": "soundtrack",
									"type": {
										"name": "Object",
										"properties": [
											{
												"name": "title",
												"type": {
													"name": "Text"
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "author",
												"type": {
													"name": "Text"
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "length",
												"type": {
													"name": "Float",
													"bitSize": 64
												},
												"readOptional": true,
												"description": ""
											},
											{
												"name": "genre",
												"type": {
													"name": "Text"
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
			"identityColumn": "",
			"lastChangeTimeColumn": "",
			"lastChangeTimeFormat": "",
			"compression": "",
			"orderBy": "email",
			"format": "CSV",
			"formatSettings": {
				"Comma": ",",
				"Comment": "",
				"FieldsPerRecord": 0,
				"HasColumnNames": false,
				"LazyQuotes": false,
				"TrimLeadingSpace": false,
				"UseCRLF": false
			}
		}`;

		let isRequestDone = false;
		page.on('request', async (request) => {
			if (request.url().includes('/actions') && request.method() === 'POST') {
				isRequestDone = true;
				const body = JSON.parse(request.postData());
				let expected = JSON.parse(expectedBody);
				expected.connection = connectionID;
				deepCompareActionSchema(body, expected);
			}
		});

		let saveButton = page.locator('.action__header-save >> button');
		await saveButton.click();

		await expect(page.locator('.connection-actions__grid')).toBeVisible();
		expect(isRequestDone).toBe(true);

		await page.reload();

		await expect(page.locator('.connection-actions__grid')).toBeVisible();
	});
});

test(`Add "Import events" action on Javascript`, async ({ page }) => {
	const id = await addJavascriptSource(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Import events',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Filters.
	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();

	let filters = page.locator('.action__filters-filter');

	await filters.nth(0).locator('.action__filters-property sl-input').click();
	await filters
		.nth(0)
		.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^type$/ })
		.click();
	await filters.nth(0).locator('.action__filters-operator sl-option[value="0"]').click(); // option is "is".
	await filters.nth(0).locator('.action__filters-value-input >> input').fill('track');

	const expectedBody = `
	{
		"target": "Events",
		"eventType": null,
		"name": "Import events",
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

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(body, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});

test(`Add "Import users" action on Javascript`, async ({ page }) => {
	const id = await addJavascriptSource(page);
	await page.goto(`${uiURL}connections/${id}/actions`);
	let name = page.locator('.connection-actions__no-action-action-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeVisible();

	let button = name.locator('..').locator('..').locator('sl-button');
	await button.click();
	await expect(page.locator('.action__header')).toBeVisible();

	// Filters.
	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();
	await page.locator('.action__filters-add-condition').click();

	let filters = page.locator('.action__filters-filter');

	await filters.nth(0).locator('.action__filters-property sl-input').click();
	await filters
		.nth(0)
		.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^type$/ })
		.click();
	await filters.nth(0).locator('.action__filters-operator sl-option[value="0"]').click(); // option is "is".
	await filters.nth(0).locator('.action__filters-value-input >> input').fill('identify');

	const expectedBody = `
	{
		"target": "Users",
		"eventType": null,
		"name": "Import users",
		"enabled": false,
		"filter": {
			"logical": "and",
			"conditions": [
				{
					"property": "type",
					"operator": "is",
					"values": [
						"identify"
					]
				}
			]
		},
		"inSchema": null,
		"outSchema": null,
		"transformation": null
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = JSON.parse(request.postData());
			let expected = JSON.parse(expectedBody);
			expected.connection = id;
			deepCompareActionSchema(body, expected);
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions__grid')).toBeVisible();
});
