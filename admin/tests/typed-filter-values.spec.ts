import { expect, test } from '@playwright/test';
import BigNumber from 'bignumber.js';
import JSONbig from 'json-bigint';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { build } from 'esbuild';
import { spawnSync } from 'child_process';
import API from '../src/lib/api/api';
import { WORKSPACE_ID_KEY } from '../src/constants/storage';
import Type, { ObjectType, Property } from '../src/lib/api/types/types';
import { EditorFilter, Filter, FilterCondition, FilterOperator, Pipeline } from '../src/lib/api/types/pipeline';
import {
	prepareEditorFilter,
	transformInPipelineToSet,
	transformPipeline,
	TransformedPipelineType,
} from '../src/lib/core/pipeline';
import TransformedConnection from '../src/lib/core/connection';
import {
	decodeFilter,
	encodeFilter,
	filterNumber,
	filterValueKind,
	formatJSONFilterValue,
	isDecimalNumber,
	parseJSONFilterText,
	parseText,
	serializeFilter,
	toEditorFilter,
} from '../src/utils/filters';
import { parseFilter } from '../src/utils/filters_parser';

const property = (name: string, type: Type): Property => ({
	name,
	type,
	prefilled: '',
	createRequired: false,
	updateRequired: false,
	readOptional: true,
	nullable: false,
	displayName: '',
	description: '',
});
const schema: ObjectType = {
	kind: 'object',
	properties: [
		property('properties', { kind: 'json' }),
		property('traits', { kind: 'json' }),
		property('type', { kind: 'string' }),
		property('event', { kind: 'string' }),
		property('account', { kind: 'int', bitSize: 64, unsigned: false }),
		property('unsigned', { kind: 'int', bitSize: 64, unsigned: true }),
		property('amount', { kind: 'decimal', precision: 5, scale: 2 }),
		property('ratio', { kind: 'float', bitSize: 64 }),
		property('year', { kind: 'year' }),
		property('when', { kind: 'datetime' }),
		property('date', { kind: 'date' }),
		property('time', { kind: 'time' }),
		property('uuid', { kind: 'uuid' }),
		property('ip', { kind: 'ip' }),
	],
};
const draft = (property: string, values: string[], operator: FilterOperator = 'is'): EditorFilter => ({
	operator: 'and',
	rules: [{ property, operator, values }],
});
const condition = (filter: Filter): FilterCondition => filter.rules[0] as FilterCondition;

for (const [input, kind, value] of [
	['100', 'number', '100'],
	['100.50', 'number', '100.50'],
	['1e3', 'number', '1e3'],
	['"100"', 'string', '100'],
	["'100'", 'string', '100'],
	['EUR', 'string', 'EUR'],
	['" EUR "', 'string', ' EUR '],
	['00123', 'string', '00123'],
	['true', 'boolean', true],
	['false', 'boolean', false],
	['null', 'null', null],
	['"true"', 'string', 'true'],
	["'false'", 'string', 'false'],
	['"null"', 'string', 'null'],
	['""', 'string', ''],
	["''", 'string', ''],
	['+100', 'string', '+100'],
	['.5', 'string', '.5'],
] as const) {
	test(`JSON control preserves ${input}`, () => {
		const parsed = parseJSONFilterText(input, 'is one of');
		expect(filterValueKind(parsed)).toBe(kind);
		expect(kind === 'number' && typeof parsed === 'object' && parsed !== null ? parsed.text : parsed).toBe(value);
	});
}

test('Boolean and Null recognition is restricted to membership', () => {
	for (const operator of ['is', 'contains', 'starts with', 'ends with'] as const) {
		for (const token of ['true', 'false', 'null']) expect(parseJSONFilterText(token, operator)).toBe(token);
	}
	for (const input of ['', ' ', '\t', '1e76', '1e-38'])
		expect(() => parseJSONFilterText(input, 'is one of')).toThrow();
	expect(() => parseText('""')).toThrow();
	expect(parseText('"foo\\n"')).toBe('foon');
});

