import { IdentityConnectionMetric, IdentityMetric, IdentityMetricDay } from '../../../lib/api/types/metrics';

const DELETED_CONNECTION_SCOPE = 'deleted';
const DELETED_CONNECTION_LABEL = 'Removed connections';

interface DisplayDateRange {
	start: string;
	end: string;
}

interface IdentityMetricChartDay {
	day: string;
	total: number | null;
	anonymous: number | null;
	recognized: number | null;
}

type IdentityOverviewDatePreset = 'last7Days' | 'last30Days' | 'last90Days';

const IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET: IdentityOverviewDatePreset = 'last30Days';

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

const displayRangeForPreset = (preset: IdentityOverviewDatePreset, end: string): DisplayDateRange => {
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

const formatTrendPercent = (value: number | null): string => {
	if (value == null) return '—';
	const rounded = Number(value.toFixed(1));
	if (rounded === 0) return '0.0%';
	return `${rounded >= 0 ? '+' : ''}${rounded.toFixed(1)}%`;
};

const formatSharePercent = (value: number, total: number, fractionDigits = 1): string => {
	return total === 0 ? '—' : `${((value / total) * 100).toFixed(fractionDigits)}%`;
};
export {
	DELETED_CONNECTION_LABEL,
	DELETED_CONNECTION_SCOPE,
	IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET,
	addUTCDays,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityConnectionOptions,
	buildIdentityMetricChartDays,
	calculateIdentityTrend,
	completeIdentityMetricDays,
	computeFetchRange,
	dateKeyToPickerDate,
	displayRangeForPreset,
	formatChartDate,
	formatComparisonDate,
	formatDate,
	formatSharePercent,
	formatTrendPercent,
	formatUTCTimestamp,
	instantToDateKey,
	pickerDateToDateKey,
	sparklineDomain,
	todayUTCDateKey,
};

export type { ConnectionBar, DisplayDateRange, IdentityOverviewDatePreset, IdentityTrend };
