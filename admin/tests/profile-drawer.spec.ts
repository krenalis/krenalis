// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';

const rootAttributes = {
	first_name: 'Conventional first name',
	last_name: 'Conventional last name',
	email: 'conventional@example.test',
	given: 'Root given name',
	family: 'Root family name',
	mail: 'root@example.test',
	avatar: 'https://example.test/wrong-photo.svg',
};

let componentScript: string;
let componentStyles: string;

test.beforeAll(async () => {
	const result = await build({
		entryPoints: [resolve(__dirname, 'profile-drawer.fixture.tsx')],
		bundle: true,
		write: false,
		platform: 'browser',
		format: 'iife',
		outdir: '/tmp/profile-drawer-fixture',
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	componentScript = result.outputFiles.find((file) => file.path.endsWith('.js'))!.text;
	componentStyles = result.outputFiles.find((file) => file.path.endsWith('.css'))!.text;
});

test.beforeEach(async ({ page }) => {
	await page.route('**/*', (route) => route.abort());
	await page.route('https://example.test/', (route) =>
		route.fulfill({ contentType: 'text/html', body: '<div id="root"></div>' }),
	);
	await page.goto('https://example.test/');
	await page.addStyleTag({ content: componentStyles });
});

test('displays profile information from roles assigned to nested properties', async ({ page }) => {
	await page.evaluate((attributes) => {
		window.profileDrawerAttributes = {
			...attributes,
			person: { given: 'Ada', family: 'Lovelace' },
			contact: { mail: 'ada@example.test' },
			media: { avatar: 'https://example.test/photo.svg' },
		};
	}, rootAttributes);
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.profile-drawer__top-section')).toBeVisible();
	await expect(page.locator('.profile-drawer__first-name')).toBeVisible();
	await expect(page.locator('.profile-drawer__first-name')).toHaveText('Ada');
	await expect(page.locator('.profile-drawer__last-name')).toBeVisible();
	await expect(page.locator('.profile-drawer__last-name')).toHaveText('Lovelace');
	await expect(page.locator('.profile-drawer__email')).toBeVisible();
	await expect(page.locator('.profile-drawer__email')).toHaveText('ada@example.test');
	await expect(page.locator('.profile-drawer__photo')).toBeVisible();
	await expect(page.locator('.profile-drawer__photo')).toHaveJSProperty('image', 'https://example.test/photo.svg');
});

test('leaves profile information empty when an intermediate property is missing', async ({ page }) => {
	await page.evaluate((attributes) => {
		window.profileDrawerAttributes = attributes;
	}, rootAttributes);
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.drawer-attribute__value').filter({ hasText: 'Root given name' })).toBeVisible();
	await expect(page.locator('.profile-drawer__first-name')).toHaveText('');
	await expect(page.locator('.profile-drawer__last-name')).toHaveText('');
	await expect(page.locator('.profile-drawer__email')).toHaveText('');
	await expect(page.locator('.profile-drawer__photo')).toHaveJSProperty('image', '');
});
