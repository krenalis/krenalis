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
		"action": {
			"name": "Import users",
			"enabled": false,
			"filter": null,
			"inSchema": {
				"name": "Object",
				"properties": [
					{ "name": "email", "label": "", "type": { "name": "Text" }, "note": "" },
					{ "name": "dummyId", "label": "", "type": { "name": "Text" }, "note": "" },
					{ "name": "firstName", "label": "", "type": { "name": "Text" }, "note": "" },
					{ "name": "lastName", "label": "", "type": { "name": "Text" }, "note": "" }
				]
			},
			"outSchema": {
				"name": "Object",
				"properties": [
					{ "name": "email", "label": "Email", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "note": "" },
					{ "name": "dummy_id", "label": "Dummy ID", "type": { "name": "Text" }, "readOptional": true, "note": "" },
					{ "name": "first_name", "label": "First name", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "note": "" },
					{ "name": "last_name", "label": "Last name", "type": { "name": "Text", "charLen": 300 }, "readOptional": true, "note": "" }
				]
			},
			"transformation": {
				"mapping": {
					"email": "email",
					"dummy_id": "dummyId",
					"first_name": "firstName",
					"last_name": "lastName"
				}
			}
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
	await exportMode.locator('sl-option[value="CreateOnly"]').click();

	// Matching properties.
	let matchingProperties = page.locator('.action__matching-properties');
	await matchingProperties.locator('[data-id="internal"] sl-input >> input').click();
	await matchingProperties
		.locator('[data-id="internal"] sl-menu-item .schema-combobox-item__name', { hasText: 'email' })
		.click();
	await matchingProperties.locator('[data-id="external"] sl-input >> input').click();
	await matchingProperties
		.locator('[data-id="external"] sl-menu-item .schema-combobox-item__text', { hasText: 'email' })
		.click();
	// Selected external matching property should not be visible in the mapping.
	await expect(
		page.locator('.action__transformation-mappings .action__transformation-output-property >> input', {
			hasText: 'email',
		}),
	).not.toBeVisible();

	// Export on duplicated users.
	await page.locator('.action__export-on-duplicated sl-checkbox').click();

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
		"action": {
			"name": "Export users",
			"enabled": false,
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
						"label": "First name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "last_name",
						"label": "Last name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "email",
						"label": "Email",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "dummy_id",
						"label": "Dummy ID",
						"type": {
							"name": "Text"
						},
						"readOptional": true,
						"note": ""
					}
				]
			},
			"outSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "firstName",
						"label": "",
						"type": {
							"name": "Text"
						},
						"note": ""
					},
					{
						"name": "lastName",
						"label": "",
						"type": {
							"name": "Text"
						},
						"note": ""
					},
					{
						"name": "email",
						"label": "",
						"type": {
							"name": "Text"
						},
						"note": ""
					}
				]
			},
			"transformation": {
				"mapping": {
					"firstName": "first_name",
					"lastName": "last_name"
				}
			},
			"exportMode": "CreateOnly",
			"matchingProperties": {
				"internal": "email",
				"external": {
					"name": "email",
					"label": "",
					"type": {
						"name": "Text"
					},
					"note": ""
				}
			},
			"exportOnDuplicatedUsers": true
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
		"action": {
			"name": "Send Add to Cart",
			"enabled": false,
			"filter": null,
			"inSchema": null,
			"outSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "email",
						"label": "",
						"type": {
							"name": "Text"
						},
						"createRequired": true,
						"note": ""
					}
				]
			},
			"transformation": {
				"mapping": {
					"email": "traits"
				}
			}
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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

	// Identity property.
	const identity = page.locator('.action__transformation-identity-property');
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
		"action": {
			"name": "Import users",
			"enabled": false,
			"filter": null,
			"inSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "first_name",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": true,
						"note": ""
					},
					{
						"name": "last_name",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": true,
						"note": ""
					},
					{
						"name": "email",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": true,
						"note": ""
					}
				]
			},
			"outSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "first_name",
						"label": "First name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "last_name",
						"label": "Last name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
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
			"identityProperty": "email",
			"lastChangeTimeProperty": "",
			"lastChangeTimeFormat": ""
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
		"action": {
			"name": "Export users",
			"enabled": false,
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
						"label": "Email",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "first_name",
						"label": "First name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "last_name",
						"label": "Last name",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"readOptional": true,
						"note": ""
					},
					{
						"name": "dummy_id",
						"label": "Dummy ID",
						"type": {
							"name": "Text"
						},
						"readOptional": true,
						"note": ""
					}
				]
			},
			"outSchema": {
				"name": "Object",
				"properties": [
					{
						"name": "email",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": false,
						"note": "",
						"updateRequired": true
					},
					{
						"name": "first_name",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": true,
						"note": "",
						"updateRequired": true
					},
					{
						"name": "last_name",
						"label": "",
						"type": {
							"name": "Text",
							"charLen": 300
						},
						"nullable": true,
						"note": "",
						"updateRequired": true
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
			"tableKeyProperty": "email"
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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

		// Identity property.
		const identity = page.locator('.action__transformation-identity-property');
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
			"action": {
				"name": "Import users",
				"enabled": false,
				"filter": null,
				"inSchema": {
					"name": "Object",
					"properties": [
						{
							"name": "email",
							"label": " email",
							"type": {
								"name": "Text"
							},
							"note": ""
						},
						{
							"name": "first_name",
							"label": "",
							"type": {
								"name": "Text"
							},
							"note": ""
						},
						{
							"name": "last_name",
							"label": " last_name",
							"type": {
								"name": "Text"
							},
							"note": ""
						}
					]
				},
				"outSchema": {
					"name": "Object",
					"properties": [
						{
							"name": "email",
							"label": "Email",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "first_name",
							"label": "First name",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "last_name",
							"label": "Last name",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
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
				"identityProperty": "email",
				"lastChangeTimeProperty": "",
				"lastChangeTimeFormat": "",
				"compression": "",
				"connector": "CSV",
				"uiValues": {
					"Comma": ",",
					"Comment": "",
					"FieldsPerRecord": 0,
					"HasColumnNames": true,
					"LazyQuotes": false,
					"TrimLeadingSpace": false,
					"UseCRLF": false
				}
			}
		}`;

		let isRequestDone = false;
		page.on('request', async (request) => {
			if (request.url().includes('/actions') && request.method() === 'POST') {
				isRequestDone = true;
				const body = request.postData();
				deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
			}
		});

		let saveButton = page.locator('.action__header-save >> button');
		await saveButton.click();

		await expect(page.locator('.connection-actions .grid')).toBeVisible();
		expect(isRequestDone).toBe(true);

		await page.reload();

		await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
		await page.locator('.action__file-connector').click();
		await page.locator('.action__file-connector sl-option[value="CSV"]').click();

		await page.locator('.action__file-path >> input').fill(fileName);

		const expectedBody = `
		{
			"target": "Users",
			"eventType": null,
			"action": {
				"name": "Export users",
				"enabled": false,
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
							"label": "Email",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "dummy_id",
							"label": "Dummy ID",
							"type": {
								"name": "Text"
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "android",
							"label": "Android",
							"type": {
								"name": "Object",
								"properties": [
									{
										"name": "id",
										"label": "ID",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "idfa",
										"label": "IDFA",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "push_token",
										"label": "Push token",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									}
								]
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "ios",
							"label": "IOS",
							"type": {
								"name": "Object",
								"properties": [
									{
										"name": "id",
										"label": "ID",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "idfa",
										"label": "IDFA",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "push_token",
										"label": "Push token",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									}
								]
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "first_name",
							"label": "First name",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "last_name",
							"label": "Last name",
							"type": {
								"name": "Text",
								"charLen": 300
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "gender",
							"label": "Gender",
							"type": {
								"name": "Text"
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "food_preferences",
							"label": "Food preferences",
							"type": {
								"name": "Object",
								"properties": [
									{
										"name": "drink",
										"label": "Drink",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "fruit",
										"label": "Fruit",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									}
								]
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "phone_numbers",
							"label": "Phone numbers",
							"type": {
								"name": "Array",
								"elementType": {
									"name": "Text",
									"charLen": 300
								}
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "home_ip_address",
							"label": "Home IP Address",
							"type": {
								"name": "Inet"
							},
							"readOptional": true,
							"note": ""
						},
						{
							"name": "favorite_movie",
							"label": "Favorite movie",
							"type": {
								"name": "Object",
								"properties": [
									{
										"name": "title",
										"label": "Title",
										"type": {
											"name": "Text"
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "length",
										"label": "Length",
										"type": {
											"name": "Float",
											"bitSize": 64
										},
										"readOptional": true,
										"note": ""
									},
									{
										"name": "soundtrack",
										"label": "Soundtrack",
										"type": {
											"name": "Object",
											"properties": [
												{
													"name": "title",
													"label": "Title",
													"type": {
														"name": "Text"
													},
													"readOptional": true,
													"note": ""
												},
												{
													"name": "author",
													"label": "Author",
													"type": {
														"name": "Text"
													},
													"readOptional": true,
													"note": ""
												},
												{
													"name": "length",
													"label": "Length",
													"type": {
														"name": "Float",
														"bitSize": 64
													},
													"readOptional": true,
													"note": ""
												},
												{
													"name": "genre",
													"label": "Genre",
													"type": {
														"name": "Text"
													},
													"readOptional": true,
													"note": ""
												}
											]
										},
										"readOptional": true,
										"note": ""
									}
								]
							},
							"readOptional": true,
							"note": ""
						}
					]
				},
				"outSchema": null,
				"transformation": {},
				"path": "test.csv",
				"sheet": null,
				"fileOrderingPropertyPath": "email",
				"identityProperty": "",
				"lastChangeTimeProperty": "",
				"lastChangeTimeFormat": "",
				"compression": "",
				"connector": "CSV",
				"uiValues": {
					"Comma": ",",
					"Comment": "",
					"FieldsPerRecord": 0,
					"HasColumnNames": false,
					"LazyQuotes": false,
					"TrimLeadingSpace": false,
					"UseCRLF": false
				}
			}
		}`;

		let isRequestDone = false;
		page.on('request', async (request) => {
			if (request.url().includes('/actions') && request.method() === 'POST') {
				isRequestDone = true;
				const body = request.postData();
				deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
			}
		});

		let saveButton = page.locator('.action__header-save >> button');
		await saveButton.click();

		await expect(page.locator('.connection-actions .grid')).toBeVisible();
		expect(isRequestDone).toBe(true);

		await page.reload();

		await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
		"action": {
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
			"transformation": {}
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
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
		"action": {
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
			"transformation": {}
		}
	}
	`;

	let isRequestDone = false;
	page.on('request', async (request) => {
		if (request.url().includes('/actions') && request.method() === 'POST') {
			isRequestDone = true;
			const body = request.postData();
			deepCompareActionSchema(JSON.parse(body), JSON.parse(expectedBody));
		}
	});

	let saveButton = page.locator('.action__header-save >> button');
	await saveButton.click();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
	expect(isRequestDone).toBe(true);

	await page.reload();

	await expect(page.locator('.connection-actions .grid')).toBeVisible();
});
