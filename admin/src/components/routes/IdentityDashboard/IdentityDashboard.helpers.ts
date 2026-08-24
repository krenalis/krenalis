import {
	IdentityConnectionMetric,
	IdentityMetric,
	IdentityMetricDay,
	IdentityResolutionComposition,
	IdentityResolutionMetric,
	IdentityResolutionMetricDay,
} from '../../../lib/api/types/metrics';

const DELETED_CONNECTION_SCOPE = 'deleted';
const DELETED_CONNECTION_LABEL = 'Removed connections';

interface DisplayDateRange {
	start: string;
	end: string;
}

interface IdentityDashboardTemporalSemantics {
	latestIdentityDay: string | null;
	showLatestResolutionComparison: boolean;
}

interface IdentityMetricChartDay {
	day: string;
	total: number | null;
	anonymous: number | null;
	recognized: number | null;
}

interface IdentityResolutionMetricChartDay {
	day: string;
	identities: number | null;
	profiles: number | null;
	profilesAnonymous: number | null;
	profilesRecognized: number | null;
	identitiesPerProfile: number | null;
	linkedIdentitiesRate: number | null;
}

type IdentityDashboardDatePreset = 'last7Days' | 'last30Days' | 'last90Days';

const IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET: IdentityDashboardDatePreset = 'last30Days';

interface TrendPoint {
	day: string;
	total: number | null;
}

interface IdentityTrend {
	changePercent: number | null;
	currentValue: number | null;
	referenceValue: number | null;
	currentDay: string;
	referenceDay: string;
	points: TrendPoint[];
}

interface ConnectionBar {
	connection: string;
	name: string;
	recognized: number;
	anonymous: number;
	total: number;
}

interface IdentityConnectionCatalogEntry {
	id: string;
	name: string;
	role: 'Source' | 'Destination';
}

interface IdentityConnectionOption {
	id: string;
	name: string;
}

interface ResolutionEffectivenessPoint {
	day: string;
	linkedIdentitiesRatePercent: number | null;
	identitiesPerProfile: number | null;
}

interface UnifiedProfileHistoryPoint {
	day: string;
	recognized: number | null;
	anonymous: number | null;
}

interface ResolutionKpiComparison {
	previousEnd: string;
	profilesChangePercent: number | null;
	linkedIdentitiesRatePercentagePoints: number | null;
	identitiesPerProfileChange: number | null;
}

interface ProfileCompositionBucket {
	key: keyof IdentityResolutionComposition;
	label: string;
	count: number;
	percentage: number | null;
}

interface TypeDistributionSegment {
	key: 'recognized' | 'anonymous';
	label: 'Recognized' | 'Anonymous';
	count: number;
	percentage: number | null;
}

type ChartDomain = [number, number];

const DAY_MS = 24 * 60 * 60 * 1000;

const dateKeyToUTCDate = (dateKey: string): Date => {
	const [year, month, day] = dateKey.split('-').map(Number);
	return new Date(Date.UTC(year, month - 1, day));
};

const dateKeyToPickerDate = (dateKey: string): Date => {
	const [year, month, day] = dateKey.split('-').map(Number);
	return new Date(year, month - 1, day);
};

const pickerDateToDateKey = (date: Date): string => {
	return [date.getFullYear(), date.getMonth() + 1, date.getDate()]
		.map((part, index) => (index === 0 ? String(part) : String(part).padStart(2, '0')))
		.join('-');
};

const instantToDateKey = (instant: string): string => {
	return new Date(instant).toISOString().slice(0, 10);
};

const todayUTCDateKey = (now = new Date()): string => now.toISOString().slice(0, 10);

const addUTCDays = (dateKey: string, days: number): string => {
	const date = dateKeyToUTCDate(dateKey);
	date.setUTCDate(date.getUTCDate() + days);
	return date.toISOString().slice(0, 10);
};

const displayRangeForPreset = (preset: IdentityDashboardDatePreset, end: string): DisplayDateRange => {
	const days = preset === 'last7Days' ? 7 : preset === 'last30Days' ? 30 : 90;
	return { start: addUTCDays(end, -(days - 1)), end };
};

