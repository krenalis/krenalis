// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import { expect, test, type Page, type Route } from '@playwright/test';
import { randomUUID } from 'crypto';
import { build } from 'esbuild';
import { resolve } from 'path';

let componentScript: string;

test.beforeAll(async () => {
	const bundle = await build({
		entryPoints: [resolve(__dirname, 'connection-events.fixture.tsx')],
		bundle: true,
		write: false,
		loader: { '.css': 'empty' },
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	componentScript = bundle.outputFiles[0].text;
});

const openLiveEvents = async (page: Page) => {
	await page.route('**/*', (route) => route.abort());
	await page.route('https://example.test/admin/connections/A/events', (route) =>
		route.fulfill({ contentType: 'text/html', body: '<div id="root"></div>' }),
	);
	await page.route('**/v1/connections/*', (route) => route.fulfill({ json: {} }));
	await page.route('**/v1/connections/*/pipeline-types', (route) => route.fulfill({ json: [] }));
	const headers = { 'Sync-Token': 'test' };
	const listeners = new Map<string, string>();
	await page.route('**/v1/events/listeners', (route) => {
		const id = randomUUID();
		listeners.set(id, route.request().postDataJSON().connection);
		return route.fulfill({ headers, json: { id } });
	});
	await page.route('**/v1/events/listeners/*', (route) => {
		const id = route.request().url().split('/').pop();
		const connection = listeners.get(id);
		expect(connection).toBeDefined();
		if (route.request().method() === 'DELETE') {
			listeners.delete(id);
			return route.fulfill({ headers, body: '' });
		}
		const event = { type: 'track', event: `Event from ${connection}`, receivedAt: '2026-01-01T00:00:00Z' };
		return route.fulfill({ json: { events: [event], omitted: connection === 'B' ? 1 : 0 } });
	});

	await page.goto('https://example.test/admin/connections/A/events');
	await page.evaluate(() => history.pushState(null, '', '/admin/connections/B/events'));
	await page.addScriptTag({ content: componentScript });
	return listeners;
};

test('Live events switches listeners when going back to another connection', async ({ page }) => {
	const listeners = await openLiveEvents(page);
	const previousEvent = page.getByText('Event from B', { exact: true });
	await previousEvent.first().click();
	await expect(page.locator('.connection-events__full-event')).toContainText('Event from B');
	await expect(page.locator('.connection-events__omitted')).toBeVisible();

	await page.goBack();
	await expect(page.getByText('Event from A', { exact: true }).first()).toBeVisible();
	expect([...listeners.values()]).toEqual(['A']);
	await expect(previousEvent).toHaveCount(0);
	await expect(page.locator('.connection-events__omitted')).toHaveCount(0);
	await expect(page.getByText('Click on one event')).toBeVisible();
});

test('Live events stays on the current connection when a previous connection finishes loading', async ({ page }) => {
	const listeners = await openLiveEvents(page);
	const eventsFromB = page.getByText('Event from B', { exact: true });
	await expect(eventsFromB.first()).toBeVisible();

	const heldA = Promise.withResolvers<Route>();
	await page.route('**/v1/connections/A/pipeline-types', (route) => heldA.resolve(route));
	await page.goBack();
	const requestA = await heldA.promise;
	const loadedB = page.waitForResponse('**/v1/connections/B/pipeline-types');
	await page.goForward();
	await loadedB;
	await expect(eventsFromB.first()).toBeVisible();

	const loadedA = page.waitForResponse('**/v1/connections/A/pipeline-types');
	await requestA.fulfill({ json: [] });
	await loadedA;
	await expect(page).toHaveURL(/\/B\/events$/);
	const count = await eventsFromB.count();
	await expect.poll(() => eventsFromB.count()).toBeGreaterThan(count);
	expect([...listeners.values()]).toEqual(['B']);
	await expect(page.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/admin/connections/B/settings');
});