test('quoted text and mixed JSON values round trip through both languages', () => {
	const values = [
		filterNumber('9007199254740993', true),
		'100',
		true,
		false,
		null,
		'true',
		'null',
		'',
		' EUR ',
		'a"b\'c\\d\n\r\t',
		'\\n',
		'é😀',
	];
	const model: Filter = { operator: 'and', rules: [{ property: 'properties.value', operator: 'is one of', values }] };
	const editor = toEditorFilter(model, schema);
	expect((editor.rules[0] as { values: string[] }).values.slice(0, 5)).toEqual([
		'9007199254740993',
		'"100"',
		'true',
		'false',
		'null',
	]);
	expect(encodeFilter(prepareEditorFilter(editor, schema))).toBe(encodeFilter(model));
	for (const formatted of [false, true]) {
		const text = serializeFilter(model, formatted);
		expect(encodeFilter(prepareEditorFilter(parseFilter(text, schema), schema))).toBe(encodeFilter(model));
	}
	for (const value of values) expect(parseJSONFilterText(formatJSONFilterValue(value), 'is one of')).toEqual(value);
});

for (const bitSize of [8, 16, 24, 32, 64] as const) {
	for (const unsigned of [false, true]) {
		test(`integer ${bitSize} unsigned=${unsigned} uses exact Number thresholds`, () => {
			const integerSchema: ObjectType = {
				kind: 'object',
				properties: [property('account', { kind: 'int', bitSize, unsigned })],
			};
			for (const raw of unsigned
				? ['0', '18446744073709551615', '9007199254740993']
				: [
						'-9223372036854775808',
						'9223372036854775807',
						'9007199254740991',
						'9007199254740992',
						'9007199254740993',
					]) {
				const model = prepareEditorFilter(draft('account', [raw]), integerSchema);
				const encoded = encodeFilter(model);
				expect(encoded).toContain(`"values":[${raw}]`);
				const decoded = decodeFilter(JSONbig.parse(encoded));
				expect(encodeFilter(prepareEditorFilter(toEditorFilter(decoded, integerSchema), integerSchema))).toBe(
					encoded,
				);
			}
			for (const raw of [
				'100.0',
				'1e2',
				'"100"',
				unsigned ? '-1' : '9223372036854775808',
				unsigned ? '18446744073709551616' : '-9223372036854775809',
			]) {
				expect(() => prepareEditorFilter(draft('account', [raw]), integerSchema)).toThrow();
			}
		});
	}
}

test('decimal and JSON limits do not narrow float or column-independent thresholds', () => {
	for (const raw of [
		'9'.repeat(76),
		'1e75',
		'1e-37',
		'10e-38',
		'0.005',
		'1.2300',
		'0e20000000000000000000',
		'-0.00e-20000000000000000000',
		'1e' + '0'.repeat(80) + '1',
	]) {
		expect(isDecimalNumber(raw), raw).toBe(true);
		for (const name of ['amount', 'properties.value']) {
			const prepared = prepareEditorFilter(draft(name, [raw]), schema);
			const decoded = decodeFilter(JSONbig.parse(encodeFilter(prepared)));
			const operand = condition(decoded).values[0];
			expect(typeof operand).toBe('object');
			if (operand != null && typeof operand === 'object')
				expect(new BigNumber(operand.text).eq(new BigNumber(raw))).toBe(true);
		}
	}
	for (const raw of [
		'1e76',
		'1e-38',
		'9'.repeat(77),
		'9'.repeat(40) + '.' + '9'.repeat(37),
		'+1',
		'001',
		'.5',
		'1.',
		'1e',
	]) {
		expect(isDecimalNumber(raw), raw).toBe(false);
	}
	expect(encodeFilter(prepareEditorFilter(draft('amount', ['1000.00']), schema))).toContain('"values":[1000]');
	expect(encodeFilter(prepareEditorFilter(draft('ratio', ['1e100']), schema))).toContain('"values":[1e+100]');
	for (const raw of ['100.0', '1e2', '0', '10000'])
		expect(() => prepareEditorFilter(draft('year', [raw]), schema)).toThrow();
	const float32: ObjectType = { kind: 'object', properties: [property('ratio', { kind: 'float', bitSize: 32 })] };
	expect(() => prepareEditorFilter(draft('ratio', ['1e100']), float32)).toThrow();
});

