import { expect, Locator, Page, Request, test } from '@playwright/test';
import { CountryFormat, ObjectType, Property } from '../src/lib/api/types/types';
import { ResponseProfile } from '../src/lib/api/types/responses';
import { ProfilePropertyRoles } from '../src/lib/api/types/workspace';
import { PROFILES_PROPERTIES_KEY } from '../src/constants/storage';
import { adminURL, login, logout } from './utils';

const profileProperty = (
	name: string,
	displayName?: string,
	overrides: Partial<Omit<Property, 'name' | 'displayName'>> = {},
): Property => ({
	name,
	displayName,
	prefilled: '',
	type: { kind: 'string' },
	createRequired: false,
	updateRequired: false,
	readOptional: true,
	nullable: false,
	description: '',
	...overrides,
});

const profileObjectProperty = (name: string, properties: Property[], displayName?: string): Property => ({
	...profileProperty(name, displayName),
	type: { kind: 'object', properties },
});

const profileSchema: ObjectType = {
	kind: 'object',
	properties: [
		profileProperty('customer_id', 'Customer ID'),
		profileProperty('email', '   ', { type: { kind: 'string', semantics: 'email' } }),
		profileProperty('first_name', 'First name'),
		profileProperty('last_name', 'Last name'),
		profileProperty('country', 'Country', {
			type: { kind: 'string', semantics: 'country', format: 'alpha-2' },
		}),
		profileProperty('photo_url', 'Photo', { type: { kind: 'string', semantics: 'url' } }),
		profileObjectProperty('billing_address', [profileProperty('city', 'City')], 'Billing address'),
		profileObjectProperty('shipping_address', [profileProperty('city', 'City')]),
	],
};

const getProfileSchema = (countryFormat: CountryFormat): ObjectType => ({
	...profileSchema,
	properties: profileSchema.properties?.map((property) =>
		property.name === 'country'
			? { ...property, type: { kind: 'string', semantics: 'country', format: countryFormat } }
			: property,
	),
});

const profilePhoto = 'data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=';

const propertyRoles: ProfilePropertyRoles = {
	firstName: 'first_name',
	lastName: 'last_name',
	email: 'email',
	country: 'country',
	photo: 'photo_url',
};

const profiles: ResponseProfile[] = [
	{
		kpid: 'profile-1',
		updatedAt: '2026-08-01T10:00:00Z',
		attributes: {
			customer_id: 'customer-1',
			email: 'one@example.com',
			first_name: 'Ada',
			last_name: 'Lovelace',
			country: 'GB',
			photo_url: profilePhoto,
			billing_address: { city: 'Milan' },
			shipping_address: { city: 'Rome' },
		},
	},
	{
		kpid: 'profile-2',
		updatedAt: '2026-08-02T10:00:00Z',
		attributes: {
			customer_id: 'customer-2',
			email: 'two@example.com',
			first_name: 'Grace',
			last_name: 'Hopper',
			country: 'US',
			billing_address: { city: 'Turin' },
			shipping_address: { city: 'Naples' },
		},
	},
	{
		kpid: 'profile-3',
		updatedAt: '2026-08-03T10:00:00Z',
		attributes: {
			customer_id: 'customer-3',
			email: 'three@example.com',
			first_name: '   ',
			last_name: 'Curie',
			country: '   ',
			photo_url: '   ',
			billing_address: { city: 'Bologna' },
			shipping_address: { city: 'Florence' },
		},
	},
	{
		kpid: 'profile-4',
		updatedAt: '2026-08-04T10:00:00Z',
		attributes: {
			customer_id: 'customer-4',
			email: 'four@example.com',
			first_name: '',
			last_name: '',
			country: '',
			photo_url: '',
			billing_address: { city: 'Palermo' },
			shipping_address: { city: 'Genoa' },
		},
	},
];

interface MockProfilesOptions {
	propertyRoles?: ProfilePropertyRoles;
	profilesForRequest?: (request: Request) => ResponseProfile[];
	responseProfiles?: ResponseProfile[];
	schema?: ObjectType;
}

const mockProfiles = async (
	page: Page,
	{
		propertyRoles: roleAssignments = propertyRoles,
		profilesForRequest,
		responseProfiles = profiles,
		schema = profileSchema,
	}: MockProfilesOptions = {},
) => {
	await page.route('**/v1/workspaces', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		for (const workspace of body.workspaces) {
			workspace.profileSchema.propertyRoles = roleAssignments;
		}
		await route.fulfill({ response, json: body });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({ json: { schema, propertyRoles: roleAssignments, primarySources: {} } });
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		const first = Number(request.postDataJSON().first);
		const limit = Number(request.postDataJSON().limit);
		const matchingProfiles = profilesForRequest?.(request) ?? responseProfiles;
		const resultProfiles = matchingProfiles.slice(first, first + limit);
		await route.fulfill({
			json: {
				profiles: resultProfiles,
				total: matchingProfiles.length,
				hasNext: first + resultProfiles.length < matchingProfiles.length,
			},
		});
	});
	await page.route('**/v1/profiles/*/attributes/query', async (route) => {
		const fragments = new URL(route.request().url()).pathname.split('/');
		const kpid = decodeURIComponent(fragments[fragments.length - 3]);
		const profile = responseProfiles.find((candidate) => candidate.kpid === kpid);
		await route.fulfill({ json: { attributes: profile?.attributes ?? {} } });
	});
	await page.route('**/v1/identity-resolution/latest', async (route) => {
		await route.fulfill({ json: { startTime: null, endTime: '2026-08-03T12:00:00Z' } });
	});
};

const openFirstNameFilter = async (page: Page) => {
	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const condition = editor.locator('.filter-editor__filter');
	const property = condition.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('input').fill('First');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^First name$/ }).click();

	return {
		filters,
		editor,
		condition,
		property,
		valueInput: condition.locator('.filter-editor__value-input input'),
	};
};

const fillStringFilterCondition = async (condition: Locator, propertyName: RegExp, value: string) => {
	const property = condition.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: propertyName }).click();
	const operator = condition.locator('.filter-editor__operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = condition.locator('.filter-editor__value-input input');
	await valueInput.fill(value);
	await valueInput.press('Enter');
};

test.beforeEach(async ({ page }) => {
	await login(page);
});

test.afterEach(async ({ page }) => {
	await logout(page);
});

test(`Render the profile grid and manage visible columns`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	await expect(page.locator('.profiles-list__page-header h1')).toHaveText('Profiles');
	await expect(page.locator('.profiles-list__page-header .profiles-list__grid-summary')).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list__grid-summary-content wa-icon')).toHaveAttribute('name', 'people');
	await expect(page.getByRole('button', { name: 'Refine filters' })).toHaveCount(0);
	await expect(page.locator('.profiles-list__identity-resolution-button')).toHaveText('Run identity resolution');
	await expect(page.locator('.profiles-list__card')).toBeVisible();
	await expect(page.locator('.profiles-list__footer')).toBeVisible();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–4 of 4');

	const headers = page.locator('.profiles-list .grid__header-cell');
	await expect(headers).toHaveCount(9);
	await expect(headers.nth(0)).toHaveText('Profile');
	await expect(headers.nth(1)).toHaveText('Customer ID');
	await expect(headers.nth(2)).toHaveText('email');
	await expect(headers.nth(3)).toHaveText('First name');
	await expect(headers.nth(4)).toHaveText('Last name');
	await expect(headers.nth(5)).toHaveText('Country');
	await expect(headers.nth(6)).toHaveText('Photo');
	await expect(headers.nth(7)).toHaveText('Billing address › City');
	await expect(headers.nth(8)).toHaveText('shipping_address › City');
	const hints = page.locator('.profiles-list .grid-keyboard-hints__hint');
	await expect(hints).toHaveCount(1);
	await expect(hints).toHaveText('Navigate profiles');
	await expect(hints).toHaveAccessibleName('Use Up and Down arrow keys to navigate profiles.');
	await expect(hints.locator('.grid-keyboard-hints__keys')).toHaveAttribute('aria-hidden', 'true');
	await expect(hints.locator('kbd')).toHaveCount(2);
	await expect(hints.locator('kbd').first()).not.toHaveAttribute('tabindex');
	await expect(hints.getByRole('button')).toHaveCount(0);
	const keycapCursors = await hints
		.locator('kbd')
		.evaluateAll((keycaps) => keycaps.map((keycap) => getComputedStyle(keycap).cursor));
	expect(keycapCursors).not.toContain('pointer');

	const columns = page.locator('.profiles-list__toggle-columns');
	const columnsButton = columns.getByRole('button', { name: 'Columns', exact: true });
	await expect(columnsButton).toHaveAccessibleName('Columns');
	await expect(columns.locator('wa-button[slot="trigger"]')).toHaveText('Columns');
	await columnsButton.hover();
	await expect(page.locator('wa-tooltip[open]')).toHaveCount(0);
	await columnsButton.click();
	await expect(columns).toHaveJSProperty('open', true);
	await expect(columns.locator('.profiles-list__column-chooser-header')).toBeVisible();
	await expect(columns.locator('.profiles-list__column-chooser').getByText('Columns', { exact: true })).toHaveCount(
		0,
	);
	await expect(columns.locator('.profiles-list__column-chooser').getByText(/^\d+ shown$/)).toHaveCount(0);
	await expect(columns.getByRole('button', { name: 'Reset', exact: true })).toHaveCount(0);
	await expect(columns.locator('.profiles-list__column-chooser-list')).toHaveCSS('overflow-x', 'hidden');
	await expect(columns.locator('.profiles-list__column-chooser-list')).toHaveCSS('overflow-y', 'auto');
	await expect(columns.locator('wa-checkbox')).toHaveText([
		'Customer ID',
		'email',
		'First name',
		'Last name',
		'Country',
		'Photo',
		'Billing address › City',
		'shipping_address › City',
	]);
	const columnSearch = columns.getByRole('textbox', { name: 'Search columns' });
	await expect(columns.getByRole('button', { name: 'Clear entry' })).toHaveCount(0);
	await columnSearch.fill('FIRST NAME');
	await expect(columns.locator('wa-checkbox')).toHaveText(['First name']);
	const clearColumnSearch = columns.getByRole('button', { name: 'Clear entry' });
	await expect(clearColumnSearch).toBeVisible();
	await clearColumnSearch.click();
	await expect(columnSearch).toHaveValue('');
	await expect(columns.locator('wa-checkbox')).toHaveCount(8);
	await expect(clearColumnSearch).toHaveCount(0);
	await columnSearch.fill('billing_address.city');
	await expect(columns.locator('wa-checkbox')).toHaveText(['Billing address › City']);
	await columnSearch.fill('does not exist');
	await expect(columns.locator('wa-checkbox')).toHaveCount(0);
	await expect(columns.locator('.profiles-list__column-chooser-empty')).toHaveText('No columns found');
	await columnSearch.fill('');

	const emailColumn = columns.locator('wa-checkbox', { hasText: 'email' });
	await expect(emailColumn.getByRole('checkbox')).toHaveAccessibleName('email');
	await emailColumn.click();
	await expect(columns).toHaveJSProperty('open', true);
	await expect(emailColumn).toHaveJSProperty('checked', false);
	await expect(headers).toHaveCount(8);
	await expect(headers.nth(0)).toHaveText('Profile');
	await expect(headers.nth(1)).toHaveText('Customer ID');
	await expect(headers.nth(2)).toHaveText('First name');
	await expect(headers.nth(3)).toHaveText('Last name');
	await expect(headers.nth(4)).toHaveText('Country');
	await expect(headers.nth(5)).toHaveText('Photo');
	await expect(headers.nth(6)).toHaveText('Billing address › City');
	await expect(headers.nth(7)).toHaveText('shipping_address › City');
});

