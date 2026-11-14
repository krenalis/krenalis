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
	loadSyncToken,
	syncTokenHeaders,
} from './utils';
import { ConsentPurpose } from '../src/lib/api/types/workspace';
import { join } from 'path';
import { writeFile } from 'fs';

test.beforeEach(async ({ page }) => {
	await login(page);
	await loadSyncToken(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Remove the only Pipeline condition without an Admin error and restore focus`, async ({ page }) => {
	const pageErrors: Error[] = [];
	page.on('pageerror', (error) => pageErrors.push(error));
	const id = await addDummyDestination(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	const name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Export customers',
	});
	await expect(name).toBeAttached();
	await name.locator('..').locator('..').locator('wa-button').click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	const filters = page.locator('.pipeline__filters');
	const addFilter = filters.getByRole('button', { name: 'Add filter', exact: true });
	await addFilter.click();

	const removeCondition = filters.getByRole('button', { name: 'Remove condition 1', exact: true });
	const tooltip = filters.locator('.filter-editor__remove-condition-wrapper wa-tooltip');
	await expect(removeCondition).toHaveJSProperty('tagName', 'BUTTON');
	await expect(removeCondition).toBeEnabled();
	await removeCondition.focus();
	await expect(tooltip).toHaveJSProperty('open', true);
	await page.keyboard.press('Escape');
	await expect(tooltip).toHaveJSProperty('open', false);
	await removeCondition.hover();
	await expect(tooltip).toHaveJSProperty('open', true);
	await removeCondition.click();

	await expect(addFilter).toBeFocused();
	await expect(filters.locator('.filter-editor__announcement')).toHaveText('Condition removed');
	expect(pageErrors).toEqual([]);
});

test(`Preserve nested Pipeline groups when removing a preceding condition and restoring focus`, async ({ page }) => {
	const pageErrors: Error[] = [];
	page.on('pageerror', (error) => pageErrors.push(error));
	const id = await addDummyDestination(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	const name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Export customers',
	});
	await name.locator('..').locator('..').locator('wa-button').click();
	const filters = page.locator('.pipeline__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();

	const root = filters.locator('.filter-editor__group--root');
	await root.locator(':scope > .filter-editor__group-actions .filter-editor__add-group').click();
	const nested = root.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule--group > .filter-editor__rule-content > .filter-editor__group',
	);
	await nested.locator(':scope > .filter-editor__group-actions .filter-editor__add-group').click();
	const child = nested.locator(
		':scope > .filter-editor__group-rules > .filter-editor__rule--group > .filter-editor__rule-content > .filter-editor__group',
	);
	await expect(child).toHaveCount(1);
	await nested.getByRole('button', { name: 'Remove condition 2', exact: true }).click();
	await expect(child.locator('.filter-editor__property wa-input')).toBeFocused();
	await expect(filters.locator('.filter-editor__filter')).toHaveCount(2);
	await expect(filters.locator('.filter-editor__announcement')).toHaveText('Condition removed');

	await child.locator(':scope > .filter-editor__group-header .filter-editor__remove-group').click();
	await expect(child).toHaveCount(0);
	await expect(nested.locator('.filter-editor__property wa-input')).toHaveJSProperty('value', '');
	await expect(nested.getByRole('button', { name: 'Remove condition 2', exact: true })).toBeDisabled();
	expect(pageErrors).toEqual([]);
});

test(`Add "Import customers" pipeline on Dummy`, async ({ page }) => {
	const id = await addDummySource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	let name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import Dummy customers',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	let email = page.locator('.combobox[data-id="email"]');
	await email.locator('wa-input').click();
	await email.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let dummyId = page.locator('.combobox[data-id="dummy_id"]');
	await dummyId.locator('wa-input').click();
	await dummyId.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'dummyId' }).click();
	await page.keyboard.press('Escape');

	let firstName = page.locator('.combobox[data-id="first_name"]');
	await firstName.locator('wa-input').click();
	await firstName.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'firstName' }).click();
	await page.keyboard.press('Escape');

	let lastName = page.locator('.combobox[data-id="last_name"]');
	await lastName.locator('wa-input').click();
	await lastName.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'lastName' }).click();
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

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// Export mode.
	let exportMode = page.locator('.pipeline__export-mode');
	await exportMode.locator('wa-select').click();
	await exportMode.locator('wa-option[value="UpdateOnly"]').click();

	// Matching.
	let matching = page.locator('.pipeline__matching-properties');
	await matching.locator('[data-id="in"] wa-input >> input').click();
	await matching
		.locator('[data-id="in"] .combobox-list__item .schema-combobox-item__name', { hasText: 'email' })
		.click();
	await matching.locator('[data-id="out"] wa-input >> input').click();
	await matching
		.locator('[data-id="out"] .combobox-list__item .schema-combobox-item__text', { hasText: 'email' })
		.click();
	// Selected out matching property should not be visible in the
	// mapping.
	await expect(
		page.locator('.pipeline__transformation-mappings .pipeline__transformation-output-property >> input', {
			hasText: 'email',
		}),
	).not.toBeAttached();

	// Update on duplicates.
	await page.locator('.pipeline__update-on-duplicates wa-checkbox').click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let firstName = mappings.locator('.combobox[data-id="firstName"]');
	await firstName.locator('wa-input').click();
	await firstName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="lastName"]');
	await lastName.locator('wa-input').click();
	await lastName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'last_name' }).click();
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
				createProperty('phone', { kind: 'string', semantics: 'phone' }),
				createProperty('country', { kind: 'string', semantics: 'country', format: 'alpha-3' }),
			],
		};
		const externalMatchingSchema: ObjectType = {
			kind: 'object',
			properties: [
				createProperty('phone', { kind: 'string', semantics: 'phone' }),
				createProperty('requiredPhone', { kind: 'string', semantics: 'phone' }),
				createProperty('country', { kind: 'string', semantics: 'country', format: 'alpha-3' }),
			],
		};
		const outputSchema: ObjectType = {
			kind: 'object',
			properties: [
				createProperty('firstName', { kind: 'string' }),
				createProperty('phone', { kind: 'string', semantics: 'email' }),
				createProperty('requiredPhone', { kind: 'string', semantics: 'email' }, hasRequiredPhone),
				createProperty('country', { kind: 'string', semantics: 'country', format: 'alpha-3' }),
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
			.locator('wa-button')
			.click();
		await expect(page.locator('.pipeline__header')).toBeAttached();
	};

	test('Refresh matching choices when the export mode changes', async ({ page }) => {
		await openPipeline(page, true);
		const exportMode = page.locator('.pipeline__export-mode wa-select');
		await exportMode.locator('[part="display-input"]').click();
		await exportMode.locator('wa-option[value="UpdateOnly"]').click();
		await expect(exportMode.locator('wa-option[value="UpdateOnly"]')).toBeHidden();
		const matching = page.locator('.pipeline__matching-properties [data-id="out"]');
		await matching.locator('input').click();
		await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['phone', 'country']);
		await matching.locator('.schema-combobox-item__name', { hasText: /^phone$/ }).click();
		await page.keyboard.press('Escape');

		for (const mode of ['CreateOnly', 'CreateOrUpdate']) {
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator(`wa-option[value="${mode}"]`).click();
			await expect(exportMode.locator(`wa-option[value="${mode}"]`)).toBeHidden();
			await expect(matching.locator('.combobox-input__error')).toContainText('does not exist');
			await matching.locator('input').fill('');
			await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['country']);
			await matching.locator('input').fill('phone');
			await page.keyboard.press('Escape');
		}

		await exportMode.locator('[part="display-input"]').click();
		await exportMode.locator('wa-option[value="UpdateOnly"]').click();
		await expect(exportMode.locator('wa-option[value="UpdateOnly"]')).toBeHidden();
		await expect(matching.locator('.combobox-input__error')).toHaveCount(0);
		await matching.locator('input').fill('');
		await expect(matching.locator('.schema-combobox-item__name')).toHaveText(['phone', 'country']);
	});

	const countryType = { kind: 'string', semantics: 'country', format: 'alpha-3' };
	const phoneType = { kind: 'string', semantics: 'phone' };

	for (const scenario of [
		{
			mode: 'CreateOrUpdate',
			in: { property: 'country', type: countryType },
			out: { property: 'country', type: countryType },
		},
		{
			mode: 'UpdateOnly',
			in: { property: 'phone', type: phoneType },
			out: { property: 'phone', type: phoneType },
		},
		{
			mode: 'UpdateOnly',
			in: { property: 'firstName', type: { kind: 'string' } },
			out: { property: 'phone', type: phoneType },
		},
	]) {
		test(`Save compatible ${scenario.in.property}/${scenario.out.property} semantics in ${scenario.mode} mode`, async ({
			page,
		}) => {
			await openPipeline(page);
			const exportMode = page.locator('.pipeline__export-mode wa-select');
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator(`wa-option[value="${scenario.mode}"]`).click();
			await expect(exportMode.locator(`wa-option[value="${scenario.mode}"]`)).toBeHidden();
			const matching = page.locator('.pipeline__matching-properties');
			for (const side of ['in', 'out'] as const) {
				await matching.locator(`[data-id="${side}"] input`).fill(scenario[side].property);
				await page.keyboard.press('Escape');
			}
			await page.locator('.pipeline__transformation .combobox[data-id="firstName"] input').fill('firstName');
			await page.keyboard.press('Escape');
			await page.route('**/v1/pipelines', async (route) => {
				await route.fulfill({ json: { id: 'semantics-pipeline' }, headers: syncTokenHeaders() });
			});
			const requestPromise = page.waitForRequest(
				(request) => request.url().endsWith('/v1/pipelines') && request.method() === 'POST',
			);
			await page.locator('.pipeline__header-save button').click();
			const body = (await requestPromise).postDataJSON();
			expect(body.exportMode).toBe(scenario.mode);
			expect(body.matching).toEqual({ in: scenario.in.property, out: scenario.out.property });
			const inProperty = body.inSchema.properties.find(
				(property: Property) => property.name === scenario.in.property,
			);
			expect(inProperty?.type).toEqual(scenario.in.type);
			const outProperty = body.outSchema.properties.find(
				(property: Property) => property.name === scenario.out.property,
			);
			expect(outProperty?.type).toEqual(scenario.out.type);
			expect(body.transformation.mapping).toEqual({ firstName: 'firstName' });
			await expect(page.locator('.pipeline__header')).toHaveCount(0);
		});
	}

	for (const scenario of [
		{
			in: 'country',
			out: 'phone',
			hasRequiredPhone: false,
			error: 'Input and output matching properties do not have the same semantics and options',
		},
		{
			in: 'phone',
			out: 'requiredPhone',
			hasRequiredPhone: true,
			error: 'Matching property "requiredPhone" must have the same semantics and options in the source and destination schemas',
		},
	]) {
		test(`Reject incompatible ${scenario.in}/${scenario.out} semantics before submitting a pipeline`, async ({
			page,
		}) => {
			await openPipeline(page, scenario.hasRequiredPhone);
			const exportMode = page.locator('.pipeline__export-mode wa-select');
			await exportMode.locator('[part="display-input"]').click();
			await exportMode.locator('wa-option[value="UpdateOnly"]').click();
			await expect(exportMode.locator('wa-option[value="UpdateOnly"]')).toBeHidden();
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
			await expect(page.locator('.toast')).toContainText(scenario.error);
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

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('wa-input').click();
	await email.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'traits' }).click();
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

	let button = name.locator('..').locator('..').locator('wa-button');
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
	await identity.locator('wa-input').click();
	await identity.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('wa-input').click();
	await firstName.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('wa-input').click();
	await lastName.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'last_name' }).click();
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

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// Table.
	await page.locator('.pipeline__destination_table wa-input >> input').fill('users');
	await page.locator('.pipeline__destination_table wa-button').click();

	await expect(page.locator('.pipeline__destination_table-key-section')).toBeAttached();
	await expect(page.locator('.pipeline__transformation')).toBeAttached();

	// Table key.
	let tableKey = page.locator('.pipeline__destination_table-key-property');
	await tableKey.locator('wa-input >> input').click();
	await tableKey.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('wa-input').click();
	await email.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('wa-input').click();
	await firstName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('wa-input').click();
	await lastName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'last_name' }).click();
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

	await page.click('.file-connector__storage wa-select');
	await page.locator(`.file-connector__storage wa-select wa-option[value="${connectionID}"]`).click();

	let name = page.locator('.file-connector__pipeline-types .list-tile__name', {
		hasText: 'Import users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// File
	await page.locator('.pipeline__file-path >> input').fill(fileName);
	await page.click('.connector-ui .connector-checkbox:last-child wa-checkbox');

	await page.click('.pipeline__file-confirm');

	// Identity column.
	const identity = page.locator('.pipeline__transformation-identity-column');
	await identity.locator('wa-input').click();
	await identity.locator('.combobox-list__item .schema-combobox-item__text', { hasText: 'email' }).click();

	// Mappings.
	let mappings = page.locator('.pipeline__transformation');
	let email = mappings.locator('.combobox[data-id="email"]');
	await email.locator('wa-input').click();
	await email.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'email' }).click();
	await page.keyboard.press('Escape');

	let firstName = mappings.locator('.combobox[data-id="first_name"]');
	await firstName.locator('wa-input').click();
	await firstName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'first_name' }).click();
	await page.keyboard.press('Escape');

	let lastName = mappings.locator('.combobox[data-id="last_name"]');
	await lastName.locator('wa-input').click();
	await lastName.locator('.combobox-list__item .schema-combobox-item__name', { hasText: 'last_name' }).click();
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

	await page.click('.file-connector__storage wa-select');
	await page.locator(`.file-connector__storage wa-select wa-option[value="${connectionID}"]`).click();

	let name = page.locator('.file-connector__pipeline-types .list-tile__name', {
		hasText: 'Export users',
	});

	await expect(name).toBeAttached();

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await fillUserPipelineFilters(page);

	// File
	await page.locator('.pipeline__file-format').click();
	await page.locator('.pipeline__file-format wa-option[value="csv"]').click();

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

	let button = name.locator('..').locator('..').locator('wa-button');
	await button.click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	// Filters.
	await page.locator('.pipeline__filters-add-condition').click();
	await page.locator('.filter-editor__add-condition').click();
	await page.locator('.filter-editor__add-condition').click();

	const filters = page.locator('.filter-editor__filter');

	await filters.nth(0).locator('.filter-editor__property wa-input').click();
	await filters
		.nth(0)
		.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Type$/ })
		.click();
	await filters.nth(0).locator('.filter-editor__operator wa-option[value="0"]').click(); // option is "is".

	await filters.nth(0).locator('.filter-editor__value-input wa-option[value="track"]').click(); // value select should open automatically after selecting the operator

	const jsonCondition = filters.nth(1);
	const jsonPropertyInput = jsonCondition.locator('.filter-editor__property wa-input');
	await jsonPropertyInput.click();
	await jsonCondition.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Traits$/ }).click();
	const jsonPathInput = jsonCondition.locator('.filter-editor__path >> input');
	await jsonPathInput.fill('email');
	const jsonOperatorSelect = jsonCondition.locator('.filter-editor__operator');
	await expect(jsonOperatorSelect).toHaveJSProperty('value', '0');
	const jsonValueInput = jsonCondition.locator('.filter-editor__value-input');
	await jsonValueInput.locator('input').fill('a@example.com');

	await jsonPropertyInput.click();
	await jsonCondition
		.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Properties$/ })
		.click();
	await expect(jsonPropertyInput).toHaveJSProperty('value', 'Properties');
	await expect(jsonPathInput).toHaveValue('email');
	await expect(jsonOperatorSelect).toHaveJSProperty('value', '0');
	await expect(jsonValueInput).toHaveJSProperty('value', 'a@example.com');

	await jsonPropertyInput.click();
	await jsonCondition.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Traits$/ }).click();

	for (const operator of [
		{ name: 'exists', index: 24 },
		{ name: 'does not exist', index: 25 },
	]) {
		const condition = filters.nth(1);
		await condition.locator('.filter-editor__property wa-input').click();
		await condition.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Traits$/ }).click();
		const pathInput = condition.locator('.filter-editor__path >> input');
		await pathInput.fill('email');
		const operatorSelect = condition.locator('.filter-editor__operator');
		const operatorOption = operatorSelect.locator(`wa-option[value="${operator.index}"]`);
		await expect(operatorOption).toHaveText(operator.name);
		await operatorSelect.click();
		await operatorOption.click();
		await expect(operatorSelect).toHaveJSProperty('value', String(operator.index));
		await pathInput.fill('');
		await expect(operatorSelect).toHaveJSProperty('value', null);
		await condition.locator('.filter-editor__remove-condition').click();
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

	let button = name.locator('..').locator('..').locator('wa-button');
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

test(`Keep unavailable consent purposes removable but not selectable`, async ({ page }) => {
	// Ads has no event consent location, which the event pipelines need.
	const allPurposes: ConsentPurpose[] = [
		{
			id: 'marketing',
			name: 'Marketing',
			eventConsentLocations: [{ purposeCode: 'marketing' }],
			profileConsentLocation: null,
		},
		{ id: 'ads', name: 'Ads', eventConsentLocations: [], profileConsentLocation: { property: 'consent_ads' } },
		{
			id: 'analytics',
			name: 'Analytics',
			eventConsentLocations: [{ purposeCode: 'analytics' }],
			profileConsentLocation: null,
		},
	];
	let purposes = allPurposes;
	await page.route('**/v1/consent-purposes', async (route) => {
		await route.fulfill({ json: { purposes } });
	});

	const id = await addJavascriptSource(page);
	await page.goto(`${adminURL}/connections/${id}/pipelines`);
	const name = page.locator('.connection-pipelines__no-pipeline-pipeline-types .list-tile__name', {
		hasText: 'Import events',
	});
	await name.locator('..').locator('..').locator('wa-button').click();
	await expect(page.locator('.pipeline__header')).toBeAttached();

	const consents = page.locator('.pipeline__consents');
	await consents.locator('.pipeline__consents-toggle wa-switch').click();
	const select = consents.locator('.pipeline__consents-select');
	await select.click();
	await select.locator('wa-option[value="marketing"]').click();
	await select.locator('wa-option[value="analytics"]').click();
	await expect(select).toHaveJSProperty('value', ['marketing', 'analytics']);

	// An unavailable purpose cannot be selected, with the mouse or the keyboard.
	const ads = select.locator('wa-option[value="ads"]');
	await expect(ads).toHaveClass(/pipeline__consents-option--unavailable/);
	await ads.click();
	await expect(select).toHaveJSProperty('value', ['marketing', 'analytics']);
	await page.keyboard.press('ArrowDown'); // from Marketing, current since the listbox opened, to Ads
	await page.keyboard.press('Enter');
	await expect(select).toHaveJSProperty('value', ['marketing', 'analytics']);

	// A selected purpose that is deleted keeps a red tag, and can be removed.
	purposes = allPurposes.filter((p) => p.id !== 'analytics');
	const response = page.waitForResponse((response) => response.url().endsWith('/consent-purposes'));
	await page.evaluate(() => window.dispatchEvent(new Event('focus')));
	await response;
	const invalidTag = select.locator('[part~="invalid-tag"]');
	await expect(invalidTag).toHaveText('Analytics');
	await expect(select).toHaveJSProperty('value', ['marketing', 'analytics']);
	await expect(consents.locator('.pipeline__consents-error')).toContainText('"Analytics" no longer exists');
	await invalidTag.locator('[part~="remove-button"]').click();
	await expect(select).toHaveJSProperty('value', ['marketing']);
	await expect(select.locator('[part~="invalid-tag"]')).toHaveCount(0);
	await expect(select.locator('wa-option[value="analytics"]')).toHaveCount(0);
});