test('operator compatibility, cardinality, pruning and retained kinds', () => {
	expect(() => prepareEditorFilter(draft('properties.x', ['100', '"100"'], 'is between'), schema)).toThrow();
	for (const operator of ['starts with', 'ends with'] as const) {
		expect(() => prepareEditorFilter(draft('properties.x', ['100'], operator), schema)).toThrow();
		expect(() => prepareEditorFilter(draft('properties.x', ['""'], operator), schema)).not.toThrow();
	}
	for (const values of [[], ['1'], ['1', '2', '3']])
		expect(() => prepareEditorFilter(draft('properties.x', values, 'is between'), schema)).toThrow();
	for (const values of [[], ['1', '2']])
		expect(() => prepareEditorFilter(draft('account', values), schema)).toThrow();
	expect(() => prepareEditorFilter(draft('properties.x', ['null'], 'is null'), schema)).toThrow();
	expect(() => prepareEditorFilter(draft('type', ['""']), schema)).toThrow();
	const model = prepareEditorFilter(draft('properties.x', ['false', 'null', '0', '""', ''], 'is one of'), schema);
	expect(encodeFilter(model)).toContain('"values":[false,null,0,""]');
	const editor = toEditorFilter(decodeFilter(JSONbig.parse(encodeFilter(model))), schema);
	const leaf = editor.rules[0];
	if (!('rules' in leaf)) {
		leaf.operator = 'is';
		leaf.values = leaf.values.slice(0, 1);
	}
	expect(() => prepareEditorFilter(editor, schema)).toThrow('changed type');
	const changed: EditorFilter = {
		operator: 'and',
		rules: [{ property: 'properties.x', operator: 'is', values: ['100'], valueKinds: ['string'] }],
	};
	expect(() => prepareEditorFilter(changed, schema)).toThrow('changed type');
});

test('all JSON operators enforce the approved operand matrix', () => {
	for (const operator of [
		'is',
		'is not',
		'is less than',
		'is less than or equal to',
		'is greater than',
		'is greater than or equal to',
		'contains',
		'does not contain',
	] as const) {
		for (const text of ['100', '"100"', '""'])
			expect(() => prepareEditorFilter(draft('properties.x', [text], operator), schema)).not.toThrow();
	}
	for (const operator of ['is between', 'is not between'] as const) {
		for (const values of [
			['100', '1e3'],
			['""', '"9"'],
		])
			expect(() => prepareEditorFilter(draft('properties.x', values, operator), schema)).not.toThrow();
		expect(() => prepareEditorFilter(draft('properties.x', ['100', '"9"'], operator), schema)).toThrow();
	}
	for (const operator of ['is one of', 'is not one of'] as const) {
		expect(
			condition(
				prepareEditorFilter(
					draft('properties.x', ['100', '"100"', 'true', 'false', 'null', '"null"'], operator),
					schema,
				),
			).values,
		).toEqual([filterNumber('100', true), '100', true, false, null, 'null']);
	}
	for (const operator of ['starts with', 'ends with'] as const) {
		expect(() => prepareEditorFilter(draft('properties.x', ['100'], operator), schema)).toThrow();
		expect(() => prepareEditorFilter(draft('properties.x', ['""'], operator), schema)).not.toThrow();
	}
	for (const operator of ['is before', 'is on or before', 'is after', 'is on or after'] as const) {
		expect(() => prepareEditorFilter(draft('properties.x', ['100'], operator), schema)).toThrow();
	}
	for (const operator of [
		'is true',
		'is false',
		'is empty',
		'is not empty',
		'is null',
		'is not null',
		'exists',
		'does not exist',
	] as const) {
		const filter: EditorFilter = { operator: 'and', rules: [{ property: 'properties.x', operator }] };
		expect(condition(prepareEditorFilter(filter, schema))).toEqual({ property: 'properties.x', operator });
		expect(() => prepareEditorFilter(draft('properties.x', [], operator), schema)).toThrow();
	}
});