test(`Keep the column chooser within a narrow viewport with long labels`, async ({ page }) => {
	const longLabel = 'Customer preferred shipping street address and apartment number';
	await mockProfiles(page, {
		schema: {
			...profileSchema,
			properties: [...(profileSchema.properties ?? []), profileProperty('shipping_street', longLabel)],
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	await page.setViewportSize({ width: 360, height: 720 });
	const menu = columns.locator('[part="menu"]');
	await expect(menu).toBeVisible();
	await expect(columns.getByRole('checkbox', { name: longLabel, exact: true })).toBeVisible();
	await expect.poll(() => menu.evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(360);
	await expect.poll(() => menu.evaluate((element) => element.getBoundingClientRect().left)).toBeGreaterThanOrEqual(0);
});

test(`Toggle a column locally when its profile property is already loaded`, async ({ page }) => {
	let profileRequests = 0;
	let loadedProjection: string[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (!url.pathname.endsWith('/v1/profiles/query')) {
			return;
		}
		profileRequests++;
		loadedProjection = request.postDataJSON().properties ?? [];
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	const shippingAddressColumn = columns.locator('wa-checkbox', { hasText: 'shipping_address › City' });
	const requestsBeforeToggle = profileRequests;
	expect(loadedProjection).toContain('shipping_address');

	await shippingAddressColumn.click();
	await expect(page.locator('.profiles-list .grid__header-cell', { hasText: 'shipping_address › City' })).toHaveCount(
		0,
	);
	await shippingAddressColumn.click();
	await expect(page.locator('.profiles-list .grid__header-cell', { hasText: 'shipping_address › City' })).toHaveCount(
		1,
	);
	await expect(page.locator('.profiles-list .grid__loading')).toHaveCount(0);
	await page.evaluate(() => new Promise(requestAnimationFrame));
	expect(profileRequests).toBe(requestsBeforeToggle);
});

test(`Keep the profile grid stable while loading a newly selected property`, async ({ page }) => {
	await page.evaluate(({ key, value }) => localStorage.setItem(key, value), {
		key: PROFILES_PROPERTIES_KEY,
		value: JSON.stringify([
			{
				label: 'shipping_address › City',
				name: 'shipping_address.city',
				isUsed: false,
				type: 'string',
			},
		]),
	});

	let markProjectionStarted = () => {};
	let releaseProjection = () => {};
	const projectionStarted = new Promise<void>((resolve) => {
		markProjectionStarted = resolve;
	});
	const projectionCanFinish = new Promise<void>((resolve) => {
		releaseProjection = resolve;
	});
	let projectionRequest: Request | undefined;
	let initialProjection: string[] = [];
	await mockProfiles(page);
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		const properties = request.postDataJSON().properties ?? [];
		if (!properties.includes('shipping_address')) {
			initialProjection = properties;
			await route.fallback();
			return;
		}
		projectionRequest = request;
		markProjectionStarted();
		await projectionCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	expect(initialProjection).not.toContain('shipping_address');
	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	const shippingAddressColumn = columns.locator('wa-checkbox', { hasText: 'shipping_address › City' });
	await shippingAddressColumn.click();
	await projectionStarted;

	await expect(shippingAddressColumn).toHaveJSProperty('checked', true);
	await expect(grid.locator('.grid__header-cell', { hasText: 'shipping_address › City' })).toHaveCount(0);
	await expect(grid.locator('.grid__loading')).toHaveCount(0);
	await expect(grid.locator('.grid__row--clickable')).toHaveCount(4);
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–4 of 4');

	releaseProjection();
	await expect(grid.locator('.grid__header-cell', { hasText: 'shipping_address › City' })).toHaveCount(1);
	await expect(grid.locator('.grid__row--clickable').first()).toContainText('Rome');
	expect(projectionRequest?.postDataJSON().first).toBe(0);
	expect(projectionRequest?.postDataJSON().limit).toBe(100);
	expect(projectionRequest!.postDataJSON().schema).toEqual(profileSchema);
});

test(`Reorder profile columns without reloading their data and preserve the order`, async ({ page }) => {
	let gridRequests = 0;
	page.on('request', (request) => {
		if (new URL(request.url()).pathname.endsWith('/v1/profiles/query')) {
			gridRequests++;
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const headers = page.locator('.profiles-list .grid__header-cell');
	const firstRowCells = page.locator('.profiles-list .grid__row').first().locator('.grid__cell');
	await expect(headers).toHaveCount(9);
	await expect(page.getByRole('button', { name: 'Move Profile column' })).toHaveCount(0);
	const requestsBeforeReorder = gridRequests;

	const emailHandle = page.getByRole('button', { name: 'Move email column' });
	const emailHeader = headers.nth(2);
	const firstNameHeader = headers.nth(3);
	await expect(emailHeader).toHaveAccessibleName('email');
	await emailHeader.hover();
	await expect(emailHandle).toHaveCSS('opacity', '1');
	const emailHandleBox = await emailHandle.boundingBox();
	const firstNameHeaderBox = await firstNameHeader.boundingBox();
	if (emailHandleBox == null || firstNameHeaderBox == null) {
		throw new Error('Unable to locate profile column reorder controls');
	}
	await page.mouse.move(emailHandleBox.x + emailHandleBox.width / 2, emailHandleBox.y + emailHandleBox.height / 2);
	await page.mouse.down();
	await page.mouse.move(
		emailHandleBox.x + emailHandleBox.width / 2 + 5,
		emailHandleBox.y + emailHandleBox.height / 2,
		{
			steps: 2,
		},
	);
	await page.mouse.move(
		firstNameHeaderBox.x + firstNameHeaderBox.width / 2,
		firstNameHeaderBox.y + firstNameHeaderBox.height / 2,
		{ steps: 8 },
	);
	const dragOverlay = page.locator('.grid__column-reorder-overlay');
	await expect(emailHeader).toHaveCSS('opacity', '0');
	await expect(dragOverlay).toContainText('email');
	await page.mouse.up();
	await expect(dragOverlay).toHaveCount(0);
	await page.mouse.move(0, 0);
	await expect(emailHandle).toHaveCSS('opacity', '0');

	await expect(headers.nth(0)).toHaveText('Profile');
	await expect(headers.nth(1)).toHaveText('Customer ID');
	await expect(headers.nth(2)).toHaveText('First name');
	await expect(headers.nth(3)).toHaveText('email');
	await expect(firstRowCells.nth(1)).toHaveText('customer-1');
	await expect(firstRowCells.nth(2)).toHaveText('Ada');
	await expect(firstRowCells.nth(3)).toHaveText('one@example.com');
	expect(gridRequests).toBe(requestsBeforeReorder);

	await page.reload();
	await expect(headers.nth(0)).toHaveText('Profile');
	await expect(headers.nth(1)).toHaveText('Customer ID');
	await expect(headers.nth(2)).toHaveText('First name');
	await expect(headers.nth(3)).toHaveText('email');
	await expect(headers.nth(4)).toHaveText('Last name');
	await expect(firstRowCells.nth(2)).toHaveText('Ada');
	await expect(firstRowCells.nth(3)).toHaveText('one@example.com');
	await expect(firstRowCells.nth(4)).toHaveText('Lovelace');

	const requestsBeforeKeyboardReorder = gridRequests;
	const lastNameHandle = page.getByRole('button', { name: 'Move Last name column' });
	await lastNameHandle.focus();
	await page.keyboard.press('Space');
	await expect(page.locator('[role="status"][aria-live="assertive"]')).toContainText(
		'Draggable item last_name was moved over droppable area last_name.',
	);
	// The keyboard sensor attaches its listener in a deferred task after activation.
	await page.evaluate(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));
	await page.keyboard.press('ArrowLeft');
	await expect(page.locator('[role="status"][aria-live="assertive"]')).toContainText(
		'Draggable item last_name was moved over droppable area email.',
	);
	await page.keyboard.press('Space');

	await expect(headers.nth(2)).toHaveText('First name');
	await expect(headers.nth(3)).toHaveText('Last name');
	await expect(headers.nth(4)).toHaveText('email');
	await expect(firstRowCells.nth(2)).toHaveText('Ada');
	await expect(firstRowCells.nth(3)).toHaveText('Lovelace');
	await expect(firstRowCells.nth(4)).toHaveText('one@example.com');
	await expect(lastNameHandle).toBeFocused();
	expect(gridRequests).toBe(requestsBeforeKeyboardReorder);
});

test(`Keep a reopened operator menu open and respect focus moved to another control`, async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);
	const { condition, property, valueInput } = await openFirstNameFilter(page);
	const operatorSelect = condition.locator('.filter-editor__operator');
	await expect(operatorSelect.locator('wa-option[value="0"]')).toBeVisible();
	await page.clock.install();
	await operatorSelect.evaluate(async (select: any) => {
		await select.hide();
		await select.show();
	});
	// Exercise the delayed focus handoff that used to close a newly reopened menu.
	await page.clock.runFor(100);
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await operatorSelect.locator('wa-option[value="0"]').click();
	await expect(valueInput).toBeFocused();

	await operatorSelect.getByRole('combobox').click();
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await property.locator('input').click();
	await expect(operatorSelect).toHaveJSProperty('open', false);
	await page.clock.runFor(100);
	await expect(property.locator('input')).toBeFocused();
});

test(`Apply a completed filter explicitly`, async ({ page }) => {
	const countRequests: Request[] = [];
	const gridRequests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(request);
		} else if (url.pathname.endsWith('/v1/profiles/query')) {
			gridRequests.push(request);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(filters).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	await expect(filters.locator('.profiles-list__filter-description')).not.toBeVisible();
	await expect(filters.getByRole('link', { name: 'Learn more about filters' })).not.toBeVisible();
	await expect(rows).toHaveCount(4);
	const editFilters = filters.getByRole('button', { name: 'Add filter', exact: true });
	const filterTitle = filters.locator('.profiles-list__filter-title');
	const editFiltersLayout = await editFilters.boundingBox();
	const collapsedFilterTitleLayout = await filterTitle.boundingBox();
	await editFilters.click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	await expect(editor).toBeVisible();
	const closeFilters = filters.getByRole('button', { name: 'Collapse', exact: true });
	await expect(closeFilters).not.toBeFocused();
	await expect(filterTitle).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-summary-value')).not.toBeVisible();
	const gridCountLayout = await page.locator('.profiles-list__grid-summary').boundingBox();
	const editorLayout = await editor.boundingBox();
	const overviewLayout = await filters.locator('.profiles-list__filter-overview').boundingBox();
	const closeFiltersLayout = await closeFilters.boundingBox();
	const expandedFilterTitleLayout = await filterTitle.boundingBox();
	expect(gridCountLayout).not.toBeNull();
	expect(editFiltersLayout).not.toBeNull();
	expect(collapsedFilterTitleLayout).not.toBeNull();
	expect(editorLayout).not.toBeNull();
	expect(overviewLayout).not.toBeNull();
	expect(closeFiltersLayout).not.toBeNull();
	expect(expandedFilterTitleLayout).not.toBeNull();
	expect(Math.abs(closeFiltersLayout!.x - editFiltersLayout!.x)).toBeLessThanOrEqual(1);
	expect(Math.abs(closeFiltersLayout!.y - editFiltersLayout!.y)).toBeLessThanOrEqual(1);
	expect(closeFiltersLayout!.height).toBe(editFiltersLayout!.height);
	expect(Math.abs(expandedFilterTitleLayout!.x - collapsedFilterTitleLayout!.x)).toBeLessThanOrEqual(1);
	expect(Math.abs(expandedFilterTitleLayout!.y - collapsedFilterTitleLayout!.y)).toBeLessThanOrEqual(1);
	expect(Math.abs(overviewLayout!.y - editorLayout!.y)).toBeLessThanOrEqual(1);
	await expect(filterTitle).toHaveText('Filters');
	await expect(filters.locator('.profiles-list__filter-description')).toHaveText(
		'Choose which profiles to include in the results. Leave empty to include all profiles.',
	);
	await expect(filters.getByRole('link', { name: 'Learn more about filters' })).toHaveAttribute(
		'href',
		'https://www.krenalis.com/docs/ref/admin/filters',
	);
	await expect(editor.locator('.filter-editor__filter')).toHaveCount(1);
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
	await expect.poll(() => gridRequests.length).toBe(1);
	await expect.poll(() => countRequests.length).toBe(0);
	await expect(page.locator('.profiles-list__layout')).not.toHaveAttribute('inert');
	expect(await editor.evaluate((element) => getComputedStyle(element).overflowY)).toBe('visible');
	const expandedLayout = await page.locator('.profiles-list__layout').boundingBox();
	expect(expandedLayout?.height).toBeGreaterThanOrEqual((await page.evaluate(() => window.innerHeight)) / 2 - 1);
	await expect(editor.getByRole('button', { name: 'Apply filters' })).toBeDisabled();
	await expect(editor.getByRole('button', { name: 'Cancel' })).toHaveCount(0);
	await expect(filters.locator('.filter-editor__group-subject')).toHaveText('Profiles matching');

	const condition = filters.locator('.filter-editor__filter');
	const emptyConditionRemoveButton = filters.getByRole('button', {
		name: 'Remove condition 1',
		exact: true,
	});
	await expect(emptyConditionRemoveButton).toHaveCount(0);
	const property = condition.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await expect(
		property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^email$/ }),
	).toBeVisible();
	await expect(
		property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Billing address › City$/ }),
	).toBeVisible();
	const propertyMenu = property.locator('[data-is-combobox-list]');
	await propertyMenu.hover();
	await page.mouse.wheel(0, 200);
	await expect.poll(() => propertyMenu.evaluate((menu) => menu.scrollTop)).toBeGreaterThan(0);
	const propertyMenuLayout = await property.locator('[data-is-combobox-list]').evaluate((menu) => {
		const lastItem = menu.querySelector('.combobox-list__item:last-of-type');
		if (!(lastItem instanceof HTMLElement)) {
			throw new Error('The filter property menu is incomplete');
		}
		const lastItemRect = lastItem.getBoundingClientRect();
		const hitTarget = document.elementFromPoint(
			lastItemRect.left + Math.min(10, lastItemRect.width / 2),
			lastItemRect.top + lastItemRect.height / 2,
		);
		const hitRoot = hitTarget?.getRootNode();
		return {
			lastItemBottom: lastItemRect.bottom,
			lastItemIsVisible:
				lastItem === hitTarget ||
				lastItem.contains(hitTarget) ||
				(hitRoot instanceof ShadowRoot && hitRoot.host === lastItem),
			position: getComputedStyle(menu).position,
			viewportBottom: window.innerHeight,
		};
	});
	expect(propertyMenuLayout.position).toBe('fixed');
	expect(propertyMenuLayout.lastItemBottom).toBeLessThanOrEqual(propertyMenuLayout.viewportBottom);
	expect(propertyMenuLayout.lastItemIsVisible).toBe(true);
	await property.locator('input').fill('First');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^First name$/ }).click();
	await expect(property.locator('input')).toHaveValue('First name');
	await expect(emptyConditionRemoveButton).toBeVisible();
	const operatorSelect = condition.locator('.filter-editor__operator');
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await expect(operatorSelect.locator('wa-option[value="4"]')).toBeVisible();
	const operatorMenuLayout = await operatorSelect.evaluate((select) => {
		const visibleOption = select.querySelector('wa-option[value="4"]');
		if (!(visibleOption instanceof HTMLElement)) {
			throw new Error('The filter operator menu is incomplete');
		}
		const optionRect = visibleOption.getBoundingClientRect();
		const hitTarget = document.elementFromPoint(
			optionRect.left + Math.min(10, optionRect.width / 2),
			optionRect.top + optionRect.height / 2,
		);
		const hitRoot = hitTarget?.getRootNode();
		return {
			optionBottom: optionRect.bottom,
			optionTop: optionRect.top,
			optionIsVisible:
				visibleOption === hitTarget ||
				visibleOption.contains(hitTarget) ||
				(hitRoot instanceof ShadowRoot && hitRoot.host === visibleOption),
			viewportBottom: window.innerHeight,
		};
	});
	expect(operatorMenuLayout.optionTop).toBeGreaterThanOrEqual(0);
	expect(operatorMenuLayout.optionBottom).toBeLessThanOrEqual(operatorMenuLayout.viewportBottom);
	expect(operatorMenuLayout.optionIsVisible).toBe(true);
	await operatorSelect.locator('wa-option[value="0"]').click();
	const valueInput = condition.locator('.filter-editor__value-input input');
	await valueInput.fill('Grace');
	await expect.poll(() => countRequests.length).toBe(0);
	await valueInput.press('Enter');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(editor).toBeVisible();
	await expect(rows).toHaveCount(4);
	await rows.first().click();
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toBeVisible();

	const updateResults = page.locator('wa-button.profiles-list__filter-apply');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Apply filters');
	await expect(updateResults).toHaveAttribute('size', 'm');
	await expect(updateResults.locator('wa-icon')).toHaveCount(0);
	await expect(updateResults).toContainText('Apply filters');
	await expect(editor.getByRole('button', { name: 'Apply filters' })).toBeEnabled();
	const documentationLink = filters.getByRole('link', { name: 'Learn more about filters' });
	const documentationLayoutBeforeUpdate = await documentationLink.boundingBox();
	const updateResultsLayout = await updateResults.boundingBox();
	expect(updateResultsLayout!.x + updateResultsLayout!.width).toBeLessThan(editorLayout!.x + editorLayout!.width);
	expect(updateResultsLayout!.y).toBeGreaterThan(editorLayout!.y);
	// The profile drawer covers the filter actions on the right.
	await updateResults.getByRole('button').press('Enter');
	await expect(updateResults.getByRole('button')).toBeDisabled();
	await expect(page.getByText('Results updated', { exact: true })).toHaveCount(0);
	const documentationLayoutAfterUpdate = await documentationLink.boundingBox();
	expect(documentationLayoutAfterUpdate).not.toBeNull();
	expect(Math.abs(documentationLayoutAfterUpdate!.y - documentationLayoutBeforeUpdate!.y)).toBeLessThanOrEqual(1);
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(filters).not.toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(rows).toHaveCount(1);
	await expect(rows.first()).toContainText('Grace Hopper');
	await expect(rows.first()).not.toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid')).toBeFocused();
	expect(gridRequests.at(-1)?.postDataJSON().first).toBe(0);
	expect(gridRequests.at(-1)?.postDataJSON().filter).toEqual({
		operator: 'and',
		rules: [{ property: 'first_name', operator: 'is', values: ['Grace'] }],
	});
	expect(countRequests).toHaveLength(0);

	const removeCondition = filters.getByRole('button', { name: 'Remove condition 1', exact: true });
	await expect(removeCondition).toHaveCount(1);
	await expect(removeCondition).toHaveJSProperty('tagName', 'BUTTON');
	await valueInput.focus();
	await page.keyboard.press('Tab');
	await expect(removeCondition).toBeFocused();
	const removeConditionFocusStyle = await removeCondition.evaluate((button) => {
		const rootStyle = getComputedStyle(document.documentElement);
		const focusRingWidth = rootStyle.getPropertyValue('--wa-focus-ring-width').trim();
		return {
			outlineStyle: getComputedStyle(button).outlineStyle,
			outlineWidth: parseFloat(getComputedStyle(button).outlineWidth),
			focusRingWidth:
				parseFloat(focusRingWidth) * (focusRingWidth.endsWith('rem') ? parseFloat(rootStyle.fontSize) : 1),
		};
	});
	expect(removeConditionFocusStyle.outlineStyle).toBe('solid');
	expect(removeConditionFocusStyle.outlineWidth).toBe(removeConditionFocusStyle.focusRingWidth);
	await removeCondition.click();
	await expect(filters.locator('.filter-editor__announcement')).toHaveText('Condition removed');
	await expect(editor.getByRole('button', { name: 'Add a condition', exact: true })).toBeFocused();
	await expect(removeCondition).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(rows).toHaveCount(1);
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await expect(rows).toHaveCount(4);
	expect(gridRequests.at(-1)?.postDataJSON().filter).toBeUndefined();

	const requestsBeforeCollapse = gridRequests.length;
	await closeFilters.click();
	await expect(editor).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	await expect(filters.getByRole('button', { name: 'Add filter', exact: true })).toBeFocused();
	expect(gridRequests).toHaveLength(requestsBeforeCollapse);
});

