import { assertEquals, AssertionError } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Options from './options.js';

Deno.test('Options', () => {
	localStorage.clear();

	const base = { debug: false, strategy: 'AB-C', autoTrack: true, timeout: 30 * 60000 };

	const tests = [
		{ options: undefined, ...base },
		{ options: null, ...base },
		{ options: {}, ...base },
		{ options: [], ...base },
		{ options: '', ...base },
		{ options: { sessions: {} }, ...base },
		{ options: { sessions: { autoTrack: true } }, ...base },
		{ options: { sessions: { autoTrack: false } }, ...base, autoTrack: false },
		{ options: { sessions: { autoTrack: null } }, ...base },
		{ options: { sessions: { timeout: 10 * 1000 } }, ...base, timeout: 10 * 1000 },
		{ options: { sessions: { timeout: '5000' } }, ...base, timeout: 5 * 1000 },
		{ options: { sessions: { timeout: Infinity } }, ...base, timeout: Infinity },
		{ options: { sessions: { timeout: {} } }, ...base },
		{ options: { sessions: { timeout: 0 } }, ...base, autoTrack: false },
		{ options: { sessions: { timeout: -5000 } }, ...base, autoTrack: false },
		{ options: { sessions: { autoTrack: true, timeout: 20 * 1000 } }, ...base, timeout: 20 * 1000 },
		{ options: { sessions: { autoTrack: true, timeout: 0 } }, ...base, autoTrack: false },
		{ options: { strategy: undefined }, ...base },
		{ options: { strategy: null }, ...base },
		{ options: { strategy: 'ABC' }, ...base, strategy: 'ABC' },
		{ options: { strategy: 'AB-C' }, ...base },
		{ options: { strategy: 'A-B-C' }, ...base, strategy: 'A-B-C' },
		{ options: { strategy: 'AC-B' }, ...base, strategy: 'AC-B' },
		{
			options: { strategy: 'A-B-C', sessions: { autoTrack: true, timeout: 20 * 1000 } },
			...base,
			strategy: 'A-B-C',
			timeout: 20 * 1000,
		},
		{ options: { debug: false }, ...base },
		{ options: { debug: true }, ...base, debug: true },
		{ options: { debug: 0 }, ...base },
		{ options: { debug: 1 }, ...base, debug: true },
	];

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i];
		const options = new Options(test.options);
		assertEquals(options.debug, test.debug);
		assertEquals(options.sessions.autoTrack, test.autoTrack);
		assertEquals(options.sessions.timeout, test.timeout);
		assertEquals(options.strategy, test.strategy);
	}

	// Test invalid strategies.
	const invalids = ['', 5, {}, 'CBA', 'A--BC', ' ABC'];
	for (let i = 0; i < invalids.length; i++) {
		const strategy = invalids[i];
		try {
			new Options({ strategy });
		} catch {
			continue;
		}
		throw new AssertionError(`'${strategy}' is not a strategy, but no error has been returned`);
	}
});