test('ordinary String, enums and array operands retain schema-aware controls', () => {
	const stringArraySchema: ObjectType = {
		kind: 'object',
		properties: [
			property('tags', { kind: 'array', elementType: { kind: 'string', values: ['ignored enum'] } }),
			property('kind', { kind: 'string', values: ['', '"literal"', 'a'.repeat(100)] }),
		],
	};
	for (const value of [' spaced ', '"quoted"', '\\path', "a'b", '😀']) {
		const model: Filter = { operator: 'and', rules: [{ property: 'tags', operator: 'contains', values: [value] }] };
		expect(encodeFilter(prepareEditorFilter(toEditorFilter(model, stringArraySchema), stringArraySchema))).toBe(
			encodeFilter(model),
		);
		expect(
			encodeFilter(
				prepareEditorFilter(parseFilter(serializeFilter(model, false), stringArraySchema), stringArraySchema),
			),
		).toBe(encodeFilter(model));
	}
	for (const value of ['', '"literal"', 'a'.repeat(100)]) {
		expect(condition(prepareEditorFilter(draft('kind', [value]), stringArraySchema)).values).toEqual([value]);
	}
	for (const value of ['a'.repeat(81), '\u0000', '\ud800', '""'])
		expect(() => prepareEditorFilter(draft('type', [value]), schema)).toThrow();
	for (const property of ['account', 'amount', 'ratio', 'year'])
		expect(() => parseFilter(`${property} is "100"`, schema)).toThrow();
});

test('pipeline preparation sends typed values and includes only their base input properties', async () => {
	const outputSchema: ObjectType = {
		kind: 'object',
		properties: [property('id', { kind: 'int', bitSize: 64, unsigned: false })],
	};
	const type: TransformedPipelineType = {
		name: 'Import',
		description: '',
		target: 'User',
		eventType: null,
		inputSchema: schema,
		outputSchema,
		inputMatchingSchema: null,
		outputMatchingSchema: null,
		fields: ['Filter'],
	};
	const filter = prepareEditorFilter(
		draft('properties.value', ['9007199254740993', '"100"', 'false', 'null'], 'is one of'),
		schema,
	);
	const raw = {
		id: 'pipeline',
		name: 'Import',
		target: 'User',
		filter,
		userIDColumn: 'account',
		transformation: { mapping: { id: 'account' }, function: null },
	} as unknown as Pipeline;
	const pipeline = transformPipeline(raw, outputSchema, true, schema);
	const api = new API('https://krenalis.test', 'workspace');
	api.expressionsProperties = async () => ['account'];
	const connection = { isSource: true, isDatabase: true, connector: { type: 'Database' } } as TransformedConnection;
	const result = await transformInPipelineToSet(pipeline, null, type, api, connection, false, [], []);
	expect(encodeFilter(result.filter)).toBe(encodeFilter(filter));
	expect(result.inSchema.properties.map((p) => p.name)).toEqual(['account', 'properties']);
	expect(result.outSchema.properties.map((p) => p.name)).toEqual(['id']);
	expect(raw.filter).toEqual(filter);
});

test('JSONbig response branches preserve semantic values and kind', () => {
	for (const raw of [
		'100',
		'100.0',
		'1e2',
		'999999999999999',
		'9999999999999999',
		'-99999999999999',
		'0.1234567890123',
		'0.1234567890123456789012345678901234567',
		'1e75',
		'1e-37',
		'1e100',
	]) {
		const model = decodeFilter(
			JSONbig.parse(
				`{"operator":"and","rules":[{"property":"ratio","operator":"is","values":[${raw},"${raw}"]}]}`,
			),
		);
		const values = condition(model).values;
		expect(filterValueKind(values[0])).toBe('number');
		expect(values[1]).toBe(raw);
		if (typeof values[0] === 'object' && values[0] !== null)
			expect(new BigNumber(values[0].text).eq(raw)).toBe(true);
	}
	expect(() =>
		decodeFilter({
			operator: 'and',
			rules: [{ property: 'x', operator: 'is', values: [{ kind: 'number', text: '1' }] }],
		}),
	).toThrow();
});

test('temporal, UUID and IP operands keep their String representation', () => {
	for (const [name, value] of [
		['when', '2026-10-24T00:00:00.123456789+02:00'],
		['date', '2026-10-24'],
		['time', '12:03:04.123456789'],
		['uuid', '38d065ab-ca46-4812-a83c-a9712e09c153'],
		['ip', '2001:db8::1'],
	]) {
		const prepared = prepareEditorFilter(draft(name, [value]), schema);
		expect(condition(prepared).values).toEqual([value]);
		expect(encodeFilter(prepareEditorFilter(toEditorFilter(prepared, schema), schema))).toBe(
			encodeFilter(prepared),
		);
	}
});