test(`Enable Apply filters when there are unapplied changes`, async ({ page }) => {
	const profileRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count') || url.pathname.endsWith('/v1/profiles/query')) {
			profileRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	const updateResults = page.getByRole('button', {
		name: 'Apply filters',
		exact: true,
	});
	await expect(updateResults).toBeEnabled();

	await valueInput.fill('');
	await expect(updateResults).toBeEnabled();

	await valueInput.fill('Grace');
	await expect(updateResults).toBeEnabled();
	expect(profileRequests).toHaveLength(1);
	await updateResults.click();
	await expect(editor.locator('.profiles-list__filter-apply wa-spinner')).toHaveCount(0);
	await expect(updateResults).toBeDisabled();

	const rootLogical = editor.locator(
		'.filter-editor__group--root > .filter-editor__group-header > .filter-editor__logical',
	);
	await rootLogical.click();
	await rootLogical.locator('wa-option[value="or"]').click();
	await expect(rootLogical).toHaveJSProperty('value', 'or');
	await expect(updateResults).toBeDisabled();
	expect(profileRequests).toHaveLength(2);
});

test(`Keep logical operator changes local until applying the filter`, async ({ page }) => {
	const gridRequests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/query')) {
			gridRequests.push(request);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	const applyFilters = page.getByRole('button', { name: 'Apply filters', exact: true });
	await expect(applyFilters).toBeVisible();
	expect(gridRequests).toHaveLength(1);

	const rootGroup = editor.locator('.filter-editor__group--root');
	const rootLogical = rootGroup.locator(':scope > .filter-editor__group-header > .filter-editor__logical');
	await rootLogical.click();
	await rootLogical.locator('wa-option[value="or"]').click();
	await expect(rootLogical).toHaveJSProperty('value', 'or');
	await expect(applyFilters).toBeVisible();
	expect(gridRequests).toHaveLength(1);

	await rootGroup.locator(':scope > .filter-editor__group-actions > .filter-editor__add-group').click();
	const nestedGroup = editor.locator('.filter-editor__group').nth(1);
	const nestedCondition = nestedGroup.locator('.filter-editor__filter');
	const nestedProperty = nestedCondition.locator('.filter-editor__property');
	await nestedProperty.locator('wa-input').click();
	await nestedProperty
		.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^First name$/ })
		.click();
	await nestedCondition.locator('.filter-editor__operator wa-option[value="0"]').click();
	const nestedValue = nestedCondition.locator('.filter-editor__value-input input');
	await nestedValue.fill('Ada');
	await nestedValue.press('Enter');
	expect(gridRequests).toHaveLength(1);
	await expect(applyFilters).toBeVisible();

	const nestedLogical = nestedGroup.locator(':scope > .filter-editor__group-header > .filter-editor__logical');
	await expect(nestedLogical).toHaveJSProperty('value', 'and');
	await nestedLogical.click();
	await nestedLogical.locator('wa-option[value="or"]').click();
	await expect(nestedLogical).toHaveJSProperty('value', 'or');
	await expect(applyFilters).toBeVisible();
	expect(gridRequests).toHaveLength(1);
	await applyFilters.click();
	await expect.poll(() => gridRequests.length).toBe(2);
	expect(gridRequests.at(-1)!.postDataJSON().filter).toEqual({
		operator: 'or',
		rules: [
			{ property: 'first_name', operator: 'is', values: ['Grace'] },
			{ operator: 'or', rules: [{ property: 'first_name', operator: 'is', values: ['Ada'] }] },
		],
	});
});