const daysBetween = (start: string, end: string): number => {
	return Math.round((dateKeyToUTCDate(end).getTime() - dateKeyToUTCDate(start).getTime()) / DAY_MS);
};

const computeFetchRange = (displayRange: DisplayDateRange, fixedTrendEnd = displayRange.end): DisplayDateRange => {
	const trendStart = addUTCDays(fixedTrendEnd, -59);
	const comparisonLength = daysBetween(displayRange.start, displayRange.end) + 1;
	const comparisonStart = addUTCDays(displayRange.start, -comparisonLength);
	return {
		start: comparisonStart < trendStart ? comparisonStart : trendStart,
		end: addUTCDays(displayRange.end > fixedTrendEnd ? displayRange.end : fixedTrendEnd, 1),
	};
};

const buildTemporalSemantics = (
	displayRange: DisplayDateRange,
	identityObservedAt?: string,
	resolutionObservedAt?: string,
): IdentityDashboardTemporalSemantics => {
	const latestIdentityDay = identityObservedAt == null ? null : instantToDateKey(identityObservedAt);
	const latestAvailableDay =
		latestIdentityDay ?? (resolutionObservedAt == null ? null : instantToDateKey(resolutionObservedAt));
	return {
		latestIdentityDay,
		showLatestResolutionComparison: latestAvailableDay != null && displayRange.end === latestAvailableDay,
	};
};

const sliceDays = <T extends { day: string }>(days: T[], range: DisplayDateRange): T[] => {
	return days
		.filter((day) => day.day >= range.start && day.day <= range.end)
		.sort((left, right) => left.day.localeCompare(right.day));
};

const valueAsOf = (days: TrendPoint[], targetDay: string): number | null => {
	let selectedDay = '';
	let selectedValue: number | null = null;
	for (const point of days) {
		if (point.day <= targetDay && point.day >= selectedDay && point.total != null) {
			selectedDay = point.day;
			selectedValue = point.total;
		}
	}
	return selectedValue;
};

const calculateIdentityTrend = (days: TrendPoint[], end: string, length: number): IdentityTrend => {
	const currentStart = addUTCDays(end, -(length - 1));
	const referenceDay = addUTCDays(end, -length);
	const currentValue = valueAsOf(days, end);
	const referenceValue = valueAsOf(days, referenceDay);
	const changePercent =
		currentValue == null || referenceValue == null || referenceValue === 0
			? null
			: ((currentValue - referenceValue) / referenceValue) * 100;

	return {
		changePercent,
		currentValue,
		referenceValue,
		currentDay: end,
		referenceDay,
		points: sliceDays(days, { start: currentStart, end }),
	};
};

const calculateIdentityChangeSinceResolution = (
	currentIdentities: number | null,
	processedIdentities: number | null,
): number | null =>
	currentIdentities == null || processedIdentities == null ? null : currentIdentities - processedIdentities;

const buildIdentityChangeSinceResolutionPoints = (
	currentObservedAt: string | null,
	currentIdentities: number | null,
	resolutionObservedAt: string | null,
	processedIdentities: number | null,
): TrendPoint[] =>
	currentObservedAt == null ||
	currentIdentities == null ||
	resolutionObservedAt == null ||
	processedIdentities == null
		? []
		: [
				{ day: resolutionObservedAt, total: processedIdentities },
				{ day: currentObservedAt, total: currentIdentities },
			];

const sparklineDomain = (points: TrendPoint[]): ChartDomain => {
	const values = points.flatMap((point) => (point.total == null ? [] : [point.total]));
	if (values.length === 0) return [0, 1];

	const minimum = Math.min(...values);
	const maximum = Math.max(...values);
	const spread = maximum - minimum;
	const padding = spread === 0 ? Math.max(Math.abs(maximum) * 0.01, 1) : spread * 0.15;
	return [Math.max(0, minimum - padding), maximum + padding];
};

