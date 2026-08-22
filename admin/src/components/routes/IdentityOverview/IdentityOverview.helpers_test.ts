/**
 * Focused tests for Identity Overview metric semantics.
 *
 * Run from admin/ with:
 *   ./node_modules/.bin/esbuild src/components/routes/IdentityOverview/IdentityOverview.helpers_test.ts \
 *     --bundle --platform=node --format=cjs --outfile=/tmp/identity-overview-helpers-test.cjs
 *   node /tmp/identity-overview-helpers-test.cjs
 */
import {
	IdentityMetric,
	IdentityMetricDay,
	IdentityResolutionMetric,
	IdentityResolutionMetricDay,
} from '../../../lib/api/types/metrics';
import {
	IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET,
	addUTCDays,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityChangeSinceResolutionPoints,
	buildIdentityConnectionOptions,
	buildIdentityMetricChartDays,
	buildIdentityResolutionMetricChartDays,
	buildProfileComposition,
	buildResolutionEffectivenessData,
	buildTemporalSemantics,
	buildUnifiedProfileHistoryData,
	buildTypeDistribution,
	calculateIdentityChangeSinceResolution,
	calculateIdentityTrend,
	calculateResolutionKpiComparison,
	chartDomainTicks,
	completeIdentityMetricDays,
	completeIdentityResolutionMetricDays,
	computeFetchRange,
	displayRangeForPreset,
	formatComparisonDate,
	formatDate,
	formatNullableRatio,
	formatPercentagePointDelta,
	formatRatio,
	formatRatioDelta,
	formatRate,
	formatResolutionComparisonUTCTimestamp,
	formatResolutionLocalTimeZoneDetails,
	formatResolutionLocalTimestamp,
	formatResolutionUTCTimestamp,
	formatRunDuration,
	formatSharePercent,
	formatSignedIntegerDelta,
	formatTrendPercent,
	formatUTCTimestamp,
	hasResolutionDataInRange,
	identityLinkRateChartDomain,
	ratioChartDomain,
	sparklineDomain,
	sliceDays,
} from './IdentityOverview.helpers';