test('all connector defaults retain meaning after deferred schema resolution', () => {
	for (const name of ['googleanalytics/event_types.go', 'mixpanel/mixpanel.go', 'posthog/posthog.go']) {
		const source = readFileSync(resolve(__dirname, '../../connectors', name), 'utf8');
		const defaults = [...source.matchAll(/DefaultFilter:\s*"((?:[^"\\]|\\.)*)"/g)];
		expect(defaults.length).toBeGreaterThan(0);
		for (const match of defaults) {
			const source = JSON.parse('"' + match[1] + '"');
			expect(() => parseFilter(source)).not.toThrow();
			const filter = prepareEditorFilter(parseFilter(source, schema), schema);
			expect(encodeFilter(prepareEditorFilter(parseFilter(serializeFilter(filter, false), schema), schema))).toBe(
				encodeFilter(filter),
			);
		}
	}
	const adminDefault = prepareEditorFilter(parseFilter('type is "identify" or traits is not empty', schema), schema);
	expect(condition(adminDefault).values).toEqual(['identify']);
});

test('existing textual parser regression corpus', async () => {
	const result = await build({
		entryPoints: [resolve(__dirname, '../src/utils/filters_parser_test.ts')],
		bundle: true,
		write: false,
		platform: 'node',
		format: 'cjs',
		loader: { '.css': 'empty' },
	});
	const run = spawnSync(process.execPath, ['-e', result.outputFiles[0].text], { encoding: 'utf8' });
	expect(run.status, run.stdout + run.stderr).toBe(0);
});

test('HTTP bodies, query strings and nested connection filters share the lossless codec', async () => {
	const api = new API('https://krenalis.test', 'workspace');
	const originalFetch = globalThis.fetch;
	const requests: { url: string; body?: string }[] = [];
	const filter = prepareEditorFilter(
		draft('properties.x', ['9007199254740993', '"9007199254740993"', 'true', 'false', 'null'], 'is one of'),
		schema,
	);
	const encoded = encodeFilter(filter);
	expect(encoded).toBe(
		'{"operator":"and","rules":[{"property":"properties.x","operator":"is one of","values":[9007199254740993,"9007199254740993",true,false,null]}]}',
	);
	globalThis.fetch = async (input, init) => {
		const url = String(input);
		requests.push({ url, body: init?.body as string });
		const connection = `{"pipelines":[{"filter":${encoded}}]}`;
		const body =
			init?.method === 'GET' && url.endsWith('/connections')
				? `{"connections":[${connection}]}`
				: init?.method === 'GET' && url.endsWith('/connections/connection')
					? connection
					: '{"id":"pipeline"}';
		return new Response(body, { status: 200, headers: { 'Content-Type': 'application/json' } });
	};
	try {
		const list = await api.workspaces.connections.find();
		const detail = await api.workspaces.connections.get('connection');
		for (const connection of [list[0], detail]) {
			expect(encodeFilter(connection.pipelines[0].filter)).toBe(encoded);
			expect(serializeFilter(connection.pipelines[0].filter, false)).toContain(
				'9007199254740993, "9007199254740993", true, false, null',
			);
		}
		const pipeline = { name: 'Typed', filter, compression: '' as const, format: '' };
		await api.workspaces.connections.createPipeline('connection', 'User', '', pipeline);
		await api.workspaces.connections.updatePipeline('pipeline', pipeline);
		await api.workspaces.eventListeners.create('connection', 3, filter);
		await api.workspaces.connections.apiUsers('connection', schema, filter);
		await api.workspaces.profiles.find(['account'], filter, '', false, 0, 20);
		await api.workspaces.profiles.events('38d065ab-ca46-4812-a83c-a9712e09c153');
		for (const request of requests.filter((request) => request.body != null)) {
			expect(request.body).toContain(`"filter":${encoded}`);
			expect(request.body).not.toContain('"kind":"number"');
		}
		for (const request of requests.filter((request) => request.url.includes('filter='))) {
			const json = new URL(request.url).searchParams.get('filter');
			if (!request.url.includes('/events?')) expect(json).toBe(encoded);
			else expect(JSON.parse(json).rules[0].values).toEqual(['38d065ab-ca46-4812-a83c-a9712e09c153']);
		}
	} finally {
		globalThis.fetch = originalFetch;
	}
});