test(`Summarize collapsed filters within their available width`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const longValue = 'A customer name deliberately long enough to require truncation in the summary';
	const { filters, editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill(longValue);
	await valueInput.press('Enter');

	const rootGroup = editor.locator('.filter-editor__group--root');
	const addRootGroup = rootGroup.locator(':scope > .filter-editor__group-actions > .filter-editor__add-group');
	await addRootGroup.click();
	const simpleGroup = editor.locator('.filter-editor__group').nth(1);
	await simpleGroup.locator(':scope > .filter-editor__group-actions > .filter-editor__add-condition').click();
	const simpleConditions = simpleGroup.locator('.filter-editor__filter');
	await fillStringFilterCondition(simpleConditions.nth(0), /^Last name$/, 'Hopper');
	await fillStringFilterCondition(simpleConditions.nth(1), /^Country$/, 'US');

	await addRootGroup.click();
	const complexGroup = editor.locator('.filter-editor__group').nth(2);
	const addComplexCondition = complexGroup.locator(
		':scope > .filter-editor__group-actions > .filter-editor__add-condition',
	);
	await addComplexCondition.click();
	await addComplexCondition.click();
	const complexConditions = complexGroup.locator('.filter-editor__filter');
	await fillStringFilterCondition(complexConditions.nth(0), /^email$/, 'one@example.com');
	await fillStringFilterCondition(complexConditions.nth(1), /^Customer ID$/, 'customer-1');
	await fillStringFilterCondition(complexConditions.nth(2), /^Billing address › City$/, 'Milan');

	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(editor).toHaveCount(0);

	const chips = filters.locator('.profiles-list__filter-summary-chips');
	await chips.evaluate((element) => {
		element.style.flex = 'none';
		element.style.width = '1200px';
	});
	const visibleUnits = chips.locator(
		'.profiles-list__filter-summary-visible-chips > .profiles-list__filter-summary-unit',
	);
	await expect(visibleUnits).toHaveCount(3);
	await expect(visibleUnits.nth(1).locator('.profiles-list__filter-summary-chip-label')).toHaveText(
		'Last name is "Hopper" or Country is "US"',
	);
	await expect(visibleUnits.nth(2).locator('.profiles-list__filter-summary-chip-label')).toHaveText(
		'any group · 3 conditions',
	);

	const firstLabel = visibleUnits.first().locator('.profiles-list__filter-summary-chip-label');
	const firstTooltip = visibleUnits.first().locator('wa-tooltip');
	await expect(visibleUnits.first().locator('.profiles-list__filter-summary-chip')).toHaveAccessibleName(
		`First name is "${longValue}"`,
	);
	await expect.poll(() => firstLabel.evaluate((element) => element.scrollWidth > element.clientWidth)).toBe(true);
	await expect(firstTooltip).toHaveJSProperty('disabled', false);
	await firstLabel.hover();
	await expect(firstTooltip).toHaveJSProperty('open', true);
	await expect(firstTooltip).toHaveText(`First name is "${longValue}"`);
	await expect(visibleUnits.first().locator('.profiles-list__filter-summary-chip-remove')).toHaveAccessibleName(
		`Remove condition: First name is "${longValue}"`,
	);
	await expect(visibleUnits.nth(2).locator('wa-tooltip')).toHaveJSProperty('disabled', true);

	const setVisibleUnitCapacity = (capacity: number) =>
		chips.evaluate((element, visibleCapacity) => {
			const measurement = element.querySelector('.profiles-list__filter-summary-measurement');
			const overflow = element.querySelector('.profiles-list__filter-summary-overflow-measurement');
			const overflowLabel = overflow?.querySelector('.profiles-list__filter-summary-overflow');
			if (!(measurement instanceof HTMLElement) || !(overflow instanceof HTMLElement) || overflowLabel == null) {
				throw new Error('The collapsed filter measurements are incomplete');
			}

			const items = Array.from(
				measurement.querySelectorAll<HTMLElement>(':scope > .profiles-list__filter-summary-unit'),
			);
			if (items.length <= visibleCapacity) {
				throw new Error('The collapsed filter does not contain enough measurable items');
			}

			overflowLabel.textContent = `+${items.length - visibleCapacity} more`;
			const visibleItemsWidth = items
				.slice(0, visibleCapacity)
				.reduce((width, item) => width + item.offsetWidth, 0);
			const gap = Number.parseFloat(getComputedStyle(measurement).columnGap);
			const requiredGapCount = visibleCapacity;
			element.style.width = `${Math.ceil(visibleItemsWidth + gap * requiredGapCount + overflow.offsetWidth + 1)}px`;
		}, capacity);

	await setVisibleUnitCapacity(2);
	await expect(visibleUnits).toHaveCount(2);
	let overflowUnit = chips.locator('.profiles-list__filter-summary-overflow-unit');
	await expect(overflowUnit.locator('.profiles-list__filter-summary-connector')).toHaveText('and');
	await expect(overflowUnit.locator('.profiles-list__filter-summary-overflow')).toHaveText('+1 more');

	await setVisibleUnitCapacity(1);
	await expect(visibleUnits).toHaveCount(1);
	overflowUnit = chips.locator('.profiles-list__filter-summary-overflow-unit');
	await expect(overflowUnit.locator('.profiles-list__filter-summary-connector')).toHaveText('and');
	await expect(overflowUnit.locator('.profiles-list__filter-summary-overflow')).toHaveText('+2 more');

	await chips.evaluate((element) => (element.style.width = '100px'));
	await expect(visibleUnits).toHaveCount(0);
	await expect(chips.locator('.profiles-list__filter-summary-overflow-unit')).toHaveCount(0);
	await expect(chips.locator('.profiles-list__filter-summary-fallback')).toHaveText('6 conditions');
});

test(`Apply a filter with zero matches`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Missing');
	await valueInput.press('Enter');
	const updateResults = page.locator('wa-button.profiles-list__filter-apply');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Apply filters');
	await expect(updateResults.locator('wa-icon')).toHaveCount(0);
	await expect(updateResults).toContainText('Apply filters');
	await updateResults.click();

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('0 profiles');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid__no-rows')).toContainText('No profiles to show');
});

test(`Keep the displayed grid intact when applying filters fails`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (request) =>
			request.postDataJSON().filter != null
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (request.postDataJSON().filter == null) {
			await route.fallback();
			return;
		}
		await route.fulfill({
			status: 503,
			json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const rows = page.locator('.profiles-list .grid__row--clickable');
	const { editor, filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await rows.first().click();
	await expect(page.locator('.profile-drawer')).toBeVisible();
	await page.getByRole('button', { name: 'Apply filters', exact: true }).press('Enter');

	await expect(editor.locator('.profiles-list__filter-grid-error')).toContainText('Unable to apply filters');
	await expect(page.getByRole('button', { name: 'Retry', exact: true })).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-summary-value')).not.toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(rows).toHaveCount(4);
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toBeVisible();
	await expect(filters).not.toHaveClass(/profiles-list__filters--updating-results/);
	await expect(filters).not.toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(page.getByRole('button', { name: 'Columns', exact: true })).toBeEnabled();
	await expect(page.getByRole('combobox', { name: 'Profiles per page', exact: true })).toBeEnabled();
});

test(`Apply the current Undo entry when collapsing and preserve its history`, async ({ page }) => {
	const gridRequests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/query')) {
			gridRequests.push(request);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(valueInput).toHaveValue('Grace');

	const gridRequestsBeforeCollapse = gridRequests.length;
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(filters.getByRole('region', { name: 'Profile filters' })).toHaveCount(0);
	await expect.poll(() => gridRequests.length).toBe(gridRequestsBeforeCollapse + 1);
	await expect(filters.getByRole('button', { name: 'Edit filters', exact: true })).toBeFocused();
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(filters.getByRole('region', { name: 'Profile filters' })).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-title')).toHaveText('Filters');
	await expect(filters.locator('wa-button.profiles-list__filter-toggle')).toHaveText('Edit filters');
	await expect(
		filters.locator('.profiles-list__filter-summary-visible-chips .profiles-list__filter-summary-chip-label'),
	).toHaveText('First name is "Grace"');
	const visibleChipLabel = filters.locator(
		'.profiles-list__filter-summary-visible-chips .profiles-list__filter-summary-chip-label',
	);
	await expect(visibleChipLabel).toHaveCSS('user-select', 'none');
	await expect(visibleChipLabel).toHaveCSS('cursor', 'default');
	await expect(filters.locator('.profiles-list__filter-summary-chip-remove')).toHaveCSS('cursor', 'pointer');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Grace Hopper']);
	await expect(filters.getByRole('button', { name: 'Edit filters', exact: true })).toBeFocused();
	const materializedFilter = gridRequests.at(-1)?.postDataJSON().filter ?? null;
	expect(materializedFilter.rules[0].values).toEqual(['Grace']);

	await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
	await expect(filters.getByRole('region', { name: 'Profile filters' }).locator(':focus')).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Redo', exact: true })).toBeEnabled();
});

test(`Reopen the property menu after collapsing a nested property filter`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const condition = filters.locator('.filter-editor__filter');
	const property = condition.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property
		.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^Billing address › City$/ })
		.click();
	await condition.locator('.filter-editor__operator wa-option[value="0"]').click();
	await condition.locator('.filter-editor__value-input input').fill('Milan');
	await condition.locator('.filter-editor__value-input input').press('Enter');
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();

	await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
	const reopenedProperty = filters.locator('.filter-editor__property');
	await expect(reopenedProperty.locator('input')).toHaveValue('Billing address › City');
	await reopenedProperty.locator('wa-input').click();
	await expect(
		reopenedProperty.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^First name$/ }),
	).toBeVisible();
});

