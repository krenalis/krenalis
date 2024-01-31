import { assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Options from './options.js';

Deno.test('Options', () => {
	localStorage.clear();

	const thirtyMinutes = 30 * 60000;

	const tests = [
		{ options: undefined, autoTrack: true, timeout: thirtyMinutes },
		{ options: null, autoTrack: true, timeout: thirtyMinutes },
		{ options: {}, autoTrack: true, timeout: thirtyMinutes },
		{ options: [], autoTrack: true, timeout: thirtyMinutes },
		{ options: '', autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: {} }, autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: true } }, autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: false } }, autoTrack: false, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: null } }, autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: 10 * 1000 } }, autoTrack: true, timeout: 10 * 1000 },
		{ options: { sessions: { timeout: '5000' } }, autoTrack: true, timeout: 5 * 1000 },
		{ options: { sessions: { timeout: Infinity } }, autoTrack: true, timeout: Infinity },
		{ options: { sessions: { timeout: {} } }, autoTrack: true, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: 0 } }, autoTrack: false, timeout: thirtyMinutes },
		{ options: { sessions: { timeout: -5000 } }, autoTrack: false, timeout: thirtyMinutes },
		{ options: { sessions: { autoTrack: true, timeout: 20 * 1000 } }, autoTrack: true, timeout: 20 * 1000 },
		{ options: { sessions: { autoTrack: true, timeout: 0 } }, autoTrack: false, timeout: thirtyMinutes },
	];

	for (let i = 0; i < tests.length; i++) {
		const test = tests[i];
		const options = new Options(test.options);
		assertEquals(options.sessions.autoTrack, test.autoTrack);
		assertEquals(options.sessions.timeout, test.timeout);
	}
});
