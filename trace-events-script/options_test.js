import { assertEquals, AssertionError } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Options from './options.js';

Deno.test('Options', () => {
	localStorage.clear();

	const thirtyMinutes = 30 * 60000;

	const tests = [
		{ options: undefined, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: null, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: {}, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: [], strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: '', strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: {} }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: true } }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: false } }, strategy: 'AB-C', autoTrack: false, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: null } }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: 10 * 1000 } }, strategy: 'AB-C', autoTrack: true, timeout: 10 * 1000 },
		{ options: { sessions: { timeout: '5000' } }, strategy: 'AB-C', autoTrack: true, timeout: 5 * 1000 },
		{ options: { sessions: { timeout: Infinity } }, strategy: 'AB-C', autoTrack: true, timeout: Infinity },
		{ options: { sessions: { timeout: {} } }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: 0 } }, strategy: 'AB-C', autoTrack: false, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: -5000 } }, strategy: 'AB-C', autoTrack: false, timeout: thirtyMinutes },
		{
			options: { sessions: { autoTrack: true, timeout: 20 * 1000 } },
			strategy: 'AB-C',
			autoTrack: true,
			timeout: 20 * 1000,
		},
		{
			options: { sessions: { autoTrack: true, timeout: 0 } },
			strategy: 'AB-C',
			autoTrack: false,
			timeout: thirtyMinutes,
		},
		{ options: { strategy: undefined }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { strategy: null }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { strategy: 'ABC' }, strategy: 'ABC', autoTrack: true, timeout: thirtyMinutes },
		{ options: { strategy: 'AB-C' }, strategy: 'AB-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { strategy: 'A-B-C' }, strategy: 'A-B-C', autoTrack: true, timeout: thirtyMinutes },
		{ options: { strategy: 'AC-B' }, strategy: 'AC-B', autoTrack: true, timeout: thirtyMinutes },
		{
			options: { strategy: 'A-B-C', sessions: { autoTrack: true, timeout: 20 * 1000 } },
			strategy: 'A-B-C',
			autoTrack: true,
			timeout: 20 * 1000,
		},
	];

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i];
		const options = new Options(test.options);
		assertEquals(options.sessions.autoTrack, test.autoTrack);
		assertEquals(options.sessions.timeout, test.timeout);
		assertEquals(options.strategy, test.strategy);
	}

	// Test invalid strategies.
	const invalides = ['', 5, {}, 'CBA', 'A--BC', ' ABC'];
	for (let i = 0; i < invalides.length; i++) {
		const strategy = invalides[i];
		try {
			new Options({ strategy });
		} catch {
			continue;
		}
		throw new AssertionError(`'${strategy}' is not a strategy, but no error has been returned`);
	}
});