// buildIdentityMetricChartDays restores every day in the displayed range so
// omitted API days preserve the existing no-data visualization.
const buildIdentityMetricChartDays = (days: IdentityMetricDay[], range: DisplayDateRange): IdentityMetricChartDay[] => {
	const metricsByDay = new Map(days.map((day) => [day.day, day]));
	const result: IdentityMetricChartDay[] = [];
	for (let day = range.start; day <= range.end; day = addUTCDays(day, 1)) {
		result.push(
			metricsByDay.get(day) ?? {
				day,
				total: null,
				anonymous: null,
				recognized: null,
			},
		);
	}
	return result;
};

// buildDeletedConnectionMetric derives the synthetic latest category from the
// workspace parent total and the live connection children in the same response.
const buildDeletedConnectionMetric = (latest: IdentityMetric | null): IdentityConnectionMetric | null => {
	if (latest == null) return null;
	const childCounts = (latest.connections ?? []).reduce(
		(counts, connection) => ({
			anonymous: counts.anonymous + Number(connection.anonymous),
			recognized: counts.recognized + Number(connection.recognized),
			withoutProfile: counts.withoutProfile + Number(connection.withoutProfile),
		}),
		{ anonymous: 0, recognized: 0, withoutProfile: 0 },
	);
	return {
		connection: DELETED_CONNECTION_SCOPE,
		anonymous: Number(latest.anonymous) - childCounts.anonymous,
		recognized: Number(latest.recognized) - childCounts.recognized,
		withoutProfile: Number(latest.withoutProfile) - childCounts.withoutProfile,
	};
};

// completeIdentityMetricDays extends historical days with the state anchored
// by the latest observation. The range end is exclusive.
const completeIdentityMetricDays = (
	days: IdentityMetricDay[],
	latest: IdentityMetric,
	range: DisplayDateRange,
	connectionSelection?: string,
): IdentityMetricDay[] => {
	let anonymous: number;
	let recognized: number;
	if (connectionSelection == null) {
		anonymous = Number(latest.anonymous);
		recognized = Number(latest.recognized);
	} else if (connectionSelection === DELETED_CONNECTION_SCOPE) {
		const deleted = buildDeletedConnectionMetric(latest)!;
		anonymous = deleted.anonymous;
		recognized = deleted.recognized;
	} else {
		const connection = latest.connections.find((metric) => metric.connection === connectionSelection);
		anonymous = connection == null ? 0 : Number(connection.anonymous);
		recognized = connection == null ? 0 : Number(connection.recognized);
	}

	const latestDay = instantToDateKey(latest.observedAt);
	const result = days.filter((day) => day.day >= range.start && day.day < range.end && day.day < latestDay);
	for (let day = latestDay < range.start ? range.start : latestDay; day < range.end; day = addUTCDays(day, 1)) {
		result.push({ day, total: anonymous + recognized, anonymous, recognized });
	}
	return result;
};

// completeIdentityResolutionMetricDays extends historical days with the state
// anchored by the latest successful Resolution. The range end is exclusive.
const completeIdentityResolutionMetricDays = (
	days: IdentityResolutionMetricDay[],
	latest: IdentityResolutionMetric,
	range: DisplayDateRange,
): IdentityResolutionMetricDay[] => {
	const latestDay = instantToDateKey(latest.observedAt);
	const result = days.filter((day) => day.day >= range.start && day.day < range.end && day.day < latestDay);
	for (let day = latestDay < range.start ? range.start : latestDay; day < range.end; day = addUTCDays(day, 1)) {
		result.push({
			day,
			identities: Number(latest.identities.total),
			profiles: Number(latest.profiles.total),
			profilesAnonymous: Number(latest.profiles.anonymous),
			profilesRecognized: Number(latest.profiles.recognized),
			identitiesPerProfile: Number(latest.identitiesPerProfile),
			linkedIdentitiesRate: Number(latest.linkedIdentitiesRate),
		});
	}
	return result;
};

