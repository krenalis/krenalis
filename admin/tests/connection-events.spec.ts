import { expect, test } from '@playwright/test';
import { randomUUID } from 'crypto';
import { build } from 'esbuild';
import { resolve } from 'path';

test('Live events switches listeners when going back to another connection', async ({ page }) => {
	const bundle = await build({
		entryPoints: [resolve(__dirname, 'connection-events.fixture.tsx')],
		bundle: true,
		write: false,
		loader: { '.css': 'empty' },
		define: { 'process.env.NODE_ENV': '"test"' },
	});
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
	await page.addScriptTag({ content: bundle.outputFiles[0].text });
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
