import { test, expect, type Page } from '@playwright/test';
import Type, { ObjectType, Property } from '../src/lib/api/types/types';
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
		"requiredConsents": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{ "name": "email", "type": { "kind": "string" }, "displayName": "Email", "nullable": true },
				{ "name": "dummyId", "type": { "kind": "string" }, "displayName": "Dummy ID" },
				{ "name": "firstName", "type": { "kind": "string" }, "displayName": "First name", "nullable": true },
				{ "name": "lastName", "type": { "kind": "string" }, "displayName": "Last name", "nullable": true }
			]
		},
		"outSchema": {
			"kind": "object",
			"properties": [
				{ "name": "email", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true },
				{ "name": "dummy_id", "type": { "kind": "string" }, "readOptional": true },
				{ "name": "first_name", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true },
				{ "name": "last_name", "type": { "kind": "string", "maxLength": 300 }, "readOptional": true }
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
			"operator": "or",
			"rules": [
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
		"requiredConsents": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "dummy_id",
					"type": {
						"kind": "string"
					},
					"readOptional": true
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
					"displayName": "First name"
				},
				{
					"name": "lastName",
					"type": {
						"kind": "string"
					},
					"nullable": true,
					"displayName": "Last name"
				},
				{
					"name": "email",
					"type": {
						"kind": "string"
					},
					"nullable": true,
					"displayName": "Email"
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

test.describe('Matching semantics', () => {
	const openPipeline = async (page: Page, hasRequiredPhone = false) => {
		const id = await addDummyDestination(page);
		// API schemas omit the legacy role field.
		const createProperty = (name: string, type: Type, updateRequired = false): Property => ({
			name,
			type,
			prefilled: '',
			createRequired: false,
			updateRequired,
			readOptional: false,
			nullable: false,
			displayName: '',
			description: '',
		});
		const inputSchema: ObjectType = {
			kind: 'object',
			properties: [
				createProperty('firstName', { kind: 'string' }),
				createProperty('phone', { kind: 'string', semantic: 'phone' }),
				createProperty('country', { kind: 'string', semantic: 'country', format: 'alpha-3' }),
			],
		};
		const externalMatchingSchema: ObjectType = {
			kind: 'object',
			properties: [
				createProperty('phone', { kind: 'string', semantic: 'phone' }),
				createProperty('requiredPhone', { kind: 'string', semantic: 'phone' }),
				createProperty('country', { kind: 'string', semantic: 'country', format: 'alpha-3' }),
			],
		};
		const outputSchema: ObjectType = {
			kind: 'object',
			properties: [
				createProperty('firstName', { kind: 'string' }),
				createProperty('phone', { kind: 'string', semantic: 'email' }),
				createProperty('requiredPhone', { kind: 'string', semantic: 'email' }, hasRequiredPhone),
				createProperty('country', { kind: 'string', semantic: 'country', format: 'alpha-3' }),
			],
		};
		await page.route(`**/connections/${id}/pipelines/schemas/User`, async (route) => {
			await route.fulfill({
				json: {
					in: inputSchema,
					out: outputSchema,
					matchings: { internal: inputSchema, external: externalMatchingSchema },
				},
			});
		});
		await page.goto(`${adminURL}/connections/${id}/pipelines`);
		await page
			.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
				hasText: 'Export customers',
			})
			.locator('..')
			.locator('..')
			.locator('sl-button')
			.click();
		await expect(page.locator('.pipeline__header')).toBeAttached();
	};

	test('Refresh matching choices when the export mode changes', async ({ page }) => {
		await openPipeline(page, true);
		const exportMode = page.locator('.pipeline__export-mode sl-select');
		await exportMode.locator('[part="display-input"]').click();
		await exportMode.locator('sl-option[value="UpdateOnly"]').click();
		await expect(exportMode.locator('sl-option[value="UpdateOnly"]')).toBeHidden();
		const matching = page.locator('.pipeline__matching-properties [data-id="out"]');
		await matching.locator('input').click();
		await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['phone', 'country']);
		await matching.locator('.schema-combobox-item__name', { hasText: /^phone$/ }).click();
		await page.keyboard.press('Escape');

		for (const mode of ['CreateOnly', 'CreateOrUpdate']) {
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator(`sl-option[value="${mode}"]`).click();
			await expect(exportMode.locator(`sl-option[value="${mode}"]`)).toBeHidden();
			await expect(matching.locator('.combobox-input__error')).toContainText('does not exist');
			await matching.locator('input').fill('');
			await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['country']);
			await matching.locator('input').fill('phone');
			await page.keyboard.press('Escape');
		}

		await exportMode.locator('[part="display-input"]').click();
		await exportMode.locator('sl-option[value="UpdateOnly"]').click();
		await expect(exportMode.locator('sl-option[value="UpdateOnly"]')).toBeHidden();
		await expect(matching.locator('.combobox-input__error')).toHaveCount(0);
		await matching.locator('input').fill('');
		await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['phone', 'country']);
	});

	for (const scenario of [
		{
			property: 'country',
			mode: 'CreateOrUpdate',
			type: { kind: 'string', semantic: 'country', format: 'alpha-3' },
		},
		{ property: 'phone', mode: 'UpdateOnly', type: { kind: 'string', semantic: 'phone' } },
	]) {
		test(`Preserve ${scenario.property} semantics when saving in ${scenario.mode} mode`, async ({ page }) => {
			await openPipeline(page);
			const exportMode = page.locator('.pipeline__export-mode sl-select');
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator(`sl-option[value="${scenario.mode}"]`).click();
			await expect(exportMode.locator(`sl-option[value="${scenario.mode}"]`)).toBeHidden();
			const matching = page.locator('.pipeline__matching-properties');
			for (const side of ['in', 'out']) {
				await matching.locator(`[data-id="${side}"] input`).fill(scenario.property);
				await page.keyboard.press('Escape');
			}
			await page.locator('.pipeline__transformation .combobox[data-id="firstName"] input').fill('firstName');
			await page.keyboard.press('Escape');
			await page.route('**/v1/pipelines', async (route) => {
				await route.fulfill({ json: { id: 'semantic-pipeline' } });
			});
			const requestPromise = page.waitForRequest(
				(request) => request.url().endsWith('/v1/pipelines') && request.method() === 'POST',
			);
			await page.locator('.pipeline__header-save button').click();
			const body = (await requestPromise).postDataJSON();
			expect(body.exportMode).toBe(scenario.mode);
			expect(body.matching).toEqual({ in: scenario.property, out: scenario.property });
			for (const schema of [body.inSchema, body.outSchema]) {
				const property = schema.properties.find((property: Property) => property.name === scenario.property);
				expect(property?.type).toEqual(scenario.type);
			}
			expect(body.transformation.mapping).toEqual({ firstName: 'firstName' });
			await expect(page.locator('.pipeline__header')).toHaveCount(0);
		});
	}

	for (const scenario of [
		{
			in: 'firstName',
			out: 'phone',
			hasRequiredPhone: false,
			error: 'Input and output matching properties do not have the same semantic and options',
		},
		{
			in: 'phone',
			out: 'requiredPhone',
			hasRequiredPhone: true,
			error: 'Matching property "requiredPhone" must have the same semantic and options in the source and destination schemas',
		},
	]) {
		test(`Reject incompatible ${scenario.in}/${scenario.out} semantics before submitting a pipeline`, async ({
			page,
		}) => {
			await openPipeline(page, scenario.hasRequiredPhone);
			const exportMode = page.locator('.pipeline__export-mode sl-select');
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator('sl-option[value="UpdateOnly"]').click();
			await expect(exportMode.locator('sl-option[value="UpdateOnly"]')).toBeHidden();
			const matching = page.locator('.pipeline__matching-properties');
			await matching.locator('[data-id="in"] input').fill(scenario.in);
			await page.keyboard.press('Escape');
			await matching.locator('[data-id="out"] input').fill(scenario.out);
			await page.keyboard.press('Escape');
			await page.locator('.pipeline__transformation .combobox[data-id="firstName"] input').fill('firstName');
			await page.keyboard.press('Escape');
			let submissionCount = 0;
			await page.route('**/v1/pipelines', async (route) => {
				submissionCount++;
				await route.fulfill({ status: 500 });
			});
			await page.locator('.pipeline__header-save button').click();
			await expect(page.locator('sl-alert[variant="danger"]')).toContainText(scenario.error);
			await expect(page.locator('.pipeline__header-save')).not.toHaveAttribute('loading');
			expect(submissionCount).toBe(0);
			await expect(page.locator('.pipeline__header')).toBeAttached();
		});
	}
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
		"requiredConsents": null,
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
					"displayName": "Email"
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
		"requiredConsents": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true
				},
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true
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
					"readOptional": true
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
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
			"operator": "or",
			"rules": [
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
		"requiredConsents": null,
		"inSchema": {
			"kind": "object",
			"properties": [
				{
					"name": "email",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"readOptional": true
				},
				{
					"name": "dummy_id",
					"type": {
						"kind": "string"
					},
					"readOptional": true
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
					"nullable": false
				},
				{
					"name": "first_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true
				},
				{
					"name": "last_name",
					"type": {
						"kind": "string",
						"maxLength": 300
					},
					"nullable": true
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
			"requiredConsents": null,
			"inSchema": {
				"kind": "object",
				"properties": [
					{
						"name": "email",
						"type": {
							"kind": "string"
						},
						"displayName": " email"
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string"
						},
						"displayName": "first_name"
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string"
						},
						"displayName": " last_name"
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
						"readOptional": true
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true
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
				"operator": "or",
				"rules": [
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
			"requiredConsents": null,
			"inSchema": {
				"kind": "object",
				"properties": [
					{
						"name": "email",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true
					},
					{
						"name": "dummy_id",
						"type": {
							"kind": "string"
						},
						"readOptional": true
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
									"readOptional": true
								},
								{
									"name": "idfa",
									"type": {
										"kind": "string"
									},
									"readOptional": true
								},
								{
									"name": "push_token",
									"type": {
										"kind": "string"
									},
									"readOptional": true
								}
							]
						},
						"readOptional": true
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
									"readOptional": true
								},
								{
									"name": "idfa",
									"type": {
										"kind": "string"
									},
									"readOptional": true
								},
								{
									"name": "push_token",
									"type": {
										"kind": "string"
									},
									"readOptional": true
								}
							]
						},
						"readOptional": true
					},
					{
						"name": "first_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true
					},
					{
						"name": "last_name",
						"type": {
							"kind": "string",
							"maxLength": 300
						},
						"readOptional": true
					},
					{
						"name": "gender",
						"type": {
							"kind": "string"
						},
						"readOptional": true
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
									"readOptional": true
								},
								{
									"name": "fruit",
									"type": {
										"kind": "string"
									},
									"readOptional": true
								}
							]
						},
						"readOptional": true
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
						"readOptional": true
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
									"readOptional": true
								},
								{
									"name": "length",
									"type": {
										"kind": "float",
										"bitSize": 64
									},
									"readOptional": true
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
												"readOptional": true
											},
											{
												"name": "author",
												"type": {
													"kind": "string"
												},
												"readOptional": true
											},
											{
												"name": "length",
												"type": {
													"kind": "float",
													"bitSize": 64
												},
												"readOptional": true
											},
											{
												"name": "genre",
												"type": {
													"kind": "string"
												},
												"readOptional": true
											}
										]
									},
									"readOptional": true
								}
							]
						},
						"readOptional": true
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

	const filters = page.locator('.pipeline__filters-filter');

	await filters.nth(0).locator('.pipeline__filters-property sl-input').click();
	await filters
		.nth(0)
		.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^type$/ })
		.click();
	await filters.nth(0).locator('.pipeline__filters-operator sl-option[value="0"]').click(); // option is "is".

	await filters.nth(0).locator('.pipeline__filters-value-input sl-option[value="track"]').click(); // value select should open automatically after selecting the operator

	const jsonCondition = filters.nth(1);
	const jsonPropertyInput = jsonCondition.locator('.pipeline__filters-property sl-input');
	await jsonPropertyInput.click();
	await jsonCondition.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^traits$/ }).click();
	const jsonPathInput = jsonCondition.locator('.pipeline__filters-path >> input');
	await jsonPathInput.fill('email');
	const jsonOperatorSelect = jsonCondition.locator('.pipeline__filters-operator');
	await expect(jsonOperatorSelect).toHaveJSProperty('value', '0');
	const jsonValueInput = jsonCondition.locator('.pipeline__filters-value-input');
	await jsonValueInput.locator('input').fill('a@example.com');

	await jsonPropertyInput.click();
	await jsonCondition.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^properties$/ }).click();
	await expect(jsonPropertyInput).toHaveJSProperty('value', 'properties');
	await expect(jsonPathInput).toHaveValue('email');
	await expect(jsonOperatorSelect).toHaveJSProperty('value', '0');
	await expect(jsonValueInput).toHaveJSProperty('value', 'a@example.com');

	await jsonPropertyInput.click();
	await jsonCondition.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^traits$/ }).click();

	for (const operator of [
		{ name: 'exists', index: 24 },
		{ name: 'does not exist', index: 25 },
	]) {
		const condition = filters.nth(1);
		await condition.locator('.pipeline__filters-property sl-input').click();
		await condition.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^traits$/ }).click();
		const pathInput = condition.locator('.pipeline__filters-path >> input');
		await pathInput.fill('email');
		const operatorSelect = condition.locator('.pipeline__filters-operator');
		const operatorOption = operatorSelect.locator(`sl-option[value="${operator.index}"]`);
		await expect(operatorOption).toHaveText(operator.name);
		await operatorSelect.click();
		await operatorOption.click();
		await expect(operatorSelect).toHaveJSProperty('value', String(operator.index));
		await pathInput.fill('');
		await expect(operatorSelect).toHaveJSProperty('value', '');
		await condition.locator('.pipeline__filters-remove-condition').click();
	}

	const expectedBody = `
	{
		"target": "Event",
		"eventType": null,
		"name": "Import events into warehouse",
		"enabled": false,
		"filter": {
			"operator": "and",
			"rules": [
				{
					"property": "type",
					"operator": "is",
					"values": [
						"track"
					]
				}
			]
		},
		"requiredConsents": null,
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
			"operator": "or",
			"rules": [
				{
					"property": "type",
					"operator": "is",
					"values": [
						"identify"
					]
				},
				{
					"property": "traits",
					"operator": "is not empty"
				}
			]
		},
		"requiredConsents": null,
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