const aggregateConnections = (
	connections: IdentityConnectionMetric[] | undefined,
	names: ReadonlyMap<string, string>,
	top = 7,
): ConnectionBar[] => {
	const totals = new Map<string, { recognized: number; anonymous: number }>();
	for (const connection of connections ?? []) {
		const current = totals.get(connection.connection) ?? { recognized: 0, anonymous: 0 };
		totals.set(connection.connection, {
			recognized: current.recognized + connection.recognized,
			anonymous: current.anonymous + connection.anonymous,
		});
	}

	const all = Array.from(totals, ([connection, counts]) => ({
		connection,
		name: names.get(connection) ?? connection,
		recognized: counts.recognized,
		anonymous: counts.anonymous,
		total: counts.recognized + counts.anonymous,
	})).sort((left, right) => right.total - left.total || left.connection.localeCompare(right.connection));

	if (all.every((connection) => connection.total === 0)) {
		return [];
	}
	if (all.length <= top + 1) {
		return all;
	}

	const remaining = all.slice(top);
	return [
		...all.slice(0, top),
		{
			connection: '__other__',
			name: 'Other',
			recognized: remaining.reduce((total, connection) => total + connection.recognized, 0),
			anonymous: remaining.reduce((total, connection) => total + connection.anonymous, 0),
			total: remaining.reduce((total, connection) => total + connection.total, 0),
		},
	];
};

const buildIdentityConnectionOptions = (
	connections: readonly IdentityConnectionCatalogEntry[],
	metrics: readonly IdentityConnectionMetric[] | undefined,
): IdentityConnectionOption[] => {
	const totals = new Map<string, number>();
	for (const metric of metrics ?? []) {
		totals.set(
			metric.connection,
			(totals.get(metric.connection) ?? 0) + Number(metric.anonymous) + Number(metric.recognized),
		);
	}
	const options = new Map<string, string>();
	for (const connection of connections) {
		if (connection.role === 'Source') {
			options.set(connection.id, connection.name || connection.id);
		}
	}
	for (const metric of metrics ?? []) {
		if (metric.connection === DELETED_CONNECTION_SCOPE) continue;
		if (!options.has(metric.connection)) {
			options.set(metric.connection, metric.connection);
		}
	}
	const deletedTotal = totals.get(DELETED_CONNECTION_SCOPE) ?? 0;
	if (deletedTotal > 0) {
		options.set(DELETED_CONNECTION_SCOPE, DELETED_CONNECTION_LABEL);
	}
	return Array.from(options, ([id, name]) => ({ id, name })).sort((left, right) => {
		const leftIsDeleted = left.id === DELETED_CONNECTION_SCOPE;
		const rightIsDeleted = right.id === DELETED_CONNECTION_SCOPE;
		if (leftIsDeleted !== rightIsDeleted) return leftIsDeleted ? 1 : -1;
		return left.name.localeCompare(right.name) || left.id.localeCompare(right.id);
	});
};

// buildIdentityResolutionMetricChartDays restores every day in the displayed
// range so omitted API days preserve the existing no-data visualization.
const buildIdentityResolutionMetricChartDays = (
	days: IdentityResolutionMetricDay[],
	range: DisplayDateRange,
): IdentityResolutionMetricChartDay[] => {
	const metricsByDay = new Map(days.map((day) => [day.day, day]));
	const result: IdentityResolutionMetricChartDay[] = [];
	for (let day = range.start; day <= range.end; day = addUTCDays(day, 1)) {
		result.push(
			metricsByDay.get(day) ?? {
				day,
				identities: null,
				profiles: null,
				profilesAnonymous: null,
				profilesRecognized: null,
				identitiesPerProfile: null,
				linkedIdentitiesRate: null,
			},
		);
	}
	return result;
};

const buildResolutionEffectivenessData = (days: IdentityResolutionMetricChartDay[]): ResolutionEffectivenessPoint[] =>
	days.map((day) => ({
		day: day.day,
		linkedIdentitiesRatePercent: day.linkedIdentitiesRate == null ? null : Number(day.linkedIdentitiesRate) * 100,
		identitiesPerProfile: day.identitiesPerProfile == null ? null : Number(day.identitiesPerProfile),
	}));

