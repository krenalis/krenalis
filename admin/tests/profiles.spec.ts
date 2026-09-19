import { expect, Locator, Page, test } from '@playwright/test';
import { CountryFormat, ObjectType, Property } from '../src/lib/api/types/types';
import { ResponseProfile } from '../src/lib/api/types/responses';
import { ProfileRoleAssignments } from '../src/lib/api/types/workspace';
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
	role: 'Both',
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
		profileProperty('email', '   ', { type: { kind: 'string', semantic: 'email' } }),
		profileProperty('first_name', 'First name'),
		profileProperty('last_name', 'Last name'),
		profileProperty('country', 'Country', {
			type: { kind: 'string', semantic: 'country', format: 'alpha-2' },
		}),
		profileProperty('photo_url', 'Photo', { type: { kind: 'string', semantic: 'url' } }),
		profileObjectProperty('billing_address', [profileProperty('city', 'City')], 'Billing address'),
		profileObjectProperty('shipping_address', [profileProperty('city', 'City')]),
	],
};

const getProfileSchema = (countryFormat: CountryFormat): ObjectType => ({
	...profileSchema,
	properties: profileSchema.properties?.map((property) =>
		property.name === 'country'
			? { ...property, type: { kind: 'string', semantic: 'country', format: countryFormat } }
			: property,
	),
});

const profilePhoto = 'data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=';

const assignedRoles: ProfileRoleAssignments = {
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
	assignedRoles?: ProfileRoleAssignments;
	profilesForRequest?: (url: URL) => ResponseProfile[];
	responseProfiles?: ResponseProfile[];
	schema?: ObjectType;
}

const mockProfiles = async (
	page: Page,
	{
		assignedRoles: roleAssignments = assignedRoles,
		profilesForRequest,
		responseProfiles = profiles,
		schema = profileSchema,
	}: MockProfilesOptions = {},
) => {
	await page.route('**/v1/workspaces', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		for (const workspace of body.workspaces) {
			workspace.assignedRoles = roleAssignments;
		}
		await route.fulfill({ response, json: body });
	});
	await page.route('**/v1/profiles/schema', async (route) => {
		await route.fulfill({ json: schema });
	});
	await page.route('**/v1/profiles/count*', async (route) => {
		const url = new URL(route.request().url());
		const matchingProfiles = profilesForRequest?.(url) ?? responseProfiles;
		await route.fulfill({
			json: { total: matchingProfiles.length },
		});
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		const first = Number(url.searchParams.get('first'));
		const limit = Number(url.searchParams.get('limit'));
		const matchingProfiles = profilesForRequest?.(url) ?? responseProfiles;
		const resultProfiles = matchingProfiles.slice(first, first + limit);
		await route.fulfill({
			json: {
				profiles: resultProfiles,
				total: matchingProfiles.length,
				hasNext: first + resultProfiles.length < matchingProfiles.length,
			},
		});
	});
	await page.route('**/v1/profiles/*/attributes*', async (route) => {
		const fragments = new URL(route.request().url()).pathname.split('/');
		const kpid = decodeURIComponent(fragments[fragments.length - 2]);
		const profile = responseProfiles.find((candidate) => candidate.kpid === kpid);
		await route.fulfill({ json: { attributes: profile?.attributes ?? {} } });
	});
	await page.route('**/v1/identity-resolution/latest', async (route) => {
		await route.fulfill({ json: { startTime: null, endTime: '2026-08-03T12:00:00Z' } });
	});
};

const openFirstNameFilter = async (page: Page) => {
	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const condition = editor.locator('.pipeline__filters-filter');
	const property = condition.locator('.pipeline__filters-property');
	await property.locator('sl-input').click();
	await property.locator('input').fill('First');
	await property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^First name$/ }).click();

	return {
		filters,
		editor,
		condition,
		property,
		valueInput: condition.locator('.pipeline__filters-value-input input'),
	};
};

