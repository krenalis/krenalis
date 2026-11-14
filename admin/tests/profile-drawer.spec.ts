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
	await expect(page.locator('.profile-drawer__profile-thumbnail')).toBeVisible();
	await expect(page.locator('.profile-drawer__profile-thumbnail')).toHaveJSProperty(
		'image',
		'https://example.test/photo.svg',
	);
});

test('shows an avatar and missing name when an intermediate property is missing', async ({ page }) => {
	await page.evaluate((attributes) => {
		window.profileDrawerAttributes = attributes;
	}, rootAttributes);
	await page.addScriptTag({ content: componentScript });
	await expect(page.locator('.drawer-attribute__value').filter({ hasText: 'Root given name' })).toBeVisible();
	await expect(page.locator('.profile-drawer__name--unavailable')).toHaveText('Name not available');
	await expect(page.locator('.profile-drawer__name--unavailable')).toHaveCSS('font-style', 'italic');
	await expect(page.locator('.profile-drawer__email')).toHaveText('');
	await expect(page.locator('.profile-drawer__profile-thumbnail')).toBeVisible();
	await expect(page.locator('.profile-drawer__profile-thumbnail wa-icon[name="user"]')).toBeVisible();
});

test('animates closing before removing the drawer and reopens with reduced motion', async ({ page }) => {
	await page.evaluate((attributes) => {
		window.profileDrawerAttributes = attributes;
	}, rootAttributes);
	await page.addScriptTag({ content: componentScript });
	const drawer = page.locator('.profile-drawer');
	await expect(drawer).toBeVisible();
	await expect(drawer).toHaveCSS('animation-name', 'profile-drawer-enter');
	await expect(drawer).toHaveCSS('animation-duration', '0.25s');
	await expect(drawer).toContainText('Root given name');

	// Pause the actual exit animation so its timing and intermediate state can be inspected.
	await drawer.evaluate((element) => {
		element.addEventListener('animationstart', (event) => {
			if (
				event.target === element &&
				event instanceof AnimationEvent &&
				event.animationName === 'profile-drawer-exit'
			) {
				for (const animation of element.getAnimations()) {
					animation.pause();
				}
			}
		});
	});
	await drawer.locator('.profile-drawer__close').click();
	await expect(drawer).toHaveClass(/profile-drawer--closing/);
	expect(await drawer.evaluate((element) => element.getAnimations()[0].effect.getTiming().duration)).toBe(160);
	await expect(drawer).toHaveAttribute('inert', '');
	await expect(drawer).toContainText('Root given name');
	await expect(page.getByRole('complementary', { name: 'Profile details' })).toHaveCount(0);
	await page.evaluate((attributes) => {
		window.profileDrawerAttributes = { ...attributes, given: 'Updated given name' };
	}, rootAttributes);
	await page.getByRole('button', { name: 'Open profile details', exact: true }).click();
	await expect(drawer).not.toHaveClass(/profile-drawer--closing/);
	await expect(drawer).toContainText('Updated given name');
	await drawer.locator('.profile-drawer__close').click();
	await expect(drawer).toHaveClass(/profile-drawer--closing/);
	await drawer.evaluate((element) => {
		for (const animation of element.getAnimations()) {
			animation.finish();
		}
	});
	await expect(drawer).toHaveCount(0);

	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.getByRole('button', { name: 'Open profile details', exact: true }).click();
	await expect(drawer).toBeVisible();
	await expect(drawer).not.toHaveClass(/profile-drawer--closing/);
	await expect(drawer).toHaveCSS('animation-duration', '0s');
	await drawer.locator('.profile-drawer__close').click();
	await expect(drawer).toHaveCount(0);
});

test('fades profile changes without remounting the drawer and cancels superseded animations', async ({ page }) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.evaluate(() => {
		window.profileDrawerAttributes = { person: { given: 'Ada', family: 'Lovelace' } };
	});
	await page.addScriptTag({ content: componentScript });
	const drawer = page.locator('.profile-drawer');
	const summary = drawer.locator('.profile-drawer__top-section');
	await expect(drawer.locator('.profile-drawer__first-name')).toHaveText('Ada');
	const tabs = await drawer.locator('wa-tab-group').elementHandle();
	await page.emulateMedia({ reducedMotion: 'no-preference' });
	await page.evaluate(() => {
		// Pause content animations so intermediate frames and cancellation can be inspected.
		const animate = Element.prototype.animate;
		Element.prototype.animate = function (keyframes, options) {
			const animation = animate.call(this, keyframes, options);
			if (this.matches('.profile-drawer__top-section, .profile-drawer > wa-tab-group')) {
				animation.pause();
			}
			return animation;
		};
		window.profileDrawerAttributes = { person: { given: 'Grace', family: 'Hopper' } };
	});
	await drawer.locator('wa-button[aria-label="Next profile"]').click();
	await expect(drawer.locator('.profile-drawer__first-name')).toHaveText('Grace');
	const previousAnimation = await summary.evaluateHandle((element) => element.getAnimations()[0]);
	const frame = await summary.evaluate((element) => {
		const animation = element.getAnimations()[0];
		animation.currentTime = 90;
		return { duration: animation.effect.getTiming().duration, opacity: Number(getComputedStyle(element).opacity) };
	});
	expect(frame.duration).toBe(180);
	expect(frame.opacity).toBeGreaterThan(0);
	expect(frame.opacity).toBeLessThan(1);
	expect(await drawer.evaluate((element) => element.getAnimations().length)).toBe(0);
	expect(await tabs.evaluate((element) => element.isConnected)).toBe(true);

	await page.evaluate(() => {
		window.profileDrawerAttributes = { person: { given: 'Ada', family: 'Lovelace' } };
	});
	await drawer.locator('wa-button[aria-label="Previous profile"]').click();
	await expect(drawer.locator('.profile-drawer__first-name')).toHaveText('Ada');
	expect(await previousAnimation.evaluate((animation) => animation.playState)).toBe('idle');
	expect(await tabs.evaluate((element) => element.isConnected)).toBe(true);

	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.evaluate(() => {
		window.profileDrawerAttributes = { person: { given: 'Grace', family: 'Hopper' } };
	});
	await drawer.locator('wa-button[aria-label="Next profile"]').click();
	await expect(drawer.locator('.profile-drawer__first-name')).toHaveText('Grace');
	expect(await summary.evaluate((element) => element.getAnimations().length)).toBe(0);
});