const buildUnifiedProfileHistoryData = (days: IdentityResolutionMetricChartDay[]): UnifiedProfileHistoryPoint[] =>
	days.map((day) => ({
		day: day.day,
		recognized: day.profilesRecognized == null ? null : Number(day.profilesRecognized),
		anonymous: day.profilesAnonymous == null ? null : Number(day.profilesAnonymous),
	}));

const calculateResolutionKpiComparison = (
	latest: IdentityResolutionMetric | null,
	days: IdentityResolutionMetricDay[],
	displayRange: DisplayDateRange,
): ResolutionKpiComparison => {
	const comparisonLength = daysBetween(displayRange.start, displayRange.end) + 1;
	const previousEnd = addUTCDays(displayRange.end, -comparisonLength);
	const profilesTrend = calculateIdentityTrend(
		days.map((day) => ({ day: day.day, total: Number(day.profiles) })),
		displayRange.end,
		comparisonLength,
	);
	const linkedIdentitiesRateTrend = calculateIdentityTrend(
		days.map((day) => ({
			day: day.day,
			total: Number(day.linkedIdentitiesRate),
		})),
		displayRange.end,
		comparisonLength,
	);
	const identitiesPerProfileTrend = calculateIdentityTrend(
		days.map((day) => ({
			day: day.day,
			total: Number(day.identitiesPerProfile),
		})),
		displayRange.end,
		comparisonLength,
	);
	const currentProfiles = latest == null ? null : Number(latest.profiles.total);
	const referenceProfiles = profilesTrend.referenceValue;
	const currentLinkedIdentitiesRate = latest == null ? null : Number(latest.linkedIdentitiesRate);
	const referenceLinkedIdentitiesRate = linkedIdentitiesRateTrend.referenceValue;
	const currentIdentitiesPerProfile = latest == null ? null : Number(latest.identitiesPerProfile);
	const referenceIdentitiesPerProfile = identitiesPerProfileTrend.referenceValue;

	return {
		previousEnd,
		profilesChangePercent:
			currentProfiles == null || referenceProfiles == null || referenceProfiles === 0
				? null
				: ((currentProfiles - referenceProfiles) / referenceProfiles) * 100,
		linkedIdentitiesRatePercentagePoints:
			currentLinkedIdentitiesRate == null || referenceLinkedIdentitiesRate == null
				? null
				: (currentLinkedIdentitiesRate - referenceLinkedIdentitiesRate) * 100,
		identitiesPerProfileChange:
			currentIdentitiesPerProfile == null || referenceIdentitiesPerProfile == null
				? null
				: currentIdentitiesPerProfile - referenceIdentitiesPerProfile,
	};
};

const paddedMetricDomain = (
	values: number[],
	minimumPadding: number,
	fallback: ChartDomain,
	upperBound?: number,
): ChartDomain => {
	if (values.length === 0) return fallback;

	const minimum = Math.min(...values);
	const maximum = Math.max(...values);
	const spread = maximum - minimum;
	const padding =
		spread === 0 ? Math.max(Math.abs(maximum) * 0.05, minimumPadding) : Math.max(spread * 0.15, minimumPadding);
	const lower = Math.max(0, minimum - padding);
	const upper = upperBound == null ? maximum + padding : Math.min(upperBound, maximum + padding);

	return [Number(lower.toFixed(4)), Number(upper.toFixed(4))];
};

const identityLinkRateChartDomain = (points: ResolutionEffectivenessPoint[]): ChartDomain =>
	paddedMetricDomain(
		points.flatMap((point) =>
			point.linkedIdentitiesRatePercent == null ? [] : [point.linkedIdentitiesRatePercent],
		),
		1,
		[0, 100],
		100,
	);

const ratioChartDomain = (points: ResolutionEffectivenessPoint[]): ChartDomain => {
	const values = points.flatMap((point) => (point.identitiesPerProfile == null ? [] : [point.identitiesPerProfile]));
	return paddedMetricDomain(values, 0.05, [0, 1]);
};

const chartDomainTicks = ([minimum, maximum]: ChartDomain): [number, number, number] => [
	minimum,
	Number(((minimum + maximum) / 2).toFixed(4)),
	maximum,
];