test('HTTP bodies and query strings preserve 76 digits, scale 37 and float thresholds', async () => {
	const api = new API('https://krenalis.test', 'workspace');
	const originalFetch = globalThis.fetch;
	const requests: { url: string; body?: string }[] = [];
	globalThis.fetch = async (input, init) => {
		requests.push({ url: String(input), body: init?.body as string });
		return new Response('{"id":"pipeline"}', { status: 200, headers: { 'Content-Type': 'application/json' } });
	};
	try {
		for (const [name, text] of [
			['amount', '9'.repeat(76)],
			['properties.x', '0.' + '1'.repeat(37)],
			['ratio', '1e100'],
		]) {
			const filter = prepareEditorFilter(draft(name, [text]), schema);
			await api.workspaces.connections.updatePipeline('pipeline', {
				name: 'Typed',
				filter,
				compression: '',
				format: '',
			});
			await api.workspaces.profiles.find([name], filter, '', false, 0, 20);
			const body = requests[requests.length - 2].body;
			const query = new URL(requests[requests.length - 1].url).searchParams.get('filter');
			for (const wire of [JSONbig.parse(body).filter, JSONbig.parse(query)]) {
				const value = wire.rules[0].values[0];
				expect(BigNumber.isBigNumber(value) || typeof value === 'number').toBe(true);
				expect(new BigNumber(value).eq(text)).toBe(true);
			}
			expect(body).toMatch(/"values":\[-?\d/);
			expect(query).toMatch(/"values":\[-?\d/);
			expect(body).not.toContain('"kind":"number"');
		}
	} finally {
		globalThis.fetch = originalFetch;
	}
});

test('grid, builder, save and preview preserve typed operands in the browser', async ({ page }) => {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	const result = await build({
		entryPoints: [resolve(__dirname, 'typed-filter-values.fixture.tsx')],
		bundle: true,
		write: false,
		platform: 'browser',
		format: 'iife',
		outdir: '/tmp/typed-filter-values-fixture',
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	await page.setContent('<div id="root"></div>');
	await page.addStyleTag({ content: result.outputFiles.find((file) => file.path.endsWith('.css'))!.text });
	await page.addScriptTag({ content: result.outputFiles.find((file) => file.path.endsWith('.js'))!.text });
	const grid = page.locator('.connection-pipelines__pipeline-filter');
	await expect(grid).toContainText('9007199254740993, "100", true, false, null, ""');
	await expect(grid).toContainText('9223372036854775807');
	await expect(grid).toContainText('18446744073709551615');
	await page.getByRole('button', { name: 'Manage...', exact: true }).click();
	const input = (name: string) => page.locator(`.pipeline__filters-value-input input[name="${name}"]`);
	await expect(input('value-0-0')).toHaveValue('9007199254740993');
	await expect(input('value-0-1')).toHaveValue('"100"');
	await expect(input('value-0-5')).toHaveValue('""');
	await expect(input('value-1-0')).toHaveValue('9223372036854775807');
	await expect(input('value-2-0')).toHaveValue('18446744073709551615');
	await page.getByRole('button', { name: 'Save typed filter', exact: true }).click();
	await expect(page.getByTestId('payload')).toContainText('"values":[9007199254740993,"100",true,false,null,""]');
	await expect(page.getByTestId('payload')).toContainText('"values":[9223372036854775807]');
	await expect(page.getByTestId('payload')).toContainText('"values":[18446744073709551615]');
	await input('value-0-1').fill('" EUR \\"quoted\\" \\\\ "');
	await input('value-1-0').fill('-9223372036854775808');
	await page.getByRole('button', { name: 'Save typed filter', exact: true }).click();
	await expect(page.getByTestId('payload')).toContainText('"values":[-9223372036854775808]');
	const saved = await page.getByTestId('payload').innerText();
	expect(JSONbig.parse(saved).filter.rules[0].values[1]).toBe(' EUR "quoted" \\ ');
	await page.getByRole('button', { name: 'Preview typed filter', exact: true }).click();
	await expect(page.getByTestId('preview')).toContainText('"values":[-9223372036854775808]');
	expect(JSONbig.parse(await page.getByTestId('preview').innerText()).filter.rules[0].values[1]).toBe(
		' EUR "quoted" \\ ',
	);
	await page.getByRole('button', { name: 'Stop preview', exact: true }).click();
	await input('value-1-0').fill('1e2');
	await page.getByRole('button', { name: 'Save typed filter', exact: true }).click();
	await expect(page.getByTestId('error')).not.toHaveText('');
	await expect(page.getByTestId('payload')).toHaveText(saved);
	const preview = await page.getByTestId('preview').innerText();
	await page.getByRole('button', { name: 'Preview typed filter', exact: true }).click();
	await expect(page.getByTestId('preview')).toHaveText(preview);
	await input('value-1-0').fill('-9223372036854775808');
	await page.getByRole('button', { name: 'Reload from API', exact: true }).click();
	await expect(grid).toContainText('-9223372036854775808');
	await page.getByRole('button', { name: 'Manage...', exact: true }).click();
	await expect(input('value-0-1')).toHaveValue('" EUR \\"quoted\\" \\\\ "');
	await expect(input('value-1-0')).toHaveValue('-9223372036854775808');
	await expect(page.getByTestId('prepared')).not.toContainText('"kind":"number"');
	expect(errors).toEqual([]);
});

test('live backend preserves typed JSON filters through the API, builder and reload', async ({ page }) => {
	test.skip(process.env.KRENALIS_TEST_FS_TEMP_DIR == null, 'Requires the backend started by TestAdmin');
	// Load the live harness only when TestAdmin has provided its configuration.
	const { login, logout, addJavascriptSource, adminURL } = require('./utils') as typeof import('./utils');
	await page.setViewportSize({ width: 1600, height: 1000 });
	await login(page);
	try {
		const connectionID = await addJavascriptSource(page);
		const filter = prepareEditorFilter(
			draft(
				'properties.value',
				[
					'9007199254740993',
					'"9007199254740993"',
					'true',
					'false',
					'null',
					'""',
					'9'.repeat(76),
					'0.' + '1'.repeat(37),
				],
				'is one of',
			),
			schema,
		);
		const pipelineID = await page.evaluate(
			async ({ connectionID, filter, workspaceIDKey }) => {
				const workspaceID = localStorage.getItem(workspaceIDKey);
				const api = new (window as any).API(window.location.origin, workspaceID) as API;
				return api.workspaces.connections.createPipeline(connectionID, 'Event', null, {
					name: 'Typed JSON',
					enabled: false,
					filter,
					inSchema: null,
					outSchema: null,
					transformation: null,
					requiredConsents: null,
					compression: '',
					format: '',
				});
			},
			{ connectionID, filter, workspaceIDKey: WORKSPACE_ID_KEY },
		);
		await page.goto(`${adminURL}/connections/${connectionID}/pipelines`);
		const grid = page.locator('.connection-pipelines__pipeline-filter');
		await expect(grid).toContainText('9007199254740993, "9007199254740993", true, false, null, ""');
		await page.getByRole('button', { name: 'Manage...', exact: true }).click();
		const input = (position: number) =>
			page.locator(`.pipeline__filters-value-input input[name="value-0-${position}"]`);
		await expect(input(0)).toHaveValue('9007199254740993');
		await expect(input(1)).toHaveValue('"9007199254740993"');
		await expect(input(4)).toHaveValue('null');
		await expect(input(5)).toHaveValue('""');
		await input(0).fill('-9223372036854775808');
		const [response] = await Promise.all([
			page.waitForResponse(
				(response) =>
					response.url().endsWith(`/pipelines/${pipelineID}`) && response.request().method() === 'PUT',
			),
			page.locator('.pipeline__header-save >> button').click(),
		]);
		expect(response.status()).toBe(200);
		expect(response.request().postData()).toContain(
			'"values":[-9223372036854775808,"9007199254740993",true,false,null,"",',
		);
		await expect(grid).toContainText('-9223372036854775808');
		await page.reload();
		await expect(grid).toContainText('-9223372036854775808, "9007199254740993", true, false, null, ""');
		await page.getByRole('button', { name: 'Manage...', exact: true }).click();
		await expect(input(0)).toHaveValue('-9223372036854775808');
		await expect(input(1)).toHaveValue('"9007199254740993"');
		await expect(input(2)).toHaveValue('true');
		await expect(input(3)).toHaveValue('false');
		await expect(input(4)).toHaveValue('null');
		await expect(input(5)).toHaveValue('""');
		for (const [position, expected] of [
			[6, '9'.repeat(76)],
			[7, '0.' + '1'.repeat(37)],
		] as const) {
			expect(new BigNumber(await input(position).inputValue()).eq(expected)).toBe(true);
		}
	} finally {
		await logout(page);
	}
});