const equal = (actual: unknown, expected: unknown, message: string) => {
	if (JSON.stringify(actual) !== JSON.stringify(expected)) {
		throw new Error(`${message}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
	}
};

const closeTo = (actual: number | null, expected: number, message: string) => {
	if (actual == null || Math.abs(actual - expected) > 0.000001) {
		throw new Error(`${message}: expected ${expected}, got ${String(actual)}`);
	}
};

const identityDays = (start: string, count: number, value: (day: string, index: number) => number | null) => {
	return Array.from({ length: count }, (_, index) => {
		const day = addUTCDays(start, index);
		return { day, total: value(day, index) };
	});
};

const tests: { name: string; run: () => void }[] = [
	{
		name: 'inclusive UI range becomes an exclusive API range with a 60-day trend window',
		run: () => {
			equal(
				computeFetchRange({ start: '2026-08-01', end: '2026-08-10' }),
				{ start: '2026-06-12', end: '2026-08-11' },
				'fetch range',
			);
			equal(
				computeFetchRange({ start: '2026-05-01', end: '2026-08-10' }),
				{ start: '2026-01-19', end: '2026-08-11' },
				'long display range includes the immediately preceding period',
			);
			equal(
				computeFetchRange({ start: '2026-07-01', end: '2026-07-10' }, '2026-08-10'),
				{ start: '2026-06-12', end: '2026-08-11' },
				'historical Trend range also fetches the fixed latest 60-day trend window',
			);
		},
	},
	{
		name: 'date presets produce inclusive 7, 30, and 90 day display ranges',
		run: () => {
			equal(IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET, 'last30Days', 'default preset');
			equal(
				displayRangeForPreset('last7Days', '2026-08-10'),
				{ start: '2026-08-04', end: '2026-08-10' },
				'last 7 days',
			);
			equal(
				displayRangeForPreset('last30Days', '2026-08-10'),
				{ start: '2026-07-12', end: '2026-08-10' },
				'last 30 days',
			);
			equal(
				displayRangeForPreset('last90Days', '2026-08-10'),
				{ start: '2026-05-13', end: '2026-08-10' },
				'last 90 days',
			);
		},
	},
	{
		name: 'Trend range changes historical charts without changing latest semantics',
		run: () => {
			const latestObservedAt = '2026-08-10T09:15:00Z';
			const resolutionObservedAt = '2026-08-10T02:00:00Z';
			const rollingRange = displayRangeForPreset('last30Days', '2026-08-10');
			const customCurrentRange = { start: '2026-08-01', end: '2026-08-10' };
			const customHistoricalRange = { start: '2026-07-01', end: '2026-07-10' };
			const rolling = buildTemporalSemantics(rollingRange, latestObservedAt, resolutionObservedAt);
			const customCurrent = buildTemporalSemantics(customCurrentRange, latestObservedAt, resolutionObservedAt);
			const customHistorical = buildTemporalSemantics(
				customHistoricalRange,
				latestObservedAt,
				resolutionObservedAt,
			);

			equal(rolling.latestIdentityDay, '2026-08-10', 'rolling latest identity day');
			equal(customHistorical.latestIdentityDay, '2026-08-10', 'historical range keeps latest identity day');
			equal(rolling.showLatestResolutionComparison, true, 'rolling comparison visible');
			equal(customCurrent.showLatestResolutionComparison, true, 'current custom comparison visible');
			equal(customHistorical.showLatestResolutionComparison, false, 'historical custom comparison hidden');
			equal(formatDate(customHistorical.latestIdentityDay!), 'Aug 10, 2026', 'Current identity state label');
			equal(formatDate(customHistoricalRange.end), 'Jul 10, 2026', 'historical chart end remains distinct');

			const days = identityDays('2026-07-01', 41, (_day, index) => 1000 + index);
			equal(sliceDays(days, customHistoricalRange).length, 10, 'historical chart follows custom range');
			const rollingDays = sliceDays(days, rollingRange);
			equal(rollingDays[rollingDays.length - 1]?.day, '2026-08-10', 'rolling chart follows rolling range');
			equal(
				calculateIdentityTrend(days, rolling.latestIdentityDay!, 7),
				calculateIdentityTrend(days, customHistorical.latestIdentityDay!, 7),
				'fixed 7D trend remains independent of the Trend range',
			);
			equal(
				calculateIdentityTrend(days, rolling.latestIdentityDay!, 30),
				calculateIdentityTrend(days, customHistorical.latestIdentityDay!, 30),
				'fixed 30D trend remains independent of the Trend range',
			);
		},
	},
	{
		name: 'identity change since resolution subtracts processed from current and formats absolute deltas',
		run: () => {
			equal(calculateIdentityChangeSinceResolution(123676, 100000), 23676, 'positive change');
			equal(calculateIdentityChangeSinceResolution(90000, 100000), -10000, 'negative change');
			equal(calculateIdentityChangeSinceResolution(null, 100000), null, 'missing current state');
			equal(calculateIdentityChangeSinceResolution(100000, null), null, 'missing resolution state');
			equal(formatSignedIntegerDelta(23676), '+23,676', 'positive delta');
			equal(formatSignedIntegerDelta(-23676), '-23,676', 'negative delta');
			equal(formatSignedIntegerDelta(0), '+0', 'zero delta');
			equal(formatSignedIntegerDelta(null), '—', 'unavailable delta');
			equal(
				buildIdentityChangeSinceResolutionPoints(
					'2026-08-10T09:15:00Z',
					123676,
					'2026-08-10T02:00:00Z',
					100000,
				),
				[
					{ day: '2026-08-10T02:00:00Z', total: 100000 },
					{ day: '2026-08-10T09:15:00Z', total: 123676 },
				],
				'two-snapshot sparkline',
			);
			equal(
				buildIdentityChangeSinceResolutionPoints(null, null, '2026-08-10T02:00:00Z', 100000),
				[],
				'unavailable sparkline',
			);
		},
	},
	{
		name: 'resolution comparison timestamps omit the year only for a same-year current identity state',
		run: () => {
			const instant = '2026-08-10T02:00:00Z';
			equal(
				formatResolutionComparisonUTCTimestamp(instant, '2026-08-11'),
				'Aug 10 · 02:00 UTC',
				'same-year timestamp',
			);
			equal(
				formatResolutionComparisonUTCTimestamp(instant, '2027-01-01'),
				'Aug 10, 2026 · 02:00 UTC',
				'cross-year timestamp',
			);
		},
	},
	{
		name: 'comparison dates omit the year only when reference and current value share it',
		run: () => {
			equal(formatComparisonDate('2026-08-03', '2026-08-10'), 'Aug 3', 'same-year comparison');
			equal(formatComparisonDate('2025-12-31', '2026-01-07'), 'Dec 31, 2025', 'cross-year comparison');
		},
	},
	{
		name: '7-day change compares latest with the exact as-of reference and returns seven sparkline points',
		run: () => {
			const days = identityDays('2026-07-28', 14, (day) => {
				if (day === '2026-08-03') return 100;
				if (day === '2026-08-10') return 120;
				return 110;
			});
			const trend = calculateIdentityTrend(days, '2026-08-10', 7);
			closeTo(trend.changePercent, 20, '7D percentage');
			equal(trend.referenceDay, '2026-08-03', '7-day reference date');
			equal(trend.referenceValue, 100, '7-day reference value');
			equal(
				trend.points.map((point) => point.day),
				identityDays('2026-08-04', 7, () => 0).map((day) => day.day),
				'7D points',
			);
		},
	},
	{
		name: '30-day change compares latest with the exact as-of reference and returns 30 sparkline points',
		run: () => {
			const days = identityDays('2026-06-12', 60, (day) => {
				if (day === '2026-07-11') return 1000;
				if (day === '2026-08-10') return 1125;
				return 1050;
			});
			const trend = calculateIdentityTrend(days, '2026-08-10', 30);
			closeTo(trend.changePercent, 12.5, '30D percentage');
			equal(trend.referenceDay, '2026-07-11', '30-day reference date');
			equal(trend.referenceValue, 1000, '30-day reference value');
			equal(trend.points.length, 30, '30D point count');
		},
	},
	{
		name: 'identity change uses carry-forward for an as-of reference without an observation on that day',
		run: () => {
			const trend = calculateIdentityTrend(
				[
					{ day: '2026-08-01', total: 100 },
					{ day: '2026-08-03', total: null },
					{ day: '2026-08-10', total: 125 },
				],
				'2026-08-10',
				7,
			);
			equal(trend.referenceDay, '2026-08-03', 'requested as-of day');
			equal(trend.referenceValue, 100, 'carried-forward reference');
			closeTo(trend.changePercent, 25, 'change from carried-forward value');
		},
	},
	{
		name: 'trend division by null or zero is unavailable while a real current zero is preserved',
		run: () => {
			const nullPrevious = identityDays('2026-07-28', 14, (day) => (day === '2026-08-10' ? 8 : null));
			equal(calculateIdentityTrend(nullPrevious, '2026-08-10', 7).changePercent, null, 'null divisor');

			const zeroPrevious = identityDays('2026-07-28', 14, (day) => (day === '2026-08-10' ? 8 : 0));
			equal(calculateIdentityTrend(zeroPrevious, '2026-08-10', 7).changePercent, null, 'zero divisor');

			const currentZero = identityDays('2026-07-28', 14, (day) => {
				if (day === '2026-08-03') return 10;
				if (day === '2026-08-10') return 0;
				return 5;
			});
			equal(calculateIdentityTrend(currentZero, '2026-08-10', 7).changePercent, -100, 'real current zero');
		},
	},
	{
		name: 'sparkline domain follows the data instead of flattening trends against zero',
		run: () => {
			equal(
				sparklineDomain([
					{ day: '2026-08-09', total: 1000 },
					{ day: '2026-08-10', total: 1100 },
				]),
				[985, 1115],
				'variable series domain',
			);
			equal(
				sparklineDomain([
					{ day: '2026-08-09', total: 1000 },
					{ day: '2026-08-10', total: 1000 },
				]),
				[990, 1010],
				'constant series domain',
			);
			equal(sparklineDomain([{ day: '2026-08-10', total: null }]), [0, 1], 'missing series domain');
		},
	},
	{
		name: 'daily mapping filters inclusively, sorts ascending, and keeps null distinct from zero',
		run: () => {
			const selected = sliceDays(
				[
					{ day: '2026-08-03', total: 4 },
					{ day: '2026-08-01', total: null },
					{ day: '2026-08-02', total: 0 },
					{ day: '2026-07-31', total: 9 },
				],
				{ start: '2026-08-01', end: '2026-08-03' },
			);
			equal(
				selected,
				[
					{ day: '2026-08-01', total: null },
					{ day: '2026-08-02', total: 0 },
					{ day: '2026-08-03', total: 4 },
				],
				'daily series',
			);
		},
	},
	{
		name: 'identity chart restores API days omitted before and after known values',
		run: () => {
			equal(
				buildIdentityMetricChartDays(
					[
						{ day: '2026-08-02', total: 20, anonymous: 8, recognized: 12 },
						{ day: '2026-08-03', total: 30, anonymous: 10, recognized: 20 },
					],
					{ start: '2026-08-01', end: '2026-08-05' },
				),
				[
					{ day: '2026-08-01', total: null, anonymous: null, recognized: null },
					{ day: '2026-08-02', total: 20, anonymous: 8, recognized: 12 },
					{ day: '2026-08-03', total: 30, anonymous: 10, recognized: 20 },
					{ day: '2026-08-04', total: null, anonymous: null, recognized: null },
					{ day: '2026-08-05', total: null, anonymous: null, recognized: null },
				],
				'partially omitted API range',
			);
		},
	},
	{
		name: 'identity chart restores a completely omitted API range',
		run: () => {
			equal(
				buildIdentityMetricChartDays([], { start: '2026-08-01', end: '2026-08-03' }),
				[
					{ day: '2026-08-01', total: null, anonymous: null, recognized: null },
					{ day: '2026-08-02', total: null, anonymous: null, recognized: null },
					{ day: '2026-08-03', total: null, anonymous: null, recognized: null },
				],
				'empty API range',
			);
		},
	},
	{
		name: 'latest observation completes workspace, connection, and deleted daily histories',
		run: () => {
			const days: IdentityMetricDay[] = [
				{ day: '2026-08-08', total: 8, anonymous: 3, recognized: 5 },
				{ day: '2026-08-09', total: 9, anonymous: 4, recognized: 5 },
				{ day: '2026-08-11', total: 99, anonymous: 49, recognized: 50 },
			];
			const latest: IdentityMetric = {
				observedAt: '2026-08-10T09:00:00Z',
				total: 30,
				anonymous: 10,
				recognized: 20,
				withoutProfile: 3,
				connections: [{ connection: 'live', anonymous: 3, recognized: 4, withoutProfile: 1 }],
			};
			const range = { start: '2026-08-08', end: '2026-08-13' };
			equal(
				completeIdentityMetricDays(days, latest, range),
				[
					{ day: '2026-08-08', total: 8, anonymous: 3, recognized: 5 },
					{ day: '2026-08-09', total: 9, anonymous: 4, recognized: 5 },
					{ day: '2026-08-10', total: 30, anonymous: 10, recognized: 20 },
					{ day: '2026-08-11', total: 30, anonymous: 10, recognized: 20 },
					{ day: '2026-08-12', total: 30, anonymous: 10, recognized: 20 },
				],
				'workspace history',
			);
			equal(
				completeIdentityMetricDays(days, latest, { start: '2026-08-11', end: '2026-08-13' }),
				[
					{ day: '2026-08-11', total: 30, anonymous: 10, recognized: 20 },
					{ day: '2026-08-12', total: 30, anonymous: 10, recognized: 20 },
				],
				'latest before the requested range',
			);
			equal(
				completeIdentityMetricDays(days, latest, { start: '2026-08-08', end: '2026-08-10' }),
				days.slice(0, 2),
				'range before the latest observation',
			);
			equal(
				completeIdentityMetricDays([], latest, range, 'live').at(-1),
				{ day: '2026-08-12', total: 7, anonymous: 3, recognized: 4 },
				'connection history',
			);
			equal(
				completeIdentityMetricDays([], latest, range, 'missing').at(-1),
				{ day: '2026-08-12', total: 0, anonymous: 0, recognized: 0 },
				'missing connection history',
			);
			equal(
				completeIdentityMetricDays([], latest, range, 'deleted').at(-1),
				{ day: '2026-08-12', total: 23, anonymous: 7, recognized: 16 },
				'deleted connection history',
			);
		},
	},
	{
		name: 'latest Resolution completes and anchors its daily history',
		run: () => {
			const days: IdentityResolutionMetricDay[] = [
				{
					day: '2026-08-08',
					identities: 8,
					profiles: 5,
					profilesAnonymous: 2,
					profilesRecognized: 3,
					identitiesPerProfile: 1.6,
					linkedIdentitiesRate: 0.5,
				},
				{
					day: '2026-08-10',
					identities: 999,
					profiles: 999,
					profilesAnonymous: 999,
					profilesRecognized: 0,
					identitiesPerProfile: 1,
					linkedIdentitiesRate: 0,
				},
			];
			const latest: IdentityResolutionMetric = {
				observedAt: '2026-08-10T09:00:00Z',
				identities: { total: 30, anonymous: 10, recognized: 20 },
				profiles: { total: 12, anonymous: 4, recognized: 8 },
				composition: { one: 3, two: 2, three: 1, fourToTen: 1, elevenToTwenty: 0, moreThanTwenty: 0 },
				identitiesPerProfile: 2.5,
				linkedIdentitiesRate: 0.9,
			};
			equal(
				completeIdentityResolutionMetricDays(days, latest, { start: '2026-08-08', end: '2026-08-13' }),
				[
					days[0],
					{
						day: '2026-08-10',
						identities: 30,
						profiles: 12,
						profilesAnonymous: 4,
						profilesRecognized: 8,
						identitiesPerProfile: 2.5,
						linkedIdentitiesRate: 0.9,
					},
					{
						day: '2026-08-11',
						identities: 30,
						profiles: 12,
						profilesAnonymous: 4,
						profilesRecognized: 8,
						identitiesPerProfile: 2.5,
						linkedIdentitiesRate: 0.9,
					},
					{
						day: '2026-08-12',
						identities: 30,
						profiles: 12,
						profilesAnonymous: 4,
						profilesRecognized: 8,
						identitiesPerProfile: 2.5,
						linkedIdentitiesRate: 0.9,
					},
				],
				'anchored Resolution history',
			);
		},
	},
	{
		name: 'connections aggregate only two or more entries beyond the first seven into Other',
		run: () => {
			const connections = Array.from({ length: 9 }, (_, index) => ({
				connection: `c${index + 1}`,
				anonymous: 9 - index,
				recognized: 1,
				withoutProfile: 1000,
			}));
			connections.push({ connection: 'c1', anonymous: 5, recognized: 0, withoutProfile: 0 });
			const bars = aggregateConnections(connections, new Map([['c1', 'Salesforce']]));
			equal(bars.length, 8, 'top seven plus Other');
			equal(
				bars[0],
				{ connection: 'c1', name: 'Salesforce', recognized: 1, anonymous: 14, total: 15 },
				'aggregated first connection',
			);
			equal(
				bars[1],
				{ connection: 'c2', name: 'c2', recognized: 1, anonymous: 8, total: 9 },
				'connection ID fallback',
			);
			equal(
				bars[7],
				{ connection: '__other__', name: 'Other', recognized: 2, anonymous: 3, total: 5 },
				'Other sums both remaining connections',
			);
			equal(
				aggregateConnections(connections.slice(0, 8), new Map()).at(-1),
				{ connection: 'c8', name: 'c8', recognized: 1, anonymous: 2, total: 3 },
				'a single remaining connection retains its own name',
			);
		},
	},
	{
		name: 'all-zero connection breakdown produces the semantic zero/empty state',
		run: () => {
			const connections = [{ connection: 'c1', anonymous: 0, recognized: 0, withoutProfile: 99 }];
			equal(aggregateConnections(connections, new Map()), [], 'connection chart');
		},
	},
	{
		name: 'deleted connection aggregate is the latest parent/children residual',
		run: () => {
			equal(
				buildDeletedConnectionMetric({
					observedAt: '2026-08-10T09:00:00Z',
					total: 100,
					anonymous: 40,
					recognized: 60,
					withoutProfile: 10,
					connections: [
						{ connection: 'live-a', anonymous: 20, recognized: 35, withoutProfile: 3 },
						{ connection: 'live-b', anonymous: 10, recognized: 15, withoutProfile: 3 },
					],
				}),
				{ connection: 'deleted', anonymous: 10, recognized: 10, withoutProfile: 4 },
				'deleted aggregate',
			);
		},
	},
	{
		name: 'identity chart connection options include sources and persisted metric fallbacks alphabetically',
		run: () => {
			equal(
				buildIdentityConnectionOptions(
					[
						{ id: 'source-b', name: 'Zulu', role: 'Source' },
						{ id: 'destination', name: 'Destination', role: 'Destination' },
						{ id: 'source-a', name: 'Alpha', role: 'Source' },
					],
					[
						{ connection: 'source-a', anonymous: 1, recognized: 2, withoutProfile: 0 },
						{ connection: 'source-b', anonymous: 8, recognized: 12, withoutProfile: 0 },
						{ connection: 'historical', anonymous: 2, recognized: 3, withoutProfile: 0 },
						{ connection: 'deleted', anonymous: 3, recognized: 7, withoutProfile: 0 },
					],
				),
				[
					{ id: 'source-a', name: 'Alpha' },
					{ id: 'historical', name: 'historical' },
					{ id: 'source-b', name: 'Zulu' },
					{ id: 'deleted', name: 'Removed connections' },
				],
				'source and historical options sorted alphabetically with removed connections last',
			);
		},
	},
	{
		name: 'deleted connection option is omitted when its latest residual is zero',
		run: () => {
			equal(
				buildIdentityConnectionOptions(
					[{ id: 'source-a', name: 'Alpha', role: 'Source' }],
					[
						{ connection: 'source-a', anonymous: 4, recognized: 6, withoutProfile: 0 },
						{ connection: 'deleted', anonymous: 0, recognized: 0, withoutProfile: 0 },
					],
				),
				[{ id: 'source-a', name: 'Alpha' }],
				'zero latest deleted aggregate is hidden',
			);
		},
	},
	{
		name: 'connection shares support axis and tooltip precision and handle a zero denominator',
		run: () => {
			equal(formatSharePercent(1, 3, 0), '33%', 'axis share');
			equal(formatSharePercent(1, 3, 2), '33.33%', 'tooltip share');
			equal(formatSharePercent(0, 0), '—', 'zero total');
		},
	},
	{
		name: 'resolution effectiveness maps direct fields in the Trend range without normalizing profiles',
		run: () => {
			const days: IdentityResolutionMetricDay[] = [
				{
					day: '2026-07-31',
					identities: 10,
					profiles: 1000,
					profilesAnonymous: 400,
					profilesRecognized: 600,
					identitiesPerProfile: 1,
					linkedIdentitiesRate: 0.1,
				},
				{
					day: '2026-08-01',
					identities: 100,
					profiles: 50,
					profilesAnonymous: 20,
					profilesRecognized: 30,
					identitiesPerProfile: '2' as unknown as number,
					linkedIdentitiesRate: 0.25,
				},
				{
					day: '2026-08-02',
					identities: 200,
					profiles: 200,
					profilesAnonymous: 80,
					profilesRecognized: 120,
					identitiesPerProfile: 1,
					linkedIdentitiesRate: 0,
				},
			];
			const range = { start: '2026-08-01', end: '2026-08-03' };
			const chartDays = buildIdentityResolutionMetricChartDays(days, range);
			const chart = buildResolutionEffectivenessData(chartDays);
			const profiles = buildUnifiedProfileHistoryData(chartDays);
			equal(
				chart[0],
				{ day: '2026-08-01', linkedIdentitiesRatePercent: 25, identitiesPerProfile: 2 },
				'direct first point with runtime numeric coercion',
			);
			equal(
				chart[1],
				{ day: '2026-08-02', linkedIdentitiesRatePercent: 0, identitiesPerProfile: 1 },
				'observed zero identity link rate',
			);
			equal(
				profiles,
				[
					{ day: '2026-08-01', recognized: 30, anonymous: 20 },
					{ day: '2026-08-02', recognized: 120, anonymous: 80 },
					{ day: '2026-08-03', recognized: null, anonymous: null },
				],
				'unified profile category history preserves range and null semantics',
			);
			equal(
				chart[2],
				{ day: '2026-08-03', linkedIdentitiesRatePercent: null, identitiesPerProfile: null },
				'null gap',
			);
		},
	},
	{
		name: 'resolution KPI comparisons share one as-of reference and preserve each metric unit',
		run: () => {
			const comparison = calculateResolutionKpiComparison(
				{
					observedAt: '2026-08-10T02:00:00Z',
					identities: { total: 200, anonymous: 80, recognized: 120 },
					profiles: { total: 120, anonymous: 40, recognized: 80 },
					composition: {
						one: 40,
						two: 30,
						three: 20,
						fourToTen: 20,
						elevenToTwenty: 8,
						moreThanTwenty: 2,
					},
					linkedIdentitiesRate: 0.404,
					identitiesPerProfile: 1.28,
				},
				[
					{
						day: '2026-08-03',
						identities: 160,
						profiles: 100,
						profilesAnonymous: 40,
						profilesRecognized: 60,
						linkedIdentitiesRate: 0.378,
						identitiesPerProfile: 1.25,
					},
				],
				{ start: '2026-08-04', end: '2026-08-10' },
			);

			equal(comparison.previousEnd, '2026-08-03', 'shared comparison reference date');
			closeTo(comparison.profilesChangePercent, 20, 'profiles percentage change');
			closeTo(comparison.linkedIdentitiesRatePercentagePoints, 2.6, 'link-rate percentage points');
			closeTo(comparison.identitiesPerProfileChange, 0.03, 'ratio absolute change');
			equal(formatPercentagePointDelta(comparison.linkedIdentitiesRatePercentagePoints), '+2.6 pp', 'pp format');
			equal(formatRatioDelta(comparison.identitiesPerProfileChange), '+0.03', 'ratio delta format');
		},
	},
	{
		name: 'resolution KPI comparisons treat zero ratios as values and a zero profile reference as unavailable',
		run: () => {
			const latest = {
				observedAt: '2026-08-10T02:00:00Z',
				identities: { total: 0, anonymous: 0, recognized: 0 },
				profiles: { total: 12, anonymous: 4, recognized: 8 },
				composition: {
					one: 0,
					two: 0,
					three: 0,
					fourToTen: 0,
					elevenToTwenty: 0,
					moreThanTwenty: 0,
				},
				linkedIdentitiesRate: 0.2,
				identitiesPerProfile: 0,
			};
			const comparison = calculateResolutionKpiComparison(
				latest,
				[
					{
						day: '2026-08-03',
						identities: 0,
						profiles: 0,
						profilesAnonymous: 0,
						profilesRecognized: 0,
						linkedIdentitiesRate: 0,
						identitiesPerProfile: 0,
					},
				],
				{ start: '2026-08-04', end: '2026-08-10' },
			);

			equal(comparison.profilesChangePercent, null, 'zero profile denominator');
			closeTo(comparison.linkedIdentitiesRatePercentagePoints, 20, 'zero link-rate reference');
			closeTo(comparison.identitiesPerProfileChange, 0, 'zero ratio values');
			equal(formatPercentagePointDelta(null), '—', 'unavailable pp');
			equal(formatRatioDelta(null), '—', 'unavailable ratio delta');
		},
	},
	{
		name: 'resolution KPI comparison formatters preserve available zero deltas',
		run: () => {
			equal(formatTrendPercent(0), '0.0%', 'zero profiles percentage');
			equal(formatPercentagePointDelta(0), '0.0 pp', 'zero link-rate percentage points');
			equal(formatRatioDelta(0), '0.00', 'zero identities-per-profile delta');
			equal(formatTrendPercent(null), '—', 'unavailable profiles percentage');
			equal(formatPercentagePointDelta(null), '—', 'unavailable percentage points');
			equal(formatRatioDelta(null), '—', 'unavailable identities-per-profile delta');
		},
	},
	{
		name: 'profile composition preserves the approved exact buckets and calculates percentages',
		run: () => {
			const buckets = buildProfileComposition(
				{
					one: 10,
					two: 3,
					three: 1,
					fourToTen: 3,
					elevenToTwenty: 2,
					moreThanTwenty: 1,
				},
				20,
			);
			equal(
				buckets.map(({ label, count, percentage }) => ({ label, count, percentage })),
				[
					{ label: '1 identity', count: 10, percentage: 50 },
					{ label: '2 identities', count: 3, percentage: 15 },
					{ label: '3 identities', count: 1, percentage: 5 },
					{ label: '4–10 identities', count: 3, percentage: 15 },
					{ label: '11–20 identities', count: 2, percentage: 10 },
					{ label: '21+ identities', count: 1, percentage: 5 },
				],
				'composition buckets',
			);
			equal(
				buildProfileComposition(
					{
						one: 0,
						two: 0,
						three: 0,
						fourToTen: 0,
						elevenToTwenty: 0,
						moreThanTwenty: 0,
					},
					0,
				).map((bucket) => bucket.percentage),
				[null, null, null, null, null, null],
				'zero-profile percentages',
			);
		},
	},
	{
		name: 'type distributions use the supplied Resolution total and preserve null versus zero semantics',
		run: () => {
			equal(
				buildTypeDistribution(70, 30, 100),
				[
					{ key: 'recognized', label: 'Recognized', count: 70, percentage: 70 },
					{ key: 'anonymous', label: 'Anonymous', count: 30, percentage: 30 },
				],
				'type distribution',
			);
			equal(
				buildTypeDistribution(0, 0, 0).map((segment) => segment.percentage),
				[null, null],
				'empty distribution percentages',
			);
		},
	},
	{
		name: 'resolution effectiveness domains add padding around visible non-null values without forcing zero',
		run: () => {
			const points = [
				{ day: '2026-08-01', linkedIdentitiesRatePercent: 40, identitiesPerProfile: 1.2 },
				{ day: '2026-08-02', linkedIdentitiesRatePercent: null, identitiesPerProfile: null },
				{ day: '2026-08-03', linkedIdentitiesRatePercent: 44, identitiesPerProfile: 1.4 },
			];
			equal(identityLinkRateChartDomain(points), [39, 45], 'padded identity link rate domain');
			equal(ratioChartDomain(points), [1.15, 1.45], 'padded ratio domain');
			equal(chartDomainTicks([39, 45]), [39, 42, 45], 'three readable ticks');
			equal(identityLinkRateChartDomain([]), [0, 100], 'empty rate fallback');
			equal(ratioChartDomain([]), [0, 1], 'empty ratio fallback');
			equal(formatRatio(1.236), '1.24', 'ratio maximum two decimals');
			equal(formatRatio(1.2), '1.2', 'ratio avoids trailing decimals');
			equal(formatNullableRatio(null), '—', 'null ratio is unavailable');
			equal(
				formatNullableRatio('1.2761069502114943' as unknown as number),
				'1.28',
				'KPI ratio handles runtime high-precision numeric values',
			);
			equal(formatRate(null), '—', 'null latest identity link rate');
		},
	},
	{
		name: 'Trend-range resolution state treats zero as data and an omitted range as empty',
		run: () => {
			const zero: IdentityResolutionMetricDay = {
				day: '2026-08-01',
				identities: 0,
				profiles: 0,
				profilesAnonymous: 0,
				profilesRecognized: 0,
				identitiesPerProfile: 0,
				linkedIdentitiesRate: 0,
			};
			equal(hasResolutionDataInRange([zero], { start: zero.day, end: zero.day }), true, 'real zero is data');
			equal(hasResolutionDataInRange([], { start: zero.day, end: zero.day }), false, 'omitted range is empty');
		},
	},
	{
		name: 'UTC timestamps use the approved dashboard format',
		run: () => {
			equal(formatUTCTimestamp('2026-08-10T02:15:00Z'), 'Aug 10, 2026 02:15 AM', 'UTC timestamp');
		},
	},
	{
		name: 'last successful resolution shows canonical UTC and the complete browser-local date and time',
		run: () => {
			const instant = '2026-08-10T02:00:00Z';
			equal(formatResolutionUTCTimestamp(instant), 'Aug 10, 2026 · 02:00 UTC', 'canonical UTC timestamp');
			equal(
				formatResolutionLocalTimestamp(instant, 'Europe/Rome', 'it-IT'),
				'Aug 10, 2026 · 04:00 CEST',
				'local timestamp',
			);
			equal(
				formatResolutionLocalTimestamp(instant, 'America/Los_Angeles', 'en-US'),
				'Aug 9, 2026 · 19:00 PDT',
				'local timestamp preserves a different local calendar date',
			);
			equal(
				formatResolutionLocalTimeZoneDetails(instant, 'Europe/Rome'),
				'Europe/Rome · UTC+02:00',
				'positive local offset details',
			);
			equal(
				formatResolutionLocalTimeZoneDetails(instant, 'America/Los_Angeles'),
				'America/Los_Angeles · UTC-07:00',
				'negative local offset details',
			);
		},
	},
	{
		name: 'run history durations are concise and preserve unavailable values',
		run: () => {
			const start = '2026-08-10T01:00:00Z';
			equal(formatRunDuration(start, null), '—', 'running duration');
			equal(formatRunDuration(start, '2026-08-10T01:00:42Z'), '42s', 'seconds');
			equal(formatRunDuration(start, '2026-08-10T01:04:32Z'), '4m 32s', 'minutes');
			equal(formatRunDuration(start, '2026-08-10T03:05:00Z'), '2h 5m', 'hours');
			equal(formatRunDuration(start, '2026-08-11T03:00:00Z'), '1d 2h', 'days');
			equal(formatRunDuration(start, '2026-08-09T03:00:00Z'), '—', 'invalid interval');
		},
	},
];

let failures = 0;
for (const test of tests) {
	try {
		test.run();
		console.log(`PASS ${test.name}`);
	} catch (error) {
		failures++;
		console.error(`FAIL ${test.name}`);
		console.error(error);
	}
}

if (failures > 0) {
	process.exitCode = 1;
} else {
	console.log(`\n${tests.length} Identity Overview helper tests passed.`);
}