const PROFILE_COMPOSITION_BUCKETS: Pick<ProfileCompositionBucket, 'key' | 'label'>[] = [
	{ key: 'one', label: '1 identity' },
	{ key: 'two', label: '2 identities' },
	{ key: 'three', label: '3 identities' },
	{ key: 'fourToTen', label: '4–10 identities' },
	{ key: 'elevenToTwenty', label: '11–20 identities' },
	{ key: 'moreThanTwenty', label: '21+ identities' },
];

const buildProfileComposition = (
	composition: IdentityResolutionComposition,
	profiles: number,
): ProfileCompositionBucket[] =>
	PROFILE_COMPOSITION_BUCKETS.map(({ key, label }) => ({
		key,
		label,
		count: composition[key],
		percentage: profiles === 0 ? null : (composition[key] / profiles) * 100,
	}));

const buildTypeDistribution = (
	recognized: number,
	anonymous: number,
	total = recognized + anonymous,
): TypeDistributionSegment[] => {
	return [
		{
			key: 'recognized',
			label: 'Recognized',
			count: recognized,
			percentage: total === 0 ? null : (recognized / total) * 100,
		},
		{
			key: 'anonymous',
			label: 'Anonymous',
			count: anonymous,
			percentage: total === 0 ? null : (anonymous / total) * 100,
		},
	];
};

const hasResolutionDataInRange = (days: IdentityResolutionMetricDay[], range: DisplayDateRange): boolean =>
	sliceDays(days, range).length !== 0;

const formatDate = (dateKey: string, includeYear = true): string => {
	return new Intl.DateTimeFormat('en-US', {
		timeZone: 'UTC',
		month: 'short',
		day: 'numeric',
		...(includeYear ? { year: 'numeric' as const } : {}),
	}).format(dateKeyToUTCDate(dateKey));
};

const formatComparisonDate = (referenceDay: string, currentDay: string): string => {
	const includeYear =
		dateKeyToUTCDate(referenceDay).getUTCFullYear() !== dateKeyToUTCDate(currentDay).getUTCFullYear();
	return formatDate(referenceDay, includeYear);
};

const formatChartDate = (dateKey: string): string => formatDate(dateKey, false);

const formatUTCTimestamp = (instant: string): string => {
	return new Intl.DateTimeFormat('en-US', {
		timeZone: 'UTC',
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		hour12: true,
	})
		.format(new Date(instant))
		.replace(/, (?=\d{2}:\d{2})/, ' ');
};

const formatTimestampInTimeZone = (instant: string, timeZone?: string, includeYear = true): string => {
	const parts = new Intl.DateTimeFormat('en-US', {
		timeZone,
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23',
	}).formatToParts(new Date(instant));
	const value = (type: Intl.DateTimeFormatPartTypes): string => parts.find((part) => part.type === type)?.value ?? '';

	const date = `${value('month')} ${value('day')}${includeYear ? `, ${value('year')}` : ''}`;
	return `${date} · ${value('hour')}:${value('minute')}`;
};

const formatResolutionUTCTimestamp = (instant: string, includeYear = true): string =>
	`${formatTimestampInTimeZone(instant, 'UTC', includeYear)} UTC`;

const formatResolutionComparisonUTCTimestamp = (instant: string, currentIdentityDay: string): string => {
	const resolutionDay = instantToDateKey(instant);
	const includeYear =
		dateKeyToUTCDate(resolutionDay).getUTCFullYear() !== dateKeyToUTCDate(currentIdentityDay).getUTCFullYear();
	return formatResolutionUTCTimestamp(instant, includeYear);
};