const fillStringFilterCondition = async (condition: Locator, propertyName: RegExp, value: string) => {
	const property = condition.locator('.pipeline__filters-property');
	await property.locator('sl-input').click();
	await property.locator('sl-menu-item .schema-combobox-item__name', { hasText: propertyName }).click();
	const operator = condition.locator('.pipeline__filters-operator').getByRole('combobox');
	await operator.press('Home');
	await operator.press('Enter');
	const valueInput = condition.locator('.pipeline__filters-value-input input');
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
	await expect(page.locator('.profiles-list__grid-summary-content sl-icon')).toHaveAttribute('name', 'people');
	await expect(page.getByRole('button', { name: 'Refine filters' })).toHaveCount(0);
	await expect(page.locator('.profiles-list__identity-resolution-button')).toHaveText('Run Profile Unification');
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
	await expect(hints).toHaveText('↑↓Navigate profiles');
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
	await expect(columns.locator('sl-button[slot="trigger"]')).toHaveText('Columns');
	await columnsButton.hover();
	await expect(page.locator('sl-tooltip[open]')).toHaveCount(0);
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
	await expect(columns.locator('sl-checkbox')).toHaveText([
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
	await expect(columns.locator('sl-checkbox')).toHaveText(['First name']);
	const clearColumnSearch = columns.getByRole('button', { name: 'Clear entry' });
	await expect(clearColumnSearch).toBeVisible();
	await clearColumnSearch.click();
	await expect(columnSearch).toHaveValue('');
	await expect(columns.locator('sl-checkbox')).toHaveCount(8);
	await expect(clearColumnSearch).toHaveCount(0);
	await columnSearch.fill('billing_address.city');
	await expect(columns.locator('sl-checkbox')).toHaveText(['Billing address › City']);
	await columnSearch.fill('does not exist');
	await expect(columns.locator('sl-checkbox')).toHaveCount(0);
	await expect(columns.locator('.profiles-list__column-chooser-empty')).toHaveText('No columns found');
	await columnSearch.fill('');

	const emailColumn = columns.locator('sl-checkbox', { hasText: 'email' });
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

test(`Toggle a column locally when its profile property is already loaded`, async ({ page }) => {
	let profileRequests = 0;
	let loadedProjection: string[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (!url.pathname.endsWith('/v1/profiles')) {
			return;
		}
		profileRequests++;
		loadedProjection = url.searchParams.get('properties')?.split(',') ?? [];
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	const shippingAddressColumn = columns.locator('sl-checkbox', { hasText: 'shipping_address › City' });
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
	let projectionRequest: URL | undefined;
	let initialProjection: string[] = [];
	await mockProfiles(page);
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		const properties = url.searchParams.get('properties')?.split(',') ?? [];
		if (!properties.includes('shipping_address')) {
			initialProjection = properties;
			await route.fallback();
			return;
		}
		projectionRequest = url;
		markProjectionStarted();
		await projectionCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const grid = page.locator('.profiles-list .grid');
	expect(initialProjection).not.toContain('shipping_address');
	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	const shippingAddressColumn = columns.locator('sl-checkbox', { hasText: 'shipping_address › City' });
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
	expect(projectionRequest?.searchParams.get('first')).toBe('0');
	expect(projectionRequest?.searchParams.get('limit')).toBe('100');
	expect(JSON.parse(projectionRequest!.searchParams.get('schema')!)).toEqual(profileSchema);
});

test(`Reorder profile columns without reloading their data and preserve the order`, async ({ page }) => {
	let gridRequests = 0;
	page.on('request', (request) => {
		if (new URL(request.url()).pathname.endsWith('/v1/profiles')) {
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
	const operatorSelect = condition.locator('.pipeline__filters-operator');
	await expect(operatorSelect.locator('sl-option[value="0"]')).toBeVisible();
	await page.clock.install();
	await operatorSelect.evaluate(async (select: any) => {
		await select.hide();
		await select.show();
	});
	// Exercise the delayed focus handoff that used to close a newly reopened menu.
	await page.clock.runFor(100);
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await operatorSelect.locator('sl-option[value="0"]').click();
	await expect(valueInput).toBeFocused();

	await operatorSelect.getByRole('combobox').click();
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await property.locator('input').click();
	await expect(operatorSelect).toHaveJSProperty('open', false);
	await page.clock.runFor(100);
	await expect(property.locator('input')).toBeFocused();
});

test(`Preview a completed filter and show its profiles explicitly`, async ({ page }) => {
	const countRequests: URL[] = [];
	const gridRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		} else if (url.pathname.endsWith('/v1/profiles')) {
			gridRequests.push(url);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const rows = page.locator('.profiles-list .grid__row--clickable');
	await expect(filters).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters');
	await expect(filters.locator('.profiles-list__filter-description')).not.toBeVisible();
	await expect(filters.getByRole('link', { name: 'Learn more about filters' })).not.toBeVisible();
	await expect(rows).toHaveCount(4);
	const editFilters = filters.getByRole('button', { name: 'Edit', exact: true });
	const filterTitle = filters.locator('.profiles-list__filter-title');
	const editFiltersLayout = await editFilters.boundingBox();
	const collapsedFilterTitleLayout = await filterTitle.boundingBox();
	await editFilters.click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	await expect(editor).toBeVisible();
	const closeFilters = filters.getByRole('button', { name: 'Done', exact: true });
	await expect(closeFilters).not.toBeFocused();
	await expect(filterTitle).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-summary-value')).not.toBeVisible();
	await expect(page.locator('.profiles-list__filter-preview-count')).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-preview')).toHaveCount(0);
	const gridCountLayout = await page.locator('.profiles-list__grid-summary').boundingBox();
	const editorLayout = await editor.boundingBox();
	const overviewLayout = await filters.locator('.profiles-list__filter-overview').boundingBox();
	const filterActionsLayout = await editor.locator('.pipeline__filters-group-actions').boundingBox();
	const closeFiltersLayout = await closeFilters.boundingBox();
	const expandedFilterTitleLayout = await filterTitle.boundingBox();
	expect(gridCountLayout).not.toBeNull();
	expect(editFiltersLayout).not.toBeNull();
	expect(collapsedFilterTitleLayout).not.toBeNull();
	expect(editorLayout).not.toBeNull();
	expect(overviewLayout).not.toBeNull();
	expect(filterActionsLayout).not.toBeNull();
	expect(closeFiltersLayout).not.toBeNull();
	expect(expandedFilterTitleLayout).not.toBeNull();
	expect(Math.abs(closeFiltersLayout!.x - editFiltersLayout!.x)).toBeLessThanOrEqual(1);
	expect(Math.abs(closeFiltersLayout!.y - editFiltersLayout!.y)).toBeLessThanOrEqual(1);
	expect(closeFiltersLayout!.width).toBe(editFiltersLayout!.width);
	expect(closeFiltersLayout!.height).toBe(editFiltersLayout!.height);
	expect(Math.abs(expandedFilterTitleLayout!.x - collapsedFilterTitleLayout!.x)).toBeLessThanOrEqual(1);
	expect(Math.abs(expandedFilterTitleLayout!.y - collapsedFilterTitleLayout!.y)).toBeLessThanOrEqual(1);
	expect(Math.abs(overviewLayout!.y - editorLayout!.y)).toBeLessThanOrEqual(1);
	expect(
		Math.abs(editorLayout!.y + editorLayout!.height - (filterActionsLayout!.y + filterActionsLayout!.height) - 36),
	).toBeLessThanOrEqual(1);
	await expect(filterTitle).toHaveText('Filters');
	await expect(filters.locator('.profiles-list__filter-description')).toHaveText(
		'Choose which profiles to include in the results. Leave empty to include all profiles.',
	);
	await expect(filters.getByRole('link', { name: 'Learn more about filters' })).toHaveAttribute(
		'href',
		'https://www.krenalis.com/docs/ref/admin/filters',
	);
	await expect(editor.locator('.pipeline__filters-filter')).toHaveCount(1);
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
	await expect.poll(() => gridRequests.length).toBe(1);
	await expect.poll(() => countRequests.length).toBe(0);
	await expect(page.locator('.profiles-list__layout')).not.toHaveAttribute('inert');
	expect(await editor.evaluate((element) => getComputedStyle(element).overflowY)).toBe('visible');
	const expandedLayout = await page.locator('.profiles-list__layout').boundingBox();
	expect(expandedLayout?.height).toBeGreaterThanOrEqual((await page.evaluate(() => window.innerHeight)) / 2 - 1);
	await expect(editor.getByRole('button', { name: 'Apply filters' })).toHaveCount(0);
	await expect(editor.getByRole('button', { name: 'Cancel' })).toHaveCount(0);
	await expect(filters.locator('.pipeline__filters-group-subject')).toHaveText('Profiles matching');

	const condition = filters.locator('.pipeline__filters-filter');
	const emptyConditionRemoveButton = filters.getByRole('button', {
		name: 'Remove condition 1',
		exact: true,
	});
	await expect(emptyConditionRemoveButton).toHaveCount(0);
	const property = condition.locator('.pipeline__filters-property');
	await property.locator('sl-input').click();
	await expect(property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^email$/ })).toBeVisible();
	await expect(
		property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^Billing address › City$/ }),
	).toBeVisible();
	const propertyMenu = property.locator('[data-is-combobox-list]');
	await propertyMenu.hover();
	await page.mouse.wheel(0, 200);
	await expect.poll(() => propertyMenu.evaluate((menu) => menu.scrollTop)).toBeGreaterThan(0);
	const propertyMenuLayout = await property.locator('[data-is-combobox-list]').evaluate((menu) => {
		const lastItem = menu.querySelector('sl-menu-item:last-of-type');
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
	await property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^First name$/ }).click();
	await expect(property.locator('input')).toHaveValue('First name');
	await expect(emptyConditionRemoveButton).toBeVisible();
	await expect(filters.locator('.pipeline__filters-logical')).toHaveJSProperty('hoist', true);
	const operatorSelect = condition.locator('.pipeline__filters-operator');
	await expect(operatorSelect).toHaveJSProperty('hoist', true);
	await expect(operatorSelect).toHaveJSProperty('open', true);
	await expect(operatorSelect.locator('sl-option[value="4"]')).toBeVisible();
	const operatorMenuLayout = await operatorSelect.evaluate((select) => {
		const visibleOption = select.querySelector('sl-option[value="4"]');
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
	await operatorSelect.locator('sl-option[value="0"]').click();
	const valueInput = condition.locator('.pipeline__filters-value-input input');
	await valueInput.fill('Grace');
	await expect.poll(() => countRequests.length).toBe(0);
	await valueInput.press('Enter');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(editor).toBeVisible();
	await expect(rows).toHaveCount(4);
	await rows.first().click();
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toBeVisible();
	const previewFilter = JSON.parse(countRequests.at(-1)?.searchParams.get('filter') ?? 'null');
	expect(previewFilter).toEqual({
		operator: 'and',
		rules: [{ property: 'first_name', operator: 'is', values: ['Grace'] }],
	});
	expect(countRequests.at(-1)?.searchParams.has('properties')).toBe(false);

	const updateResults = page.locator('sl-button.profiles-list__filter-update-results');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Show 1 match');
	await expect(updateResults).toHaveAttribute('size', 'small');
	await expect(updateResults.locator('sl-icon')).toHaveCount(0);
	await expect(updateResults).toContainText('Show 1 match');
	await expect(
		page.locator('.profiles-list__grid-summary-group').getByRole('button', { name: 'Show 1 match' }),
	).toBeVisible();
	const documentationLink = filters.getByRole('link', { name: 'Learn more about filters' });
	const documentationLayoutBeforeUpdate = await documentationLink.boundingBox();
	const gridSummaryLayout = await page.locator('.profiles-list__grid-summary').boundingBox();
	const updateResultsLayout = await updateResults.boundingBox();
	expect(documentationLayoutBeforeUpdate).not.toBeNull();
	expect(gridSummaryLayout).not.toBeNull();
	expect(updateResultsLayout).not.toBeNull();
	expect(updateResultsLayout!.x).toBeGreaterThan(gridSummaryLayout!.x + gridSummaryLayout!.width);
	expect(
		Math.abs(
			updateResultsLayout!.y +
				updateResultsLayout!.height / 2 -
				(gridSummaryLayout!.y + gridSummaryLayout!.height / 2),
		),
	).toBeLessThanOrEqual(1);
	await updateResults.click();
	await expect(updateResults).toHaveCount(0);
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
	await expect(page.locator('.profiles-list__filter-preview-count')).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(rows).toHaveCount(1);
	await expect(rows.first()).toContainText('Grace Hopper');
	await expect(rows.first()).not.toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid')).toBeFocused();
	expect(gridRequests.at(-1)?.searchParams.get('first')).toBe('0');

	const removeCondition = filters.getByRole('button', { name: 'Remove condition 1', exact: true });
	await expect(removeCondition).toHaveCount(1);
	await expect(removeCondition).toHaveJSProperty('tagName', 'BUTTON');
	await valueInput.focus();
	await page.keyboard.press('Tab');
	await expect(removeCondition).toBeFocused();
	const removeConditionFocusStyle = await removeCondition.evaluate((button) => ({
		outlineStyle: getComputedStyle(button).outlineStyle,
		outlineWidth: getComputedStyle(button).outlineWidth,
		focusRingWidth: getComputedStyle(document.documentElement).getPropertyValue('--sl-focus-ring-width').trim(),
	}));
	expect(removeConditionFocusStyle.outlineStyle).toBe('solid');
	expect(removeConditionFocusStyle.outlineWidth).toBe(removeConditionFocusStyle.focusRingWidth);
	await removeCondition.click();
	await expect(filters.locator('.filter-editor__announcement')).toHaveText('Condition removed');
	await expect(editor.getByRole('button', { name: 'Add a condition', exact: true })).toBeFocused();
	await expect(removeCondition).toHaveCount(0);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(rows).toHaveCount(1);
	await page.getByRole('button', { name: 'Show 4 matches', exact: true }).click();
	await expect(rows).toHaveCount(4);
	expect(gridRequests.at(-1)?.searchParams.has('filter')).toBe(false);

	const requestsBeforeCollapse = gridRequests.length;
	await closeFilters.click();
	await expect(editor).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters');
	await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeFocused();
	expect(gridRequests).toHaveLength(requestsBeforeCollapse);
});

test(`Offer Show matches only for a complete preview that differs from the grid`, async ({ page }) => {
	const countRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	const updateResults = page.getByRole('button', {
		name: 'Show 4 matches',
		exact: true,
	});
	await expect(updateResults).toBeVisible();

	await valueInput.fill('');
	await expect(updateResults).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-preview-incomplete')).toBeVisible();

	const requestsBeforeUndo = countRequests.length;
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(valueInput).toHaveValue('Grace');
	await expect(updateResults).toBeVisible();
	expect(countRequests).toHaveLength(requestsBeforeUndo);
});

test(`Recalculate previews when changing logical operators even in single-rule groups`, async ({ page }) => {
	const countRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	const showMatches = page.getByRole('button', { name: 'Show 4 matches', exact: true });
	await expect(showMatches).toBeVisible();
	expect(countRequests).toHaveLength(1);

	const rootGroup = editor.locator('.pipeline__filters-group--root');
	const rootLogical = rootGroup.locator(':scope > .pipeline__filters-group-header > .pipeline__filters-logical');
	await rootLogical.click();
	await rootLogical.locator('sl-option[value="or"]').click();
	await expect(rootLogical).toHaveJSProperty('value', 'or');
	await expect(showMatches).toBeVisible();
	expect(countRequests).toHaveLength(2);

	await rootGroup.locator(':scope > .pipeline__filters-group-actions > .pipeline__filters-add-group').click();
	const nestedGroup = editor.locator('.pipeline__filters-group').nth(1);
	const nestedCondition = nestedGroup.locator('.pipeline__filters-filter');
	const nestedProperty = nestedCondition.locator('.pipeline__filters-property');
	await nestedProperty.locator('sl-input').click();
	await nestedProperty.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^First name$/ }).click();
	await nestedCondition.locator('.pipeline__filters-operator sl-option[value="0"]').click();
	const nestedValue = nestedCondition.locator('.pipeline__filters-value-input input');
	await nestedValue.fill('Ada');
	await nestedValue.press('Enter');
	await expect.poll(() => countRequests.length).toBe(3);
	await expect(showMatches).toBeVisible();

	const nestedLogical = nestedGroup.locator(':scope > .pipeline__filters-group-header > .pipeline__filters-logical');
	await expect(nestedLogical).toHaveJSProperty('value', 'and');
	await nestedLogical.click();
	await nestedLogical.locator('sl-option[value="or"]').click();
	await expect(nestedLogical).toHaveJSProperty('value', 'or');
	await expect(showMatches).toBeVisible();
	expect(countRequests).toHaveLength(4);
});

test(`Summarize collapsed filters within their available width`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const longValue = 'A customer name that is deliberately long enough to require truncation in the summary';
	const { filters, editor, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill(longValue);
	await valueInput.press('Enter');

	const rootGroup = editor.locator('.pipeline__filters-group--root');
	const addRootGroup = rootGroup.locator(':scope > .pipeline__filters-group-actions > .pipeline__filters-add-group');
	await addRootGroup.click();
	const simpleGroup = editor.locator('.pipeline__filters-group').nth(1);
	await simpleGroup.locator(':scope > .pipeline__filters-group-actions > .pipeline__filters-add-condition').click();
	const simpleConditions = simpleGroup.locator('.pipeline__filters-filter');
	await fillStringFilterCondition(simpleConditions.nth(0), /^Last name$/, 'Hopper');
	await fillStringFilterCondition(simpleConditions.nth(1), /^Country$/, 'US');

	await addRootGroup.click();
	const complexGroup = editor.locator('.pipeline__filters-group').nth(2);
	const addComplexCondition = complexGroup.locator(
		':scope > .pipeline__filters-group-actions > .pipeline__filters-add-condition',
	);
	await addComplexCondition.click();
	await addComplexCondition.click();
	const complexConditions = complexGroup.locator('.pipeline__filters-filter');
	await fillStringFilterCondition(complexConditions.nth(0), /^email$/, 'one@example.com');
	await fillStringFilterCondition(complexConditions.nth(1), /^Customer ID$/, 'customer-1');
	await fillStringFilterCondition(complexConditions.nth(2), /^Billing address › City$/, 'Milan');

	await expect(page.getByRole('button', { name: 'Show 4 matches', exact: true })).toBeVisible();
	await filters.getByRole('button', { name: 'Done', exact: true }).click();
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
	const firstTooltip = visibleUnits.first().locator('sl-tooltip');
	await expect(visibleUnits.first().locator('.profiles-list__filter-summary-chip')).toHaveAccessibleName(
		`First name is "${longValue}"`,
	);
	await expect.poll(() => firstLabel.evaluate((element) => element.scrollWidth > element.clientWidth)).toBe(true);
	await expect(firstTooltip).toHaveJSProperty('disabled', false);
	await firstLabel.hover();
	await expect(firstTooltip).toHaveJSProperty('open', true);
	await expect(firstTooltip).toHaveJSProperty('content', `First name is "${longValue}"`);
	await expect(visibleUnits.first().locator('.profiles-list__filter-summary-chip-remove')).toHaveAccessibleName(
		`Remove condition: First name is "${longValue}"`,
	);
	await expect(visibleUnits.nth(2).locator('sl-tooltip')).toHaveJSProperty('disabled', true);

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

test(`Preview a text filter after one second of typing inactivity`, async ({ page }) => {
	const countRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Gra');
	await page.waitForTimeout(700);
	await valueInput.fill('Grace');
	await page.waitForTimeout(700);
	expect(countRequests).toHaveLength(0);

	await expect.poll(() => countRequests.length, { timeout: 1000 }).toBe(1);
	const previewFilter = JSON.parse(countRequests[0].searchParams.get('filter') ?? 'null');
	expect(previewFilter).toEqual({
		operator: 'and',
		rules: [{ property: 'first_name', operator: 'is', values: ['Grace'] }],
	});
});

test(`Treat zero matches as a preview that can be shown`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Missing');
	await valueInput.press('Enter');
	const updateResults = page.locator('sl-button.profiles-list__filter-update-results');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Show 0 matches');
	await expect(updateResults.locator('sl-icon')).toHaveCount(0);
	await expect(updateResults).toContainText('Show 0 matches');
	await updateResults.click();

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('0 profiles');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(0);
	await expect(page.locator('.profiles-list .grid__no-rows')).toContainText('No profiles to show');
});

test(`Keep the displayed grid intact when showing the preview fails`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (url) =>
			url.searchParams.has('filter')
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (!url.searchParams.has('filter')) {
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
	await page.getByRole('button', { name: 'Show 1 match', exact: true }).click();

	await expect(editor.locator('.profiles-list__filter-grid-error')).toContainText('Unable to show profiles.');
	await expect(
		editor.locator('.profiles-list__filter-grid-error').getByRole('button', { name: 'Retry' }),
	).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-summary-value')).not.toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(rows).toHaveCount(4);
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await expect(page.locator('.profile-drawer')).toBeVisible();
	await expect(filters).not.toHaveClass(/profiles-list__filters--updating-results/);
	await expect(filters).not.toHaveClass(/profiles-list__filters--results-updated/);
});

test(`Materialize the current Undo entry on Done and preserve its history`, async ({ page }) => {
	const gridRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles')) {
			gridRequests.push(url);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
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

	await filters.getByRole('button', { name: 'Done', exact: true }).click();
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(filters.getByRole('region', { name: 'Profile filters' })).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-title')).toHaveText('Filters');
	await expect(filters.locator('sl-button.profiles-list__filter-toggle')).toHaveText('Edit');
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
	await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeFocused();
	const materializedFilter = JSON.parse(gridRequests.at(-1)?.searchParams.get('filter') ?? 'null');
	expect(materializedFilter.rules[0].values).toEqual(['Grace']);

	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	await expect(filters.getByRole('region', { name: 'Profile filters' }).locator(':focus')).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Redo', exact: true })).toBeEnabled();
});

test(`Reopen the property menu after collapsing a nested property filter`, async ({ page }) => {
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	const condition = filters.locator('.pipeline__filters-filter');
	const property = condition.locator('.pipeline__filters-property');
	await property.locator('sl-input').click();
	await property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^Billing address › City$/ }).click();
	await condition.locator('.pipeline__filters-operator sl-option[value="0"]').click();
	await condition.locator('.pipeline__filters-value-input input').fill('Milan');
	await condition.locator('.pipeline__filters-value-input input').press('Enter');
	await filters.getByRole('button', { name: 'Done', exact: true }).click();

	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	const reopenedProperty = filters.locator('.pipeline__filters-property');
	await expect(reopenedProperty.locator('input')).toHaveValue('Billing address › City');
	await reopenedProperty.locator('sl-input').click();
	await expect(
		reopenedProperty.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^First name$/ }),
	).toBeVisible();
});