test(`Remove an applied filter from its collapsed chip, update the grid, and preserve history`, async ({ page }) => {
	const countRequests: URL[] = [];
	const gridRequests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		} else if (url.pathname.endsWith('/v1/profiles/query')) {
			gridRequests.push(request);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(
		filters.locator('.profiles-list__filter-summary-visible-chips .profiles-list__filter-summary-chip-label'),
	).toHaveText('First name is "Grace"');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');

	const countRequestsBeforeRemoval = countRequests.length;
	const gridRequestsBeforeRemoval = gridRequests.length;
	await filters.getByRole('button', { name: 'Remove condition: First name is "Grace"', exact: true }).click();

	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
	expect(countRequests).toHaveLength(countRequestsBeforeRemoval);
	expect(gridRequests).toHaveLength(gridRequestsBeforeRemoval + 1);
	expect(gridRequests.at(-1)?.postDataJSON().filter ?? null).toBeNull();

	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const undo = filters.getByRole('button', { name: 'Undo', exact: true });
	const redo = filters.getByRole('button', { name: 'Redo', exact: true });
	await expect(undo).toBeEnabled();
	await undo.click();
	await expect(editor.locator('.filter-editor__property input')).toHaveValue('First name');
	await expect(editor.locator('.filter-editor__value-input input')).toHaveValue('Grace');
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeEnabled();
	expect(countRequests).toHaveLength(countRequestsBeforeRemoval);
	await expect(redo).toBeEnabled();
	await redo.click();
	await expect(editor.locator('.filter-editor__property input')).toHaveValue('');
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeDisabled();
	await expect(undo).toBeEnabled();
	expect(countRequests).toHaveLength(countRequestsBeforeRemoval);
});

