import assert from 'node:assert/strict';
import test from 'node:test';
import {
	parseInitialUserRecords,
	parseDuplicateRecordPercent,
} from '../src/components/routes/SimulatedAccounts/simulatedAccountValues.ts';

test('initial user records parse exact preset and grouped custom numbers within the allowed range', () => {
	for (const [text, expected] of [
		['0', 0],
		['10', 10],
		['1,000', 1000],
		['1000000', 1000000],
	]) {
		assert.equal(parseInitialUserRecords(text), expected);
	}
	for (const text of ['', '-1', '1.5', '1,00', '1000001', '10000000000000000000']) {
		assert.throws(() => parseInitialUserRecords(text));
	}
});

test('duplicate percentage preserves hundredths and rejects excess precision', () => {
	for (const [text, expected] of [
		['0', 0],
		['1', 1],
		['2', 2],
		['5', 5],
		['10', 10],
		['12.34', 12.34],
		['50.00', 50],
	]) {
		assert.equal(parseDuplicateRecordPercent(text), expected);
	}
	for (const text of ['', '-1', '0.001', '50.01', '1,5', '1.234']) {
		assert.throws(() => parseDuplicateRecordPercent(text));
	}
});