const formatRunDuration = (startTime: string, endTime: string | null): string => {
	if (endTime == null) return '—';
	const milliseconds = new Date(endTime).getTime() - new Date(startTime).getTime();
	if (!Number.isFinite(milliseconds) || milliseconds < 0) return '—';
	const seconds = Math.floor(milliseconds / 1000);
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ${minutes % 60}m`;
	return `${Math.floor(hours / 24)}d ${hours % 24}h`;
};

const resolvedTimeZone = (timeZone?: string): string =>
	timeZone ?? new Intl.DateTimeFormat().resolvedOptions().timeZone;

const formatResolutionLocalTimestamp = (instant: string, timeZone?: string, locale?: string): string => {
	const resolved = resolvedTimeZone(timeZone);
	const shortName = new Intl.DateTimeFormat(locale, {
		timeZone: resolved,
		timeZoneName: 'short',
	})
		.formatToParts(new Date(instant))
		.find((part) => part.type === 'timeZoneName')?.value;

	return `${formatTimestampInTimeZone(instant, resolved)}${shortName == null ? '' : ` ${shortName}`}`;
};

const formatResolutionLocalTimeZoneDetails = (instant: string, timeZone?: string): string => {
	const resolved = resolvedTimeZone(timeZone);
	const longOffset = new Intl.DateTimeFormat('en-US', {
		timeZone: resolved,
		timeZoneName: 'longOffset',
	})
		.formatToParts(new Date(instant))
		.find((part) => part.type === 'timeZoneName')?.value;
	const offset = longOffset == null || longOffset === 'GMT' ? '+00:00' : longOffset.replace(/^GMT/, '');

	return `${resolved} · UTC${offset}`;
};

const formatTrendPercent = (value: number | null): string => {
	if (value == null) return '—';
	const rounded = Number(value.toFixed(1));
	if (rounded === 0) return '0.0%';
	return `${rounded >= 0 ? '+' : ''}${rounded.toFixed(1)}%`;
};

const formatSignedIntegerDelta = (value: number | null): string => {
	if (value == null) return '—';
	const formatted = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(Math.abs(value));
	return `${value >= 0 ? '+' : '-'}${formatted}`;
};

const formatRate = (value: number | null): string => {
	return value == null ? '—' : `${(value * 100).toFixed(1)}%`;
};

const formatSharePercent = (value: number, total: number, fractionDigits = 1): string => {
	return total === 0 ? '—' : `${((value / total) * 100).toFixed(fractionDigits)}%`;
};

const formatPercentagePointDelta = (value: number | null): string => {
	if (value == null) return '—';
	const rounded = Number(value.toFixed(1));
	if (rounded === 0) return '0.0 pp';
	return `${rounded >= 0 ? '+' : ''}${rounded.toFixed(1)} pp`;
};

const formatRatio = (value: number): string =>
	new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(value);

const formatNullableRatio = (value: number | null): string => (value == null ? '—' : formatRatio(Number(value)));

const formatRatioDelta = (value: number | null): string => {
	if (value == null) return '—';
	const rounded = Number(value.toFixed(2));
	if (rounded === 0) return '0.00';
	return `${rounded >= 0 ? '+' : ''}${formatRatio(rounded)}`;
};

export {
	DELETED_CONNECTION_LABEL,
	DELETED_CONNECTION_SCOPE,
	IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET,
	addUTCDays,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityMetricChartDays,
	buildIdentityResolutionMetricChartDays,
	buildIdentityConnectionOptions,
	buildIdentityChangeSinceResolutionPoints,
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
	dateKeyToPickerDate,
	displayRangeForPreset,
	formatChartDate,
	formatComparisonDate,
	formatDate,
	formatNullableRatio,
	formatPercentagePointDelta,
	formatRatio,
	formatRatioDelta,
	formatRate,
	formatResolutionComparisonUTCTimestamp,
	formatRunDuration,
	formatResolutionLocalTimeZoneDetails,
	formatResolutionLocalTimestamp,
	formatResolutionUTCTimestamp,
	formatSharePercent,
	formatSignedIntegerDelta,
	formatTrendPercent,
	formatUTCTimestamp,
	hasResolutionDataInRange,
	identityLinkRateChartDomain,
	instantToDateKey,
	pickerDateToDateKey,
	ratioChartDomain,
	sparklineDomain,
	sliceDays,
	todayUTCDateKey,
};

export type {
	ConnectionBar,
	DisplayDateRange,
	IdentityDashboardDatePreset,
	IdentityTrend,
	ProfileCompositionBucket,
	ResolutionEffectivenessPoint,
	UnifiedProfileHistoryPoint,
};