test(`Keep failed grid updates accessible after collapsing and discard the error after Undo`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (request) =>
			request.postDataJSON().filter != null
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (request.postDataJSON().filter == null) {
			await route.fallback();
			return;
		}
		await route.fulfill({
			status: 503,
			json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();

	await expect(filters.getByRole('region', { name: 'Profile filters' })).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-grid-error')).toContainText('Unable to apply filters');
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	await expect(filters.getByRole('region', { name: 'Profile filters' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Retry', exact: true })).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);

	await filters.getByRole('button', { name: 'Edit filters', exact: true }).click();
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(page.locator('.profiles-list__filter-grid-error')).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Retry' })).toHaveCount(0);
});

test(`Grow nested filters in the document flow without an internal scrollbar`, async ({ page }) => {
	await page.setViewportSize({ width: 1366, height: 768 });
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	await editor.getByRole('button', { name: 'Add a group', exact: true }).click();
	const nestedGroup = editor.locator('.filter-editor__group:not(.filter-editor__group--root)');
	await expect(nestedGroup).toBeVisible();
	await nestedGroup.getByRole('button', { name: 'Add a condition', exact: true }).click();

	const layout = page.locator('.profiles-list__layout');
	const measurements = await page.evaluate(() => {
		const filters = document.querySelector('.profiles-list__filters');
		const editor = document.querySelector('.profiles-list__filter-editor');
		const layout = document.querySelector('.profiles-list__layout');
		if (!(filters instanceof HTMLElement) || !(editor instanceof HTMLElement) || !(layout instanceof HTMLElement)) {
			throw new Error('The expanded Profiles layout is incomplete');
		}
		return {
			editorOverflowY: getComputedStyle(editor).overflowY,
			filtersBottom: filters.getBoundingClientRect().bottom,
			layoutHeight: layout.getBoundingClientRect().height,
			layoutTop: layout.getBoundingClientRect().top,
			viewportHeight: window.innerHeight,
		};
	});

	expect(measurements.editorOverflowY).toBe('visible');
	expect(measurements.layoutTop).toBeGreaterThanOrEqual(measurements.filtersBottom);
	expect(measurements.layoutHeight).toBeGreaterThanOrEqual(measurements.viewportHeight / 2 - 1);
	await expect(layout).toBeVisible();
});

test(`Collapse immediately and keep the previous results until filters are applied`, async ({ page }) => {
	let markCandidateStarted = () => {};
	let releaseCandidate = () => {};
	const candidateStarted = new Promise<void>((resolve) => {
		markCandidateStarted = resolve;
	});
	const candidateCanFinish = new Promise<void>((resolve) => {
		releaseCandidate = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (request) =>
			request.postDataJSON().filter != null
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (request.postDataJSON().filter == null) {
			await route.fallback();
			return;
		}
		markCandidateStarted();
		await candidateCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const rows = page.locator('.profiles-list .grid__row--clickable');
	const opened = await openFirstNameFilter(page);
	await opened.valueInput.fill('Grace');
	await opened.valueInput.press('Enter');

	await expect(rows).toHaveCount(4);
	await rows.first().click();
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await candidateStarted;
	await expect(opened.editor).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Edit filters', exact: true })).toBeFocused();
	await expect(page.locator('.profiles-list__filter-grid-action .profiles-list__filter-status')).toHaveText(
		'Updating profiles…',
	);
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters applied');
	await expect(rows).toHaveCount(4);
	releaseCandidate();
	await expect(rows).toContainText(['Grace Hopper']);
	await expect(page.locator('.profiles-list__filter-grid-action .profiles-list__filter-status')).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Edit filters', exact: true })).toBeVisible();
});

test(`Keep incomplete filters open when attempting to collapse the editor`, async ({ page }) => {
	const profileRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count') || url.pathname.endsWith('/v1/profiles/query')) {
			profileRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const trigger = filters.getByRole('button', { name: 'Add filter', exact: true });
	await trigger.click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const property = editor.locator('.filter-editor__property');
	await property.locator('wa-input').click();
	await property.locator('input').fill('First');
	await property.locator('.combobox-list__item .schema-combobox-item__name', { hasText: /^First name$/ }).click();
	await expect(page.locator('.profiles-list__layout')).not.toHaveAttribute('inert');
	await expect(editor.locator('.filter-editor__operator')).toHaveJSProperty('open', true);
	await page.keyboard.press('Escape');
	await expect(editor).toBeVisible();
	await page.keyboard.press('Escape');

	await expect(editor).toBeVisible();
	await expect(editor.getByRole('button', { name: 'Apply filters', exact: true })).toBeEnabled();
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();
	await expect(editor).toBeVisible();
	await expect(editor.locator('.filter-editor__property input')).toHaveValue('First name');
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeEnabled();
	expect(profileRequests).toHaveLength(1);
});

test(`Undo and redo confirmed filters without changing the grid`, async ({ page }) => {
	const profileRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count') || url.pathname.endsWith('/v1/profiles/query')) {
			profileRequests.push(url);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, editor, condition, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeEnabled();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');

	await valueInput.fill('');
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(valueInput).toHaveValue('Grace');
	expect(profileRequests).toHaveLength(1);

	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(editor.locator('.filter-editor__filter')).toHaveCount(1);
	await expect(editor.locator('.filter-editor__property input')).toHaveValue('');
	await expect(editor.getByRole('button', { name: 'Add filter', exact: true })).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Redo', exact: true })).toBeEnabled();
	await filters.getByRole('button', { name: 'Redo', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeEnabled();
	await expect(condition.locator('.filter-editor__value-input input')).toHaveValue('Grace');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
	expect(profileRequests).toHaveLength(1);
});

test(`Keep an explicit grid request independent from newer filter edits`, async ({ page }) => {
	let markGraceGridStarted = () => {};
	let releaseGraceGrid = () => {};
	const graceGridStarted = new Promise<void>((resolve) => {
		markGraceGridStarted = resolve;
	});
	const graceGridCanFinish = new Promise<void>((resolve) => {
		releaseGraceGrid = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const filter = route.request().postDataJSON().filter ?? null;
		if (filter?.rules?.[0]?.values?.[0] !== 'Grace') {
			await route.fallback();
			return;
		}
		markGraceGridStarted();
		await graceGridCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	const updateResults = page.locator('wa-button.profiles-list__filter-apply');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Apply filters');
	await updateResults.click();
	await graceGridStarted;
	await expect(updateResults.locator('wa-spinner')).toBeVisible();
	await expect(filters).toHaveClass(/profiles-list__filters--updating-results/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list__grid-summary-sweep')).toBeVisible();
	const refresh = page.getByRole('button', { name: 'Refresh', exact: true });
	const columns = page.locator('.profiles-list__toggle-columns');
	const pageSize = page.getByRole('combobox', { name: 'Profiles per page', exact: true });
	await expect(refresh).toBeDisabled();
	await expect(columns.getByRole('button', { name: 'Columns', exact: true })).toBeDisabled();
	await expect(columns.locator('wa-checkbox').first()).toHaveJSProperty('disabled', true);
	await expect(pageSize).toBeDisabled();

	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeDisabled();
	await expect(valueInput).toBeFocused();
	releaseGraceGrid();

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Grace Hopper']);
	await expect(filters).not.toHaveClass(/profiles-list__filters--updating-results/);
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(valueInput).toHaveValue('Ada');
	await expect(page.getByRole('button', { name: 'Apply filters', exact: true })).toBeEnabled();
	await expect(valueInput).toBeFocused();
	await expect(page.locator('.profiles-list .grid')).not.toBeFocused();
	await expect(refresh).toBeEnabled();
	await expect(columns.getByRole('button', { name: 'Columns', exact: true })).toBeEnabled();
	await expect(columns.locator('wa-checkbox').first()).toHaveJSProperty('disabled', false);
	await expect(pageSize).toBeEnabled();
});

test(`Keep unapplied filters open with Escape while the current filter is being materialized`, async ({ page }) => {
	let markGridStarted = () => {};
	let releaseGrid = () => {};
	let filteredGridRequests = 0;
	const gridStarted = new Promise<void>((resolve) => {
		markGridStarted = resolve;
	});
	const gridCanFinish = new Promise<void>((resolve) => {
		releaseGrid = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const filter = route.request().postDataJSON().filter ?? null;
		if (filter?.rules?.[0]?.values?.[0] !== 'Grace') {
			await route.fallback();
			return;
		}
		filteredGridRequests++;
		markGridStarted();
		await gridCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await gridStarted;

	await valueInput.focus();
	await valueInput.press('Escape');
	await expect(editor).toBeVisible();
	await expect(page.locator('.profiles-list__filter-grid-action .profiles-list__filter-status')).toHaveText(
		'Updating profiles…',
	);
	releaseGrid();

	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Grace Hopper']);
	await valueInput.press('Escape');
	await expect(editor).toHaveCount(0);
	expect(filteredGridRequests).toBe(1);
});

test(`Let collapsing a newer filter supersede an older grid request`, async ({ page }) => {
	let markGraceGridStarted = () => {};
	let releaseGraceGrid = () => {};
	const graceGridStarted = new Promise<void>((resolve) => {
		markGraceGridStarted = resolve;
	});
	const graceGridCanFinish = new Promise<void>((resolve) => {
		releaseGraceGrid = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (request) => {
			const filter = request.postDataJSON().filter ?? null;
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const filter = route.request().postDataJSON().filter ?? null;
		if (filter?.rules?.[0]?.values?.[0] !== 'Grace') {
			await route.fallback();
			return;
		}
		markGraceGridStarted();
		await graceGridCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await graceGridStarted;
	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await filters.getByRole('button', { name: 'Collapse', exact: true }).click();

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Ada Lovelace']);
	releaseGraceGrid();
	await page.waitForTimeout(100);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Ada Lovelace']);
});

test(`Paginate profiles on the server`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 55 }, (_, index) => {
		const profile = profiles[index % profiles.length];
		return {
			...profile,
			kpid: `paginated-profile-${index + 1}`,
			attributes: {
				...profile.attributes,
				customer_id: `paginated-customer-${index + 1}`,
			},
		};
	});
	const requests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/query')) {
			requests.push(request);
		}
	});
	await mockProfiles(page, { responseProfiles });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const rows = page.locator('.profiles-list .grid__row--clickable');
	const pagination = page.locator('.profiles-list__pagination');
	const previousPage = pagination.locator('.profiles-list__pagination-previous').getByRole('button');
	const nextPage = pagination.locator('.profiles-list__pagination-next').getByRole('button');
	const pageSize = pagination.locator('wa-select');
	const pageSizeLabel = pageSize.locator('[part~="form-control-label"]');
	const pageSizeCombobox = pageSize.getByRole('combobox');
	const pageNavigation = pagination.locator('.profiles-list__pagination-navigation');
	await expect(rows).toHaveCount(50);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('55 profiles');
	await expect(pagination).toHaveAttribute('aria-label', 'Profiles pagination');
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 55');
	await expect(pageSizeLabel).toHaveText('Profiles per page');
	await expect(pageSizeCombobox).toHaveAccessibleName('Profiles per page');
	await expect(pageSize.locator('wa-option')).toHaveText(['25', '50', '100']);
	await expect(pageNavigation.locator('wa-button')).toHaveCount(2);
	await expect(previousPage).toHaveAccessibleName('Previous page');
	await expect(nextPage).toHaveAccessibleName('Next page');
	await expect(previousPage).toBeDisabled();
	await expect(nextPage).toBeEnabled();
	await expect(pageSize).toHaveJSProperty('value', '50');
	const desktopLayout = await pagination.evaluate((element) => {
		const results = element.querySelector('.profiles-list__pagination-results');
		const controls = element.querySelector('.profiles-list__pagination-controls');
		if (!(results instanceof HTMLElement) || !(controls instanceof HTMLElement)) {
			throw new Error('The profile pagination groups are missing');
		}
		const paginationRect = element.getBoundingClientRect();
		const resultsRect = results.getBoundingClientRect();
		const controlsRect = controls.getBoundingClientRect();
		return {
			centerDifference: Math.abs(
				resultsRect.top + resultsRect.height / 2 - (controlsRect.top + controlsRect.height / 2),
			),
			controlsRightInset: paginationRect.right - controlsRect.right,
			groupGap: controlsRect.left - resultsRect.right,
			resultsLeftInset: resultsRect.left - paginationRect.left,
		};
	});
	expect(desktopLayout.centerDifference).toBeLessThan(1);
	expect(desktopLayout.controlsRightInset).toBeGreaterThan(0);
	expect(desktopLayout.groupGap).toBeGreaterThan(0);
	expect(desktopLayout.resultsLeftInset).toBeGreaterThan(0);
	expect(requests.at(-1)?.postDataJSON().first).toBe(0);
	expect(requests.at(-1)?.postDataJSON().limit).toBe(100);
	expect(requests.at(-1)?.postDataJSON().includeSchema).toBeUndefined();
	expect(requests.at(-1)!.postDataJSON().schema).toEqual(profileSchema);
	expect(requests.at(-1)?.postDataJSON().properties?.sort()).toEqual([
		'billing_address',
		'country',
		'customer_id',
		'email',
		'first_name',
		'last_name',
		'photo_url',
		'shipping_address',
	]);
	const requestCountBeforeCachedPage = requests.length;

	await pagination.locator('.profiles-list__pagination-next').click();
	await expect(rows).toHaveCount(5);
	await expect(page.locator('.profiles-list .grid__row--active')).toHaveCount(0);
	await expect(page.locator('.profile-drawer')).toHaveCount(0);
	await expect(nextPage).toBeDisabled();
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('51–55 of 55');
	await expect(previousPage).toBeEnabled();
	await expect(nextPage).toBeDisabled();
	expect(requests).toHaveLength(requestCountBeforeCachedPage);

	await pageSizeCombobox.click();
	const selectedPageSize = pageSize.locator('wa-option[aria-selected="true"]');
	await expect(selectedPageSize).toHaveText('50');
	await expect(selectedPageSize.locator('[part="checked-icon"]')).toBeVisible();
	await pageSize.locator('wa-option[value="100"]').click();
	await expect(rows).toHaveCount(55);
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–55 of 55');
	await expect(pageSize).toHaveJSProperty('value', '100');
	const pageSizeMetrics = await pageSize.evaluate((element) => {
		const label = element.shadowRoot?.querySelector('[part~="form-control-label"]');
		const input = element.shadowRoot?.querySelector('[part="display-input"]');
		const combobox = element.shadowRoot?.querySelector('[part~="combobox"]');
		const labelSlot = label?.querySelector('slot');
		if (
			!(label instanceof HTMLElement) ||
			!(input instanceof HTMLInputElement) ||
			combobox == null ||
			labelSlot == null
		) {
			throw new Error('The page size label or input is missing');
		}
		const range = document.createRange();
		range.selectNodeContents(labelSlot);
		const textRect = range.getBoundingClientRect();
		const controlRect = combobox.getBoundingClientRect();
		return {
			labelCenterDifference: Math.abs(
				textRect.top + textRect.height / 2 - controlRect.top - controlRect.height / 2,
			),
			inputFontSize: getComputedStyle(input).fontSize,
			inputWidth: input.clientWidth,
			labelFontSize: getComputedStyle(label).fontSize,
			textWidth: input.scrollWidth,
		};
	});
	expect(pageSizeMetrics.labelCenterDifference).toBeLessThan(1);
	expect(pageSizeMetrics.inputFontSize).toBe(pageSizeMetrics.labelFontSize);
	expect(pageSizeMetrics.textWidth).toBeLessThanOrEqual(pageSizeMetrics.inputWidth);
	expect(requests.at(-1)?.postDataJSON().first).toBe(0);
	expect(requests.at(-1)?.postDataJSON().limit).toBe(200);

	await pageSizeCombobox.click();
	await pageSize.locator('wa-option[value="25"]').click();
	await expect(rows).toHaveCount(25);
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–25 of 55');
	await expect(pageSize).toHaveJSProperty('value', '25');
	expect(requests.at(-1)?.postDataJSON().first).toBe(0);
	expect(requests.at(-1)?.postDataJSON().limit).toBe(50);

	const responsiveLayout = await pagination.evaluate((element) => {
		const results = element.querySelector('.profiles-list__pagination-results');
		const controls = element.querySelector('.profiles-list__pagination-controls');
		const pageSize = element.querySelector('.profiles-list__pagination-page-size');
		const pageSizeLabel = pageSize?.shadowRoot?.querySelector('[part~="form-control-label"]');
		if (
			!(results instanceof HTMLElement) ||
			!(controls instanceof HTMLElement) ||
			!(pageSizeLabel instanceof HTMLElement)
		) {
			throw new Error('The profile pagination layout is incomplete');
		}
		const style = getComputedStyle(element);
		const horizontalPadding = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
		element.style.width = `${Math.ceil(Math.max(results.scrollWidth, controls.scrollWidth) + horizontalPadding + 1)}px`;
		const resultsRect = results.getBoundingClientRect();
		const controlsRect = controls.getBoundingClientRect();
		return {
			controlsTop: controlsRect.top,
			resultsBottom: resultsRect.bottom,
			controlsFlexWrap: getComputedStyle(controls).flexWrap,
			labelWhiteSpace: getComputedStyle(pageSizeLabel).whiteSpace,
			resultsFlexWrap: getComputedStyle(results).flexWrap,
		};
	});
	expect(responsiveLayout.controlsTop).toBeGreaterThan(responsiveLayout.resultsBottom);
	expect(responsiveLayout.controlsFlexWrap).toBe('nowrap');
	expect(responsiveLayout.labelWhiteSpace).toBe('nowrap');
	expect(responsiveLayout.resultsFlexWrap).toBe('nowrap');
});

test(`Navigate continuously across cached profile page boundaries`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 120 }, (_, index) => ({
		...profiles[index % profiles.length],
		kpid: `sequence-profile-${index + 1}`,
	}));
	const requests: Request[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/query')) {
			requests.push(request);
		}
	});
	await mockProfiles(page, { responseProfiles });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	let rows = grid.locator('.grid__row--clickable');
	await expect(rows).toHaveCount(50);
	await rows.nth(49).click();
	await expect(grid).toBeFocused();
	const requestCountBeforeBoundary = requests.length;
	await page.keyboard.press('ArrowDown');

	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
	rows = grid.locator('.grid__row--clickable');
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(rows.first()).toHaveAttribute('data-id', 'sequence-profile-51');
	await expect(grid).toBeFocused();
	await expect.poll(() => requests.length).toBeGreaterThan(requestCountBeforeBoundary);
	expect(requests.at(-1)?.postDataJSON().first).toBe(100);
	expect(requests.at(-1)?.postDataJSON().limit).toBe(50);
	expect(requests.at(-1)!.postDataJSON().schema).toEqual(profileSchema);

	const requestCountBeforeReverse = requests.length;
	await page.keyboard.press('ArrowUp');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 120');
	rows = grid.locator('.grid__row--clickable');
	await expect(rows.last()).toHaveClass(/grid__row--active/);
	await expect(rows.last()).toHaveAttribute('data-id', 'sequence-profile-50');
	await expect(grid).toBeFocused();
	expect(requests).toHaveLength(requestCountBeforeReverse);
});

test(`Deduplicate a pending continuation and lock boundary navigation`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 160 }, (_, index) => ({
		...profiles[index % profiles.length],
		kpid: `locked-navigation-profile-${index + 1}`,
	}));
	let continuationRequests = 0;
	let releaseContinuation!: () => void;
	const continuation = new Promise<void>((resolve) => {
		releaseContinuation = resolve;
	});

	await mockProfiles(page, { responseProfiles });
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (Number(request.postDataJSON().first) !== 100) {
			await route.fallback();
			return;
		}
		continuationRequests++;
		await continuation;
		await route.fulfill({
			json: {
				profiles: responseProfiles.slice(100, 150),
				total: responseProfiles.length,
				hasNext: true,
			},
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	let rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 160');
	await expect.poll(() => continuationRequests).toBe(1);

	rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await page.keyboard.press('ArrowDown');
	await page.keyboard.press('ArrowDown');
	await expect(rows.last()).toHaveClass(/grid__row--active/);
	await expect(grid).toBeFocused();
	expect(continuationRequests).toBe(1);

	releaseContinuation();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('101–150 of 160');
	rows = grid.locator('.grid__row--clickable');
	await expect(rows.first()).toHaveAttribute('data-id', 'locked-navigation-profile-101');
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(grid).toBeFocused();
});

test(`Retry a failed speculative continuation only when its boundary is requested`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 160 }, (_, index) => ({
		...profiles[index % profiles.length],
		kpid: `retry-navigation-profile-${index + 1}`,
	}));
	let continuationRequests = 0;

	await mockProfiles(page, { responseProfiles });
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (Number(request.postDataJSON().first) !== 100) {
			await route.fallback();
			return;
		}
		continuationRequests++;
		if (continuationRequests < 3) {
			await route.fulfill({ status: 503, json: { message: 'temporary failure' } });
			return;
		}
		await route.fulfill({
			json: {
				profiles: responseProfiles.slice(100, 150),
				total: responseProfiles.length,
				hasNext: true,
			},
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	let rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 160');
	await expect.poll(() => continuationRequests).toBe(1);

	rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	const failedResponse = page.waitForResponse(
		(response) =>
			new URL(response.url()).pathname.endsWith('/v1/profiles/query') &&
			response.request().postDataJSON().first === 100 &&
			response.status() === 503,
	);
	await page.keyboard.press('ArrowDown');
	await failedResponse;
	await page.evaluate(() => new Promise(requestAnimationFrame));
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 160');
	await expect(rows.last()).toHaveClass(/grid__row--active/);
	await expect(grid).toBeFocused();
	expect(continuationRequests).toBe(2);

	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('101–150 of 160');
	await expect.poll(() => continuationRequests).toBe(3);
	rows = grid.locator('.grid__row--clickable');
	await expect(rows.first()).toHaveClass(/grid__row--active/);
});

test(`Keep the current page when continuation arguments no longer align with the schema`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 120 }, (_, index) => ({
		...profiles[index % profiles.length],
		kpid: `versioned-navigation-profile-${index + 1}`,
	}));

	await mockProfiles(page, { responseProfiles });
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (Number(request.postDataJSON().first) !== 100) {
			await route.fallback();
			return;
		}
		await route.fulfill({
			status: 422,
			json: { error: { code: 'SchemaNotAligned', message: 'Profile schema has changed' } },
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	let rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
	await expect(page.locator('.profiles-list__stale-notice')).toBeVisible();

	rows = grid.locator('.grid__row--clickable');
	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 120');
	await expect(rows.last()).toHaveClass(/grid__row--active/);
	await expect(grid).toBeFocused();
});

test(`Use one sequence navigator for manual pages and profile panel controls`, async ({ page }) => {
	const responseProfiles = Array.from({ length: 110 }, (_, index) => ({
		...profiles[index % profiles.length],
		kpid: `shared-navigation-profile-${index + 1}`,
	}));
	await mockProfiles(page, { responseProfiles });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const pagination = page.locator('.profiles-list__pagination');
	const previousPage = pagination.locator('.profiles-list__pagination-previous').getByRole('button');
	const nextPage = pagination.locator('.profiles-list__pagination-next').getByRole('button');
	let rows = page.locator('.profiles-list .grid__row--clickable');
	await rows.nth(10).click();
	await nextPage.focus();
	await page.keyboard.press('Enter');
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('shared-navigation-profile-51');
	rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(rows.first()).toHaveClass(/grid__row--active/);

	await previousPage.focus();
	await page.keyboard.press('Enter');
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('shared-navigation-profile-50');
	rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(rows.last()).toHaveClass(/grid__row--active/);

	const nextProfile = page.getByRole('button', { name: 'Next profile' });
	await nextProfile.click();
	await expect(nextProfile).toBeFocused();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 110');
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('shared-navigation-profile-51');

	const previousProfile = page.getByRole('button', { name: 'Previous profile' });
	await previousProfile.click();
	await expect(previousProfile).toBeFocused();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 110');
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('shared-navigation-profile-50');
});

test(`Build the fixed profile column from the assigned profile roles`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const profileCells = page.locator('.profiles-list__profile-cell');
	await expect(profileCells).toHaveCount(4);
	await expect(profileCells.nth(0).locator('.profiles-list__profile-thumbnail')).toHaveJSProperty(
		'image',
		profilePhoto,
	);
	await expect(profileCells.nth(0).locator('.profiles-list__profile-thumbnail')).toHaveCSS('width', '44px');
	await expect(profileCells.nth(0).locator('.profiles-list__profile-name')).toHaveText('Ada Lovelace');
	await expect(profileCells.nth(0).locator('.profiles-list__profile-country')).toHaveText('United Kingdom');

	await expect(profileCells.nth(1).locator('.profiles-list__profile-thumbnail')).toHaveJSProperty('initials', 'GH');
	await expect(profileCells.nth(1).locator('.profiles-list__profile-thumbnail')).toHaveCSS('width', '44px');
	await expect(profileCells.nth(1).locator('.profiles-list__profile-name')).toHaveText('Grace Hopper');
	await expect(profileCells.nth(1).locator('.profiles-list__profile-country')).toHaveText('United States');

	await expect(profileCells.nth(2).locator('.profiles-list__profile-thumbnail')).toHaveJSProperty('initials', 'C');
	await expect(profileCells.nth(2).locator('.profiles-list__profile-name')).toHaveText('Curie');
	await expect(profileCells.nth(2).locator('.profiles-list__profile-country')).toHaveCount(0);

	await expect(profileCells.nth(3).locator('.profiles-list__profile-thumbnail wa-icon[name="user"]')).toBeVisible();
	await expect(profileCells.nth(3).locator('.profiles-list__profile-name')).toHaveText('Name not available');
	await expect(profileCells.nth(3).locator('.profiles-list__profile-name')).toHaveCSS('font-style', 'italic');

	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	await columns.locator('wa-checkbox', { hasText: 'First name' }).click();
	await expect(page.locator('.profiles-list .grid__header-cell', { hasText: 'First name' })).toHaveCount(0);
	await expect(page.locator('.profiles-list__profile-name').first()).toHaveText('Ada Lovelace');
});

test(`Keep fallback avatar colors distinct and stable for each profile`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const profileCells = page.locator('.profiles-list__profile-cell');
	const graceAvatar = profileCells.nth(1).locator('.profiles-list__profile-thumbnail');
	const curieAvatar = profileCells.nth(2).locator('.profiles-list__profile-thumbnail');
	const getBackgroundColor = (element: Element): string => {
		return getComputedStyle(element).backgroundColor;
	};
	const graceBackgroundColor = await graceAvatar.evaluate(getBackgroundColor);
	const curieBackgroundColor = await curieAvatar.evaluate(getBackgroundColor);

	expect(graceBackgroundColor).not.toBe('');
	expect(graceBackgroundColor).not.toBe(curieBackgroundColor);
	await profileCells.nth(1).click();
	const drawerAvatar = page.locator('.profile-drawer__profile-thumbnail');
	await expect(drawerAvatar).toHaveJSProperty('initials', 'GH');
	await expect(drawerAvatar).toHaveCSS('width', '64px');
	expect(await drawerAvatar.evaluate(getBackgroundColor)).toBe(graceBackgroundColor);
	await page.reload();
	await expect(graceAvatar).toHaveJSProperty('initials', 'GH');
	expect(await graceAvatar.evaluate(getBackgroundColor)).toBe(graceBackgroundColor);
});