test(`Remove an applied filter from its collapsed chip, update the grid, and preserve history`, async ({ page }) => {
	const countRequests: URL[] = [];
	const gridRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		} else if (url.pathname.endsWith('/v1/profiles')) {
			gridRequests.push(url);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await filters.getByRole('button', { name: 'Done', exact: true }).click();
	await expect(
		filters.locator('.profiles-list__filter-summary-visible-chips .profiles-list__filter-summary-chip-label'),
	).toHaveText('First name is "Grace"');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');

	const countRequestsBeforeRemoval = countRequests.length;
	const gridRequestsBeforeRemoval = gridRequests.length;
	await filters.getByRole('button', { name: 'Remove condition: First name is "Grace"', exact: true }).click();

	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters');
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toHaveCSS(
		'animation-name',
		'profiles-list__grid-summary-success',
	);
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
	expect(countRequests).toHaveLength(countRequestsBeforeRemoval);
	expect(gridRequests).toHaveLength(gridRequestsBeforeRemoval + 1);
	expect(JSON.parse(gridRequests.at(-1)?.searchParams.get('filter') ?? 'null')).toBeNull();

	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const undo = filters.getByRole('button', { name: 'Undo', exact: true });
	const redo = filters.getByRole('button', { name: 'Redo', exact: true });
	await expect(undo).toBeEnabled();
	await undo.click();
	await expect(editor.locator('.pipeline__filters-property input')).toHaveValue('First name');
	await expect(editor.locator('.pipeline__filters-value-input input')).toHaveValue('Grace');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(redo).toBeEnabled();
	await redo.click();
	await expect(editor.locator('.pipeline__filters-property input')).toHaveValue('');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toHaveCount(0);
	await expect(undo).toBeEnabled();
	expect(countRequests).toHaveLength(countRequestsBeforeRemoval + 2);
});

