import { expect, test } from '@playwright/test';
import { build } from 'esbuild';
import { resolve } from 'path';

let componentScript: string;

test.beforeAll(async () => {
	const result = await build({
		entryPoints: [resolve(__dirname, 'segmented-date-range-control.fixture.tsx')],
		bundle: true,
		write: false,
		platform: 'browser',
		format: 'iife',
		loader: { '.css': 'empty' },
		define: { 'process.env.NODE_ENV': '"test"' },
	});
	componentScript = result.outputFiles[0].text;
});

test.beforeEach(async ({ page }) => {
	await page.setContent('<div id="root"></div>');
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByTestId('limited')).toBeAttached();
});

test('limits calendar and editable input selections without losing focus', async ({ page }) => {
	const control = page.getByTestId('limited');
	const output = page.getByTestId('limited-range');
	const inputs = control.locator('.rdrDateInput input');

	await inputs.first().fill('Aug 21, 2026');
	await inputs.first().press('Enter');
	await expect(output).toHaveText('2026-08-21:2026-08-21');
	await expect(inputs.first()).toBeFocused();

	await page.setContent('<div id="root"></div>');
	await page.addScriptTag({ content: componentScript });
	await expect(page.getByTestId('limited')).toBeAttached();

	const resetControl = page.getByTestId('limited');
	const resetOutput = page.getByTestId('limited-range');
	const resetInputs = resetControl.locator('.rdrDateInput input');
	await resetInputs.nth(1).fill('Aug 26, 2026');
	await resetInputs.nth(1).press('Enter');
	await expect(resetOutput).toHaveText('2026-08-20:2026-08-25');
	await expect(resetInputs.nth(1)).toHaveValue('Aug 25, 2026');
	await expect(resetInputs.nth(1)).toBeFocused();

	const futureDay = resetControl.locator('.rdrMonth').first().locator('.rdrDayDisabled:not(.rdrDayPassive)').first();
	await expect(futureDay).toContainText('26');
	await expect(futureDay).toHaveAttribute('tabindex', '-1');
	await futureDay.focus();
	await futureDay.press('Enter');
	await expect(resetOutput).toHaveText('2026-08-20:2026-08-25');
	await expect(futureDay).toBeFocused();
});

test('preserves the existing behavior when no maximum date is provided', async ({ page }) => {
	const control = page.getByTestId('unlimited');
	const output = page.getByTestId('unlimited-range');
	const endInput = control.locator('.rdrDateInput input').nth(1);

	await endInput.fill('Aug 26, 2026');
	await endInput.press('Enter');
	await expect(output).toHaveText('2026-08-20:2026-08-26');
	await expect(endInput).toBeFocused();
});
