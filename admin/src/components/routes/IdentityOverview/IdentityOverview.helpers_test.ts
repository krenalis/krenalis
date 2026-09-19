/**
 * Focused tests for Identity Overview metric semantics.
 *
 * Run from admin/ with:
 *   ./node_modules/.bin/esbuild src/components/routes/IdentityOverview/IdentityOverview.helpers_test.ts \
 *     --bundle --platform=node --format=cjs --outfile=/tmp/identity-overview-helpers-test.cjs
 *   node /tmp/identity-overview-helpers-test.cjs
 */
import { IdentityMetric } from '../../../lib/api/types/metrics';
import {
	IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET,
	addUTCDays,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityMetricChartDays,
	calculateIdentityTrend,
	completeIdentityMetricDays,
	computeFetchRange,
	displayRangeForPreset,
} from './IdentityOverview.helpers';

const equal = (actual: unknown, expected: unknown, message: string) => {
	if (JSON.stringify(actual) !== JSON.stringify(expected)) {
		throw new Error(`${message}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
	}
};

const tests: { name: string; run: () => void }[] = [
	{
		name: 'date ranges include the trend and comparison windows',
		run: () => {
			equal(IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET, 'last30Days', 'default preset');
			equal(
				displayRangeForPreset('last7Days', '2026-08-10'),
				{ start: '2026-08-04', end: '2026-08-10' },
				'last 7 days',
			);
			equal(
				computeFetchRange({ start: '2026-08-01', end: '2026-08-10' }),
				{ start: '2026-06-12', end: '2026-08-11' },
				'fetch range',
			);
		},
	},
	{
		name: 'identity trends compare states as of exact day boundaries',
		run: () => {
			const days = Array.from({ length: 8 }, (_, index) => ({
				day: addUTCDays('2026-08-01', index),
				total: 100 + index * 10,
			}));
			const trend = calculateIdentityTrend(days, '2026-08-08', 7);
			equal(trend.currentValue, 170, 'current value');
			equal(trend.referenceValue, 100, 'reference value');
			equal(trend.changePercent, 70, 'change percent');
		},
	},
	{
		name: 'latest observations extend historical identity state',
		run: () => {
			const latest: IdentityMetric = {
				observedAt: '2026-08-03T12:00:00Z',
				total: 8,
				anonymous: 5,
				recognized: 3,
				withoutProfile: 1,
				connections: [],
			};
			const completed = completeIdentityMetricDays(
				[{ day: '2026-08-02', total: 4, anonymous: 3, recognized: 1 }],
				latest,
				{ start: '2026-08-02', end: '2026-08-05' },
			);
			equal(
				completed,
				[
					{ day: '2026-08-02', total: 4, anonymous: 3, recognized: 1 },
					{ day: '2026-08-03', total: 8, anonymous: 5, recognized: 3 },
					{ day: '2026-08-04', total: 8, anonymous: 5, recognized: 3 },
				],
				'completed days',
			);
		},
	},
	{
		name: 'missing chart days remain gaps',
		run: () => {
			equal(
				buildIdentityMetricChartDays([{ day: '2026-08-02', total: 4, anonymous: 3, recognized: 1 }], {
					start: '2026-08-01',
					end: '2026-08-03',
				}),
				[
					{ day: '2026-08-01', total: null, anonymous: null, recognized: null },
					{ day: '2026-08-02', total: 4, anonymous: 3, recognized: 1 },
					{ day: '2026-08-03', total: null, anonymous: null, recognized: null },
				],
				'chart days',
			);
		},
	},
	{
		name: 'deleted connections are derived from the workspace total',
		run: () => {
			const latest: IdentityMetric = {
				observedAt: '2026-08-03T12:00:00Z',
				total: 10,
				anonymous: 6,
				recognized: 4,
				withoutProfile: 2,
				connections: [{ connection: 'connection11', anonymous: 2, recognized: 3, withoutProfile: 1 }],
			};
			equal(
				buildDeletedConnectionMetric(latest),
				{ connection: 'deleted', anonymous: 4, recognized: 1, withoutProfile: 1 },
				'deleted connection',
			);
		},
	},
	{
		name: 'connection aggregation keeps largest entries and groups the rest',
		run: () => {
			const metrics = [
				{ connection: 'a', anonymous: 5, recognized: 5, withoutProfile: 0 },
				{ connection: 'b', anonymous: 2, recognized: 3, withoutProfile: 0 },
				{ connection: 'c', anonymous: 1, recognized: 1, withoutProfile: 0 },
			];
			equal(
				aggregateConnections(metrics, new Map(), 1),
				[
					{ connection: 'a', name: 'a', recognized: 5, anonymous: 5, total: 10 },
					{ connection: '__other__', name: 'Other', recognized: 4, anonymous: 3, total: 7 },
				],
				'aggregated connections',
			);
		},
	},
];

for (const test of tests) {
	try {
		test.run();
		console.log(`ok - ${test.name}`);
	} catch (error) {
		console.error(`not ok - ${test.name}`);
		throw error;
	}
}