test(`Keep filters expanded when Done cannot update the grid and discard the error after Undo`, async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (url) =>
			url.searchParams.has('filter')
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (!url.searchParams.has('filter')) {
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
	await filters.getByRole('button', { name: 'Done', exact: true }).click();

	await expect(filters.getByRole('region', { name: 'Profile filters' })).toBeVisible();
	await expect(filters.locator('.profiles-list__filter-grid-error')).toContainText('Unable to show profiles.');
	await expect(filters.locator('.profiles-list__filter-summary-value')).not.toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);

	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(page.locator('.profiles-list__filter-preview-count')).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(filters.locator('.profiles-list__filter-grid-error')).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Retry' })).toHaveCount(0);
});

test(`Grow nested filters in the document flow without an internal scrollbar`, async ({ page }) => {
	await page.setViewportSize({ width: 1366, height: 768 });
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	await editor.getByRole('button', { name: 'Add a group', exact: true }).click();
	const nestedGroup = editor.locator('.pipeline__filters-group:not(.pipeline__filters-group--root)');
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

test(`Keep the grid unchanged when a preview fails and allow a retry`, async ({ page }) => {
	let filteredCountRequests = 0;
	await mockProfiles(page, {
		profilesForRequest: (url) =>
			url.searchParams.has('filter')
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles/count*', async (route) => {
		const url = new URL(route.request().url());
		if (!url.searchParams.has('filter')) {
			await route.fallback();
			return;
		}
		filteredCountRequests++;
		if (filteredCountRequests === 1) {
			await route.fulfill({
				status: 503,
				json: { error: { code: 'ServiceUnavailable', message: 'temporary failure' } },
			});
			return;
		}
		await route.fallback();
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const rows = page.locator('.profiles-list .grid__row--clickable');
	const opened = await openFirstNameFilter(page);
	await expect(page.locator('.profiles-list__filter-grid-action')).toContainText(
		'Complete the filter to calculate matches',
	);
	await opened.valueInput.fill('Grace');
	await opened.valueInput.press('Enter');

	await expect(page.locator('.profiles-list__filter-grid-action')).toContainText('Unable to calculate matches.');
	await expect(editor).toBeVisible();
	await expect(opened.valueInput).toHaveValue('Grace');
	await expect(page.locator('.profiles-list__filter-preview-count')).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(rows).toHaveCount(4);
	await rows.first().click();
	await expect(rows.first()).toHaveClass(/grid__row--active/);

	await page.locator('.profiles-list__filter-grid-action').getByRole('button', { name: 'Retry' }).click();
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(rows).toHaveCount(4);
	expect(filteredCountRequests).toBe(2);
});

test(`Discard an in-flight preview when collapsing the editor`, async ({ page }) => {
	let markCandidateStarted = () => {};
	let releaseCandidate = () => {};
	const candidateStarted = new Promise<void>((resolve) => {
		markCandidateStarted = resolve;
	});
	const candidateCanFinish = new Promise<void>((resolve) => {
		releaseCandidate = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (url) =>
			url.searchParams.has('filter')
				? profiles.filter((profile) => profile.attributes.first_name === 'Grace')
				: profiles,
	});
	await page.route('**/v1/profiles/count*', async (route) => {
		const url = new URL(route.request().url());
		if (!url.searchParams.has('filter')) {
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
	await candidateStarted;

	await expect(rows).toHaveCount(4);
	await rows.first().click();
	await expect(rows.first()).toHaveClass(/grid__row--active/);
	await filters.getByRole('button', { name: 'Done', exact: true }).click();
	await expect(opened.editor).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeFocused();
	releaseCandidate();

	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters');
	await expect(rows).toHaveCount(4);
	await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeVisible();
});

test(`Ignore an obsolete preview after the draft changes`, async ({ page }) => {
	let markFirstCandidateStarted = () => {};
	let releaseFirstCandidate = () => {};
	const firstCandidateStarted = new Promise<void>((resolve) => {
		markFirstCandidateStarted = resolve;
	});
	const firstCandidateCanFinish = new Promise<void>((resolve) => {
		releaseFirstCandidate = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles/count*', async (route) => {
		const filter = JSON.parse(new URL(route.request().url()).searchParams.get('filter') ?? 'null');
		if (filter?.rules?.[0]?.values?.[0] !== 'Grace') {
			await route.fallback();
			return;
		}
		markFirstCandidateStarted();
		await firstCandidateCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await firstCandidateStarted;

	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);

	releaseFirstCandidate();
	await page.waitForTimeout(100);
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');
});

test(`Restore the current successful preview when collapsing an incomplete draft`, async ({ page }) => {
	const countRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		}
	});
	await mockProfiles(page);
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const filters = page.locator('.profiles-list__filters');
	const trigger = filters.getByRole('button', { name: 'Edit', exact: true });
	await trigger.click();
	const editor = filters.getByRole('region', { name: 'Profile filters' });
	const property = editor.locator('.pipeline__filters-property');
	await property.locator('sl-input').click();
	await property.locator('input').fill('First');
	await property.locator('sl-menu-item .schema-combobox-item__name', { hasText: /^First name$/ }).click();
	await expect(page.locator('.profiles-list__layout')).not.toHaveAttribute('inert');
	await expect(editor.locator('.pipeline__filters-operator')).toHaveJSProperty('open', true);
	await page.keyboard.press('Escape');
	await expect(editor).toBeVisible();
	await page.keyboard.press('Escape');

	await expect(editor).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Edit', exact: true })).toBeFocused();
	await expect(filters.locator('.profiles-list__filter-summary-value')).toHaveText('No filters');
	expect(countRequests).toHaveLength(0);

	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	await expect(editor.locator('.pipeline__filters-filter')).toHaveCount(1);
	await expect(editor.locator('.pipeline__filters-property input')).toHaveValue('');
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
	expect(countRequests).toHaveLength(0);
});

test(`Undo and redo successful filter previews without changing the grid`, async ({ page }) => {
	const countRequests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles/count')) {
			countRequests.push(url);
		}
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { filters, editor, condition, valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('4 profiles');

	await valueInput.fill('');
	const requestCountBeforeDraftUndo = countRequests.length;
	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(valueInput).toHaveValue('Grace');
	expect(countRequests).toHaveLength(requestCountBeforeDraftUndo);

	await filters.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(page.locator('.profiles-list__filter-preview-count')).toHaveCount(0);
	await expect(page.locator('.profiles-list__filter-update-results')).toHaveCount(0);
	await expect(editor.locator('.pipeline__filters-filter')).toHaveCount(1);
	await expect(editor.locator('.pipeline__filters-property input')).toHaveValue('');
	await expect(editor.getByRole('button', { name: 'Add filter', exact: true })).toHaveCount(0);
	await expect(filters.getByRole('button', { name: 'Redo', exact: true })).toBeEnabled();
	await filters.getByRole('button', { name: 'Redo', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(condition.locator('.pipeline__filters-value-input input')).toHaveValue('Grace');
	expect(countRequests).toHaveLength(requestCountBeforeDraftUndo + 2);
});

test(`Keep an explicit grid request independent from a newer filter preview`, async ({ page }) => {
	let markGraceGridStarted = () => {};
	let releaseGraceGrid = () => {};
	const graceGridStarted = new Promise<void>((resolve) => {
		markGraceGridStarted = resolve;
	});
	const graceGridCanFinish = new Promise<void>((resolve) => {
		releaseGraceGrid = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const filter = JSON.parse(new URL(route.request().url()).searchParams.get('filter') ?? 'null');
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
	const updateResults = page.locator('sl-button.profiles-list__filter-update-results');
	await expect(updateResults.getByRole('button')).toHaveAccessibleName('Show 1 match');
	await updateResults.click();
	await graceGridStarted;
	await expect(updateResults.locator('sl-spinner')).toBeVisible();
	await expect(filters).toHaveClass(/profiles-list__filters--updating-results/);
	await expect(page.locator('.profiles-list__grid-summary-content')).toContainText('4 profiles');
	await expect(page.locator('.profiles-list__grid-summary-sweep')).toBeVisible();

	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await expect(page.getByRole('button', { name: 'Show 1 match', exact: true })).toBeVisible();
	await expect(valueInput).toBeFocused();
	releaseGraceGrid();

	await expect(page.locator('.profiles-list__grid-summary')).toContainText('1 profile');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Grace Hopper']);
	await expect(filters).not.toHaveClass(/profiles-list__filters--updating-results/);
	await expect(filters).toHaveClass(/profiles-list__filters--results-updated/);
	await expect(valueInput).toHaveValue('Ada');
	await expect(valueInput).toBeFocused();
	await expect(page.locator('.profiles-list .grid')).not.toBeFocused();
});

test(`Ignore Escape while the current filter is being materialized`, async ({ page }) => {
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
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const filter = JSON.parse(new URL(route.request().url()).searchParams.get('filter') ?? 'null');
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
	await page.getByRole('button', { name: 'Show 1 match', exact: true }).click();
	await gridStarted;

	await valueInput.focus();
	await valueInput.press('Escape');
	releaseGrid();

	await expect(page.locator('.profiles-list .grid__row--clickable')).toContainText(['Grace Hopper']);
	await expect(editor).toBeVisible();
	expect(filteredGridRequests).toBe(1);
});

test(`Let a newer explicit Show request supersede an older one`, async ({ page }) => {
	let markGraceGridStarted = () => {};
	let releaseGraceGrid = () => {};
	const graceGridStarted = new Promise<void>((resolve) => {
		markGraceGridStarted = resolve;
	});
	const graceGridCanFinish = new Promise<void>((resolve) => {
		releaseGraceGrid = resolve;
	});
	await mockProfiles(page, {
		profilesForRequest: (url) => {
			const filter = JSON.parse(url.searchParams.get('filter') ?? 'null');
			const value = filter?.rules?.[0]?.values?.[0];
			return value == null ? profiles : profiles.filter((profile) => profile.attributes.first_name === value);
		},
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const filter = JSON.parse(new URL(route.request().url()).searchParams.get('filter') ?? 'null');
		if (filter?.rules?.[0]?.values?.[0] !== 'Grace') {
			await route.fallback();
			return;
		}
		markGraceGridStarted();
		await graceGridCanFinish;
		await route.fallback().catch(() => undefined);
	});
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const { valueInput } = await openFirstNameFilter(page);
	await valueInput.fill('Grace');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Show 1 match', exact: true }).click();
	await graceGridStarted;
	await valueInput.fill('Ada');
	await valueInput.press('Enter');
	await page.getByRole('button', { name: 'Show 1 match', exact: true }).click();

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
	const requests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles')) {
			requests.push(url);
		}
	});
	await mockProfiles(page, { responseProfiles });
	await page.goto(`${adminURL}/profile-unification/profiles`);

	const rows = page.locator('.profiles-list .grid__row--clickable');
	const pagination = page.locator('.profiles-list__pagination');
	const previousPage = pagination.locator('.profiles-list__pagination-previous').getByRole('button');
	const nextPage = pagination.locator('.profiles-list__pagination-next').getByRole('button');
	const pageSize = pagination.locator('sl-select');
	const pageSizeLabel = pageSize.locator('[part="form-control-label"]');
	const pageSizeCombobox = pageSize.getByRole('combobox');
	const pageNavigation = pagination.locator('.profiles-list__pagination-navigation');
	await expect(rows).toHaveCount(50);
	await expect(page.locator('.profiles-list__grid-summary')).toContainText('55 profiles');
	await expect(pagination).toHaveAttribute('aria-label', 'Profiles pagination');
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–50 of 55');
	await expect(pageSizeLabel).toHaveText('Profiles per page');
	await expect(pageSizeCombobox).toHaveAccessibleName('Profiles per page');
	await expect(pageSize.locator('sl-option')).toHaveText(['25', '50', '100']);
	await expect(pageNavigation.locator('sl-button')).toHaveCount(2);
	await expect(previousPage).toHaveAccessibleName('Previous page');
	await expect(nextPage).toHaveAccessibleName('Next page');
	await expect(previousPage).toHaveAttribute('aria-disabled', 'true');
	await expect(nextPage).toHaveAttribute('aria-disabled', 'false');
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
	expect(requests.at(-1)?.searchParams.get('first')).toBe('0');
	expect(requests.at(-1)?.searchParams.get('limit')).toBe('100');
	expect(requests.at(-1)?.searchParams.has('includeSchema')).toBe(false);
	expect(JSON.parse(requests.at(-1)!.searchParams.get('schema')!)).toEqual(profileSchema);
	expect(requests.at(-1)?.searchParams.get('properties')?.split(',').sort()).toEqual([
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
	await expect(previousPage).toHaveAttribute('aria-disabled', 'false');
	await expect(nextPage).toHaveAttribute('aria-disabled', 'true');
	expect(requests).toHaveLength(requestCountBeforeCachedPage);

	await pageSizeCombobox.click();
	const selectedPageSize = pageSize.locator('sl-option[aria-selected="true"]');
	await expect(selectedPageSize).toHaveText('50');
	await expect(selectedPageSize.locator('[part="checked-icon"]')).toBeVisible();
	await pageSize.locator('sl-option[value="100"]').click();
	await expect(rows).toHaveCount(55);
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–55 of 55');
	await expect(pageSize).toHaveJSProperty('value', '100');
	const pageSizeMetrics = await pageSize.evaluate((element) => {
		const label = element.shadowRoot?.querySelector('[part="form-control-label"]');
		const input = element.shadowRoot?.querySelector('.select__display-input');
		if (!(label instanceof HTMLElement) || !(input instanceof HTMLInputElement)) {
			throw new Error('The page size label or input is missing');
		}
		return {
			inputFontSize: getComputedStyle(input).fontSize,
			inputWidth: input.clientWidth,
			labelFontSize: getComputedStyle(label).fontSize,
			textWidth: input.scrollWidth,
		};
	});
	expect(pageSizeMetrics.inputFontSize).toBe(pageSizeMetrics.labelFontSize);
	expect(pageSizeMetrics.textWidth).toBeLessThanOrEqual(pageSizeMetrics.inputWidth);
	expect(requests.at(-1)?.searchParams.get('first')).toBe('0');
	expect(requests.at(-1)?.searchParams.get('limit')).toBe('200');

	await pageSizeCombobox.click();
	await pageSize.locator('sl-option[value="25"]').click();
	await expect(rows).toHaveCount(25);
	await expect(pagination.locator('.profiles-list__pagination-range')).toHaveText('1–25 of 55');
	await expect(pageSize).toHaveJSProperty('value', '25');
	expect(requests.at(-1)?.searchParams.get('first')).toBe('0');
	expect(requests.at(-1)?.searchParams.get('limit')).toBe('50');

	const responsiveLayout = await pagination.evaluate((element) => {
		const results = element.querySelector('.profiles-list__pagination-results');
		const controls = element.querySelector('.profiles-list__pagination-controls');
		const pageSize = element.querySelector('.profiles-list__pagination-page-size');
		const pageSizeLabel = pageSize?.shadowRoot?.querySelector('[part="form-control-label"]');
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
	const requests: URL[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.endsWith('/v1/profiles')) {
			requests.push(url);
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
	expect(requests.at(-1)?.searchParams.get('first')).toBe('100');
	expect(requests.at(-1)?.searchParams.get('limit')).toBe('50');
	expect(JSON.parse(requests.at(-1)!.searchParams.get('schema')!)).toEqual(profileSchema);

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
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (Number(url.searchParams.get('first')) !== 100) {
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
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (Number(url.searchParams.get('first')) !== 100) {
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
		(response) => new URL(response.url()).searchParams.get('first') === '100' && response.status() === 503,
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
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (Number(url.searchParams.get('first')) !== 100) {
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

	await expect(profileCells.nth(3).locator('.profiles-list__profile-thumbnail')).toHaveCount(0);
	await expect(profileCells.nth(3).locator('.profiles-list__profile-details')).toHaveCount(0);

	const columns = page.locator('.profiles-list__toggle-columns');
	await columns.getByRole('button', { name: 'Columns', exact: true }).click();
	await columns.locator('sl-checkbox', { hasText: 'First name' }).click();
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
		const base = element.shadowRoot?.querySelector('[part="base"]');
		return base == null ? '' : getComputedStyle(base).backgroundColor;
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
		assignedRoles: {
			...assignedRoles,
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
	await page.route('**/v1/profiles/*/attributes*', async (route) => {
		const url = new URL(route.request().url());
		const fragments = url.pathname.split('/');
		const kpid = decodeURIComponent(fragments[fragments.length - 2]);
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
	await expect(page.locator('sl-drawer')).toHaveCount(0);
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
	await page.route('**/v1/profiles/*/attributes*', async (route) => {
		const url = new URL(route.request().url());
		requestedSchema = JSON.parse(url.searchParams.get('schema')!);
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
	await expect(dialog).toBeVisible();
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
	await page.route('**/v1/profiles/*/attributes*', async (route) => {
		const url = new URL(route.request().url());
		expect(JSON.parse(url.searchParams.get('schema')!)).toEqual(profileSchema);
		expect(url.searchParams.has('expectedDatasetVersion')).toBe(false);
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
	await page.route('**/v1/profiles?*', async (route) => {
		if (Number(new URL(route.request().url()).searchParams.get('first')) < 100) {
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
	await expect(pagination.getByRole('button', { name: 'Next page' })).toHaveAttribute('aria-disabled', 'true');
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
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		if (url.searchParams.get('first') === '0') {
			firstPageReads++;
			const limit = Number(url.searchParams.get('limit'));
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

test('Refresh keeps filter arguments and reloads the schema only after reset confirmation', async ({ page }) => {
	await mockProfiles(page, {
		profilesForRequest: (url) => (url.searchParams.has('filter') ? [profiles[1]] : profiles),
	});
	let schemaReads = 0;
	let incompatible = false;
	const requests: URL[] = [];
	await page.route('**/v1/profiles/schema', async (route) => {
		schemaReads++;
		await route.fulfill({ json: profileSchema });
	});
	await page.route('**/v1/profiles?*', async (route) => {
		const url = new URL(route.request().url());
		requests.push(url);
		if (incompatible && url.searchParams.has('filter')) {
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
	await page.getByRole('button', { name: 'Show 1 match', exact: true }).click();
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(1);
	const applied = requests.at(-1)!.searchParams.get('filter');
	await valueInput.fill('Uncommitted draft');
	incompatible = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.locator('.profiles-list__stale-notice')).toBeVisible();
	expect(requests.at(-1)!.searchParams.get('filter')).toBe(applied);
	expect(JSON.parse(requests.at(-1)!.searchParams.get('schema')!)).toEqual(profileSchema);
	expect(schemaReads).toBe(1);
	await expect(valueInput).toHaveValue('Uncommitted draft');
	await page.locator('.profiles-list__stale-notice').getByRole('button', { name: 'Reset view' }).click();
	const dialog = page.locator('sl-dialog', { hasText: 'Reset profile view?' });
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: 'Cancel' }).click();
	await expect(valueInput).toHaveValue('Uncommitted draft');
	expect(schemaReads).toBe(1);
	await page.locator('.profiles-list__stale-notice').getByRole('button', { name: 'Reset view' }).click();
	await dialog.getByRole('button', { name: 'Reset view' }).click();
	await expect.poll(() => schemaReads).toBe(2);
	await expect(filters).toContainText('No filters');
	await expect(page.locator('.profiles-list .grid__row--clickable')).toHaveCount(4);
	expect(requests.at(-1)!.searchParams.has('filter')).toBe(false);
	await filters.getByRole('button', { name: 'Edit', exact: true }).click();
	await expect(filters.getByRole('button', { name: 'Undo', exact: true })).toBeDisabled();
});