test(`Resolve current and withdrawn alpha-2 country codes`, async ({ page }) => {
	const responseProfiles = ['AN', 'AI', 'CS', 'ZZ'].map((country, index) => {
		const profile = profiles[index];
		return {
			...profile,
			attributes: { ...profile.attributes, country },
		};
	});
	await mockProfiles(page, { responseProfiles });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const profileCells = page.locator('.profiles-list__profile-cell');
	await expect(profileCells.nth(0).locator('.profiles-list__profile-country')).toHaveText('Netherlands Antilles');
	await expect(profileCells.nth(1).locator('.profiles-list__profile-country')).toHaveText('Anguilla');
	await expect(profileCells.nth(2).locator('.profiles-list__profile-country')).toHaveCount(0);
	await expect(profileCells.nth(3).locator('.profiles-list__profile-country')).toHaveCount(0);
});

test(`Resolve only known country codes in the alpha-3 format declared by the schema`, async ({ page }) => {
	const responseProfiles = ['ITA', 'ANT', 'IT', 'ZZZ', 'ita'].map((country, index) => {
		const profile = profiles[index % profiles.length];
		return {
			...profile,
			kpid: `alpha-3-profile-${index}`,
			attributes: { ...profile.attributes, country },
		};
	});
	await mockProfiles(page, {
		responseProfiles,
		schema: getProfileSchema('alpha-3'),
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const profileCells = page.locator('.profiles-list__profile-cell');
	await expect(profileCells.nth(0).locator('.profiles-list__profile-country')).toHaveText('Italy');
	await expect(profileCells.nth(1).locator('.profiles-list__profile-country')).toHaveText('Netherlands Antilles');
	await expect(profileCells.nth(2).locator('.profiles-list__profile-country')).toHaveCount(0);
	await expect(profileCells.nth(3).locator('.profiles-list__profile-country')).toHaveCount(0);
	await expect(profileCells.nth(4).locator('.profiles-list__profile-country')).toHaveCount(0);
});

test(`Omit unassigned profile roles and use the remaining name for initials`, async ({ page }) => {
	await mockProfiles(page, {
		propertyRoles: {
			...propertyRoles,
			firstName: '',
			country: '',
			photo: '',
		},
		responseProfiles: [profiles[0]],
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const profileCell = page.locator('.profiles-list__profile-cell');
	await expect(profileCell.locator('.profiles-list__profile-thumbnail')).toHaveJSProperty('initials', 'L');
	await expect(profileCell.locator('.profiles-list__profile-name')).toHaveText('Lovelace');
	await expect(profileCell.locator('.profiles-list__profile-country')).toHaveCount(0);
});

test(`Use profile property display names in the attributes drawer`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	await page.locator('.profiles-list .grid__row--clickable').first().click();
	const drawer = page.locator('.profile-drawer');
	await expect(drawer).toBeVisible();
	await expect(page.getByRole('complementary', { name: 'Profile details' })).toBeVisible();
	await expect(drawer.locator('.profile-drawer__profile-thumbnail')).toHaveJSProperty('image', profilePhoto);
	await expect(drawer.locator('.profile-drawer__profile-thumbnail')).toHaveCSS('width', '64px');
	const attributeKeys = drawer.locator('.profile-drawer__attribute-key');
	await expect(attributeKeys).toHaveText([
		'Customer ID:',
		'email:',
		'First name:',
		'Last name:',
		'Country:',
		'Photo:',
		'Billing address',
		'shipping_address',
	]);

	const parentAttributes = drawer.locator('.drawer-attributes--parent');
	await parentAttributes.nth(0).click();
	await parentAttributes.nth(1).click();
	await expect(attributeKeys).toHaveText([
		'Customer ID:',
		'email:',
		'First name:',
		'Last name:',
		'Country:',
		'Photo:',
		'Billing address',
		'City:',
		'shipping_address',
		'City:',
	]);
});

test(`Ignore late profile attributes and keep the non-modal panel from taking focus`, async ({ page }) => {
	let releaseFirstAttributes!: () => void;
	const firstAttributes = new Promise<void>((resolve) => {
		releaseFirstAttributes = resolve;
	});

	await mockProfiles(page);
	await page.route('**/v1/profiles/*/attributes/query', async (route) => {
		const url = new URL(route.request().url());
		const fragments = url.pathname.split('/');
		const kpid = decodeURIComponent(fragments[fragments.length - 3]);
		if (kpid === 'profile-1') {
			await firstAttributes;
		}
		await route.fulfill({
			json: {
				attributes: {
					...(profiles.find((candidate) => candidate.kpid === kpid)?.attributes ?? {}),
					customer_id: kpid === 'profile-1' ? 'late-profile-one' : 'current-profile-two',
				},
			},
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	const rows = grid.locator('.grid__row--clickable');
	await rows.nth(0).click();
	await rows.nth(1).click();
	await expect(page.locator('.profile-drawer')).toContainText('current-profile-two');
	await expect(page.getByRole('complementary', { name: 'Profile details' })).toBeVisible();
	await expect(page.locator('wa-drawer')).toHaveCount(0);
	await expect(grid).toBeFocused();

	releaseFirstAttributes();
	await page.evaluate(() => new Promise(requestAnimationFrame));
	await expect(page.locator('.profile-drawer')).not.toContainText('late-profile-one');
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-2');
});

test(`Report incompatible attribute schema without replacing displayed results`, async ({ page }) => {
	const summary = {
		...profiles[0],
		attributes: {
			first_name: 'Ada',
			last_name: 'Lovelace',
			email: 'one@example.com',
			country: 'GB',
		},
	};
	let requestedSchema: ObjectType | null = null;

	await mockProfiles(page, { responseProfiles: [summary] });
	await page.route('**/v1/profiles/*/attributes/query', async (route) => {
		const request = route.request();
		requestedSchema = request.postDataJSON().schema;
		expect(request.method()).toBe('POST');
		expect(new URL(request.url()).search).toBe('');
		await route.fulfill({
			status: 422,
			json: { error: { code: 'SchemaNotAligned', message: 'Profile schema has changed' } },
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	await grid.locator('.grid__row--clickable').click();
	await expect(page.locator('.profiles-list__stale-notice')).toBeVisible();
	await expect(page.locator('.profile-drawer')).not.toContainText('must-not-be-shown');
	await expect(grid).toBeFocused();
	expect(requestedSchema).toEqual(profileSchema);
});

test(`Navigate flat profile rows with the keyboard`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(rows).toHaveCount(4);
	const grid = page.locator('.profiles-list .grid');
	await grid.focus();
	await page.keyboard.press('ArrowDown');
	await expect(rows.nth(0)).toHaveClass(/grid__row--active/);
	const cellHeights = await rows
		.nth(0)
		.locator('.grid__cell')
		.evaluateAll((cells) => cells.map((cell) => cell.getBoundingClientRect().height));
	expect(new Set(cellHeights).size).toBe(1);
	await expect(page.locator('.profile-drawer')).toBeVisible();
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-1');
	await expect(grid).toBeFocused();
	await expect(grid).toHaveAttribute('aria-activedescendant', 'profiles-grid-row-profile-1');

	await page.keyboard.press('ArrowDown');
	await expect(rows.nth(1)).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-2');

	await page.keyboard.press('ArrowLeft');
	await expect(rows.nth(1)).toHaveClass(/grid__row--active/);
	await page.keyboard.press('ArrowUp');
	await expect(rows.nth(0)).toHaveClass(/grid__row--active/);
});

test(`Keep the first profile row below the sticky header when navigating upwards`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	await grid.evaluate((element) => {
		element.style.height = '160px';
	});
	const rows = grid.locator('.grid__row--clickable');
	await grid.focus();
	await page.keyboard.press('ArrowUp');
	await expect(rows.nth(3)).toHaveClass(/grid__row--active/);
	await page.keyboard.press('ArrowUp');
	await page.keyboard.press('ArrowUp');
	await page.keyboard.press('ArrowUp');
	await expect(rows.nth(0)).toHaveClass(/grid__row--active/);

	const positions = await grid.evaluate((element) => {
		const header = element.querySelector('.grid__header-row');
		const firstRow = element.querySelector('.grid__row--clickable');
		if (header == null || firstRow == null) {
			throw new Error('The profile grid header or first row is missing');
		}
		return {
			headerBottom: header.getBoundingClientRect().bottom,
			rowTop: firstRow.getBoundingClientRect().top,
		};
	});
	expect(positions.rowTop).toBeGreaterThanOrEqual(positions.headerBottom);
});

test(`Do not navigate profiles while a grid control or dialog is open`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const pageSizeCombobox = page.locator('.profiles-list__pagination-page-size').getByRole('combobox');
	await pageSizeCombobox.focus();
	await expect(pageSizeCombobox).toBeFocused();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list .grid__row--active')).toHaveCount(0);

	const columns = page.locator('.profiles-list__toggle-columns');
	const columnsButton = columns.getByRole('button', { name: 'Columns', exact: true });
	await columnsButton.click();
	await expect(columns).toHaveJSProperty('open', true);
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list .grid__row--active')).toHaveCount(0);
	await page.keyboard.press('Escape');
	await expect(columns).toHaveJSProperty('open', false);
	await expect(columnsButton).toBeFocused();

	await page.locator('.profiles-list__identity-resolution-button').click();
	const dialog = page.locator('.alert-dialog');
	await expect(dialog.getByRole('dialog')).toBeVisible();
	await page.keyboard.press('ArrowDown');
	await expect(page.locator('.profiles-list .grid__row--active')).toHaveCount(0);
	await dialog.getByText('Cancel', { exact: true }).click();
});

test(`Do not wrap profile navigation at the global sequence extremes`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	const rows = grid.locator('.grid__row--clickable');
	await rows.first().click();
	await page.keyboard.press('ArrowUp');
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-1');
	const previousProfile = page.getByRole('button', { name: 'Previous profile' });
	await expect(previousProfile).toBeDisabled();
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-1');

	await rows.last().click();
	await page.keyboard.press('ArrowDown');
	await expect(rows.last()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-4');
	const nextProfile = page.getByRole('button', { name: 'Next profile' });
	await expect(nextProfile).toBeDisabled();
	await expect(page.locator('.profile-drawer__kpid-value')).toHaveText('profile-4');
});

test(`Show the empty state without keyboard hints`, async ({ page }) => {
	await mockProfiles(page, { responseProfiles: [] });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('0 profiles');
	await expect(page.locator('.profiles-list .grid__no-rows')).toContainText('No profiles to show');
	await expect(page.locator('.profiles-list .grid-keyboard-hints')).toHaveCount(0);
});

test('Reload attributes when reopening a profile and handle its disappearance locally', async ({ page }) => {
	await mockProfiles(page);
	let attributeReads = 0;
	await page.route('**/v1/profiles/*/attributes/query', async (route) => {
		const request = route.request();
		expect(request.postDataJSON().schema).toEqual(profileSchema);
		expect(request.postDataJSON().expectedDatasetVersion).toBeUndefined();
		attributeReads++;
		if (attributeReads === 3) {
			await route.fulfill({
				status: 404,
				json: { error: { code: 'NotFound', message: 'Profile no longer exists' } },
			});
			return;
		}
		await route.fulfill({
			json: { attributes: { ...profiles[0].attributes, customer_id: 'live-' + attributeReads } },
		});
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);
	const firstRow = page.locator('.profiles-list .grid__row--clickable').first();
	await firstRow.click();
	await expect(page.locator('.profile-drawer')).toContainText('live-1');
	await page.getByRole('button', { name: 'Close profile details' }).click();
	await firstRow.click();
	await expect(page.locator('.profile-drawer')).toContainText('live-2');
	await page.getByRole('button', { name: 'Close profile details' }).click();
	await firstRow.click();
	await expect(page.locator('.profile-drawer')).toContainText('This profile no longer exists.');
	await expect(page.locator('.profile-drawer')).not.toContainText('live-2');
	await expect(page.locator('.profiles-list__stale-notice')).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
});

test('Allow an empty continuation and keep Previous and Refresh usable', async ({ page }) => {
	const responseProfiles = Array.from({ length: 120 }, (_, index) => ({
		...profiles[0],
		kpid: 'live-page-' + index,
	}));
	await mockProfiles(page, { responseProfiles });
	await page.route('**/v1/profiles/query', async (route) => {
		if (Number(route.request().postDataJSON().first) < 100) {
			await route.fallback();
			return;
		}
		await route.fulfill({ json: { profiles: [], total: 2, hasNext: false } });
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);
	const pagination = page.getByRole('navigation', { name: 'Profiles pagination' });
	await pagination.locator('.profiles-list__pagination-next').click();
	await expect(page.locator('.profiles-list__pagination-range')).toContainText('51–100');
	await pagination.locator('.profiles-list__pagination-next').click();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('No profiles on this page');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(0);
	await expect(pagination.getByRole('button', { name: 'Next page' })).toBeDisabled();
	await pagination.locator('.profiles-list__pagination-previous').click();
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(50);
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 120');
});

test('Reuse adjacent cached pages but refetch an evicted page against live data', async ({ page }) => {
	const responseProfiles = Array.from({ length: 360 }, (_, index) => ({
		...profiles[0],
		kpid: 'cache-profile-' + index,
	}));
	let firstPageReads = 0;
	await mockProfiles(page, { responseProfiles });
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		if (request.postDataJSON().first === 0) {
			firstPageReads++;
			const limit = Number(request.postDataJSON().limit);
			await route.fulfill({
				json: {
					profiles: responseProfiles.slice(0, limit).map((profile) => ({
						...profile,
						attributes: { ...profile.attributes, customer_id: 'observation-' + firstPageReads },
					})),
					total: 360,
					hasNext: true,
				},
			});
			return;
		}
		await route.fallback();
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);
	const pagination = page.getByRole('navigation', { name: 'Profiles pagination' });
	await expect(page.locator('.profiles-list .grid__row--clickable').first()).toContainText('observation-1');
	await pagination.locator('.profiles-list__pagination-next').click();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('51–100 of 360');
	await pagination.locator('.profiles-list__pagination-previous').click();
	await expect(page.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 360');
	expect(firstPageReads).toBe(1);
	for (let pageNumber = 1; pageNumber <= 5; pageNumber++) {
		await pagination.locator('.profiles-list__pagination-next').click();
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText(
			`${pageNumber * 50 + 1}–${pageNumber * 50 + 50} of 360`,
		);
	}
	for (let pageNumber = 4; pageNumber >= 0; pageNumber--) {
		await pagination.locator('.profiles-list__pagination-previous').click();
		await expect(page.locator('.profiles-list__pagination-range')).toHaveText(
			`${pageNumber * 50 + 1}–${pageNumber * 50 + 50} of 360`,
		);
	}
	expect(firstPageReads).toBe(2);
	await expect(page.locator('.profiles-list .grid__row--clickable').first()).toContainText('observation-2');
});

test('Refresh keeps filter arguments until Clear all filters reloads the schema', async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (request) => (request.postDataJSON().filter != null ? [profiles[1]] : profiles),
	});
	let schemaReads = 0;
	let incompatible = false;
	const resetSchema: ObjectType = {
		...profileSchema,
		properties: profileSchema.properties.filter((property) => property.name !== 'first_name'),
	};
	const requests: Request[] = [];
	await page.route('**/v1/profiles/schema', async (route) => {
		schemaReads++;
		await route.fulfill({
			json: {
				schema: schemaReads === 1 ? profileSchema : resetSchema,
				propertyRoles: schemaReads === 1 ? propertyRoles : { ...propertyRoles, firstName: '' },
				primarySources: {},
			},
		});
	});
	await page.route('**/v1/profiles/query', async (route) => {
		const request = route.request();
		requests.push(request);
		if (incompatible && request.postDataJSON().filter != null) {
			await route.fulfill({
				status: 422,
				json: { error: { code: 'SchemaNotAligned', message: 'Profile schema has changed' } },
			});
			return;
		}
		await route.fallback();
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);
	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Apply filters', exact: true }).click();
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(1);
	const applied = requests.at(-1)!.postDataJSON().filter;
	await page.locator('.profiles-list .grid__row--clickable').click();
	await valueInput.fill('Uncommitted draft');
	incompatible = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.locator('.profiles-list__stale-notice')).toBeVisible();
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(1);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--active')).toHaveAttribute('data-id', 'profile-2');
	await expect(page.locator('.profile-drawer')).toBeVisible();
	expect(requests.at(-1)!.postDataJSON().filter).toEqual(applied);
	expect(requests.at(-1)!.postDataJSON().schema).toEqual(profileSchema);
	expect(schemaReads).toBe(1);
	await expect(valueInput).toHaveValue('Uncommitted draft');
	await page.locator('.profiles-list__stale-notice').getByRole('button', { name: 'Clear all filters' }).click();
	await expect.poll(() => schemaReads).toBe(2);
	await expect(filters).toContainText('No filters applied');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
	expect(requests.at(-1)!.postDataJSON().filter).toBeUndefined();
	expect(requests.at(-1)!.postDataJSON().schema).toEqual(resetSchema);
	expect(requests.at(-1)!.postDataJSON().properties).toEqual(
		resetSchema.properties.map((property) => property.name).sort(),
	);
	await filters.getByRole('button', { name: 'Add filter', exact: true }).click();
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
});
