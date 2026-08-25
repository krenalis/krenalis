import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import './IdentityOverview.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import AppContext from '../../../context/AppContext';
import SegmentedDateRangeControl, {
	SegmentedDateRangePreset,
	SegmentedDateRangeSelection,
} from '../../base/SegmentedDateRangeControl/SegmentedDateRangeControl';
import { IdentityMetric, IdentityMetricDay } from '../../../lib/api/types/metrics';
import { formatNumber } from '../../../utils/formatNumber';
import {
	DisplayDateRange,
	DELETED_CONNECTION_LABEL,
	DELETED_CONNECTION_SCOPE,
	IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET,
	IdentityOverviewDatePreset,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityConnectionOptions,
	buildIdentityMetricChartDays,
	calculateIdentityTrend,
	completeIdentityMetricDays,
	computeFetchRange,
	dateKeyToPickerDate,
	displayRangeForPreset,
	formatComparisonDate,
	formatDate,
	formatTrendPercent,
	formatUTCTimestamp,
	instantToDateKey,
	pickerDateToDateKey,
	todayUTCDateKey,
} from './IdentityOverview.helpers';
import {
	ConnectionsChart,
	IdentitiesChart,
	KpiCard,
	SectionHeading,
	StateMessage,
} from './IdentityOverview.components';

const initialDisplayRange = (): DisplayDateRange => {
	const end = todayUTCDateKey();
	return displayRangeForPreset(IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET, end);
};

const initialCustomRange = (): SegmentedDateRangeSelection[] => {
	const range = initialDisplayRange();
	return [
		{
			startDate: dateKeyToPickerDate(range.start),
			endDate: dateKeyToPickerDate(range.end),
			key: 'selection',
		},
	];
};

const DATE_RANGE_PRESETS: SegmentedDateRangePreset<IdentityOverviewDatePreset>[] = [
	{ value: 'last7Days', label: 'Last 7 days' },
	{ value: 'last30Days', label: 'Last 30 days' },
	{ value: 'last90Days', label: 'Last 90 days' },
];

const getErrorMessage = (error: unknown): string => {
	if (error instanceof Error) return error.message;
	return String(error);
};

interface LoadMetricsOptions {
	preserveData?: boolean;
}

const IdentityOverview = () => {
	const { api, connections, selectedWorkspace, setTitle } = useContext(AppContext);
	const [displayRange, setDisplayRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [loadedDisplayRange, setLoadedDisplayRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [selectedDateRange, setSelectedDateRange] = useState<IdentityOverviewDatePreset | 'Custom'>(
		IDENTITY_OVERVIEW_DEFAULT_DATE_PRESET,
	);
	const [customDateRange, setCustomDateRange] = useState<SegmentedDateRangeSelection[]>(initialCustomRange);
	const [latestMetric, setLatestMetric] = useState<IdentityMetric>();
	const [metricDays, setMetricDays] = useState<IdentityMetricDay[]>();
	const [selectedConnection, setSelectedConnection] = useState('');
	const [connectionMetricDays, setConnectionMetricDays] = useState<IdentityMetricDay[]>();
	const [loadedConnectionRange, setLoadedConnectionRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [connectionError, setConnectionError] = useState<string>();
	const [isConnectionLoading, setIsConnectionLoading] = useState(false);
	const [error, setError] = useState<string>();
	const [isLoading, setIsLoading] = useState(true);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const requestVersion = useRef(0);
	const connectionRequestVersion = useRef(0);
	const latestMetricRef = useRef<IdentityMetric>();
	const selectedConnectionRef = useRef('');
	const previousWorkspace = useRef(selectedWorkspace);
	const sourceConnectionCatalogKey = useMemo(
		() =>
			connections
				.filter((connection) => connection.role === 'Source')
				.map((connection) => connection.id)
				.sort()
				.join(','),
		[connections],
	);
	const previousSourceConnectionCatalogKey = useRef(sourceConnectionCatalogKey);

	useLayoutEffect(() => {
		setTitle('Profile Unification / Overview');
	}, [setTitle]);

	const fetchMetricDays = useCallback(
		async (latest: IdentityMetric, requestedRange: DisplayDateRange, connection?: string) => {
			const latestDay = instantToDateKey(latest.observedAt);
			const fetchRange = computeFetchRange(requestedRange, latestDay);
			const days =
				fetchRange.start < latestDay
					? await api.workspaces.identityMetricsPerDate(fetchRange.start, latestDay, connection)
					: [];
			return completeIdentityMetricDays(days, latest, fetchRange, connection);
		},
		[api],
	);

	const loadConnectionMetrics = useCallback(
		async (latest: IdentityMetric, connection: string, requestedRange: DisplayDateRange) => {
			const version = ++connectionRequestVersion.current;
			setConnectionError(undefined);
			setIsConnectionLoading(true);
			try {
				const days = await fetchMetricDays(latest, requestedRange, connection);
				if (version !== connectionRequestVersion.current || connection !== selectedConnectionRef.current)
					return;
				setLoadedConnectionRange(requestedRange);
				setConnectionMetricDays(days);
			} catch (err) {
				if (version !== connectionRequestVersion.current || connection !== selectedConnectionRef.current)
					return;
				setLoadedConnectionRange(requestedRange);
				setConnectionMetricDays(undefined);
				setConnectionError(getErrorMessage(err));
			} finally {
				if (version === connectionRequestVersion.current) setIsConnectionLoading(false);
			}
		},
		[fetchMetricDays],
	);

	const loadMetrics = useCallback(
		async ({ preserveData = false }: LoadMetricsOptions = {}) => {
			const version = ++requestVersion.current;
			const requestedRange = displayRange;
			setError(undefined);
			if (!preserveData) {
				setIsLoading(true);
				setLatestMetric(undefined);
				setMetricDays(undefined);
			}
			try {
				let latest = latestMetricRef.current;
				if (latest == null) {
					latest = await api.workspaces.latestIdentityMetric();
					if (version !== requestVersion.current) return;
					latestMetricRef.current = latest;
				}
				const connection = selectedConnectionRef.current;
				const [days] = await Promise.all([
					fetchMetricDays(latest, requestedRange),
					connection === '' ? Promise.resolve() : loadConnectionMetrics(latest, connection, requestedRange),
				]);
				if (version !== requestVersion.current) return;
				setLoadedDisplayRange(requestedRange);
				setLatestMetric(latest);
				setMetricDays(days);
			} catch (err) {
				if (version !== requestVersion.current) return;
				setLoadedDisplayRange(requestedRange);
				setLatestMetric(undefined);
				setMetricDays(undefined);
				setError(getErrorMessage(err));
			} finally {
				if (version === requestVersion.current) setIsLoading(false);
			}
		},
		[api, displayRange, fetchMetricDays, loadConnectionMetrics],
	);

	useEffect(() => {
		const workspaceChanged = previousWorkspace.current !== selectedWorkspace;
		const connectionCatalogChanged = previousSourceConnectionCatalogKey.current !== sourceConnectionCatalogKey;
		previousWorkspace.current = selectedWorkspace;
		previousSourceConnectionCatalogKey.current = sourceConnectionCatalogKey;
		if (workspaceChanged || connectionCatalogChanged) latestMetricRef.current = undefined;
		if (workspaceChanged) {
			connectionRequestVersion.current += 1;
			selectedConnectionRef.current = '';
			setSelectedConnection('');
			setConnectionMetricDays(undefined);
			setConnectionError(undefined);
			setIsConnectionLoading(false);
		}
		void loadMetrics();
	}, [loadMetrics, selectedWorkspace, sourceConnectionCatalogKey]);

	useEffect(() => {
		if (latestMetric == null || selectedDateRange === 'Custom') return;
		const latestDay = instantToDateKey(latestMetric.observedAt);
		setDisplayRange((current) => {
			const next = displayRangeForPreset(selectedDateRange, latestDay);
			return current.start === next.start && current.end === next.end ? current : next;
		});
	}, [latestMetric, selectedDateRange]);

	const identityChartDays = useMemo(
		() => buildIdentityMetricChartDays(metricDays ?? [], loadedDisplayRange),
		[metricDays, loadedDisplayRange],
	);
	const selectedConnectionDays = useMemo(
		() => buildIdentityMetricChartDays(connectionMetricDays ?? [], loadedConnectionRange),
		[connectionMetricDays, loadedConnectionRange],
	);
	const showAllConnections = selectedConnection === '';
	const connectionChartDays = connectionMetricDays == null ? [] : selectedConnectionDays;
	const chartDays = showAllConnections ? identityChartDays : connectionChartDays;
	const chartLoading = showAllConnections
		? isLoading
		: isConnectionLoading && connectionMetricDays == null && metricDays == null;
	const chartError = showAllConnections ? error : connectionError;
	const latestDay = latestMetric == null ? loadedDisplayRange.end : instantToDateKey(latestMetric.observedAt);
	const sevenDayTrend = useMemo(
		() => calculateIdentityTrend(metricDays ?? [], latestDay, 7),
		[latestDay, metricDays],
	);
	const thirtyDayTrend = useMemo(
		() => calculateIdentityTrend(metricDays ?? [], latestDay, 30),
		[latestDay, metricDays],
	);
	const deletedConnectionMetric = useMemo(() => buildDeletedConnectionMetric(latestMetric ?? null), [latestMetric]);
	const latestConnectionMetrics = useMemo(() => {
		const metrics = [...(latestMetric?.connections ?? [])];
		if (deletedConnectionMetric == null) return metrics;
		const deletedTotal =
			deletedConnectionMetric.anonymous +
			deletedConnectionMetric.recognized +
			deletedConnectionMetric.withoutProfile;
		if (deletedTotal > 0) metrics.push(deletedConnectionMetric);
		return metrics;
	}, [deletedConnectionMetric, latestMetric]);
	const connectionNames = useMemo(
		() =>
			new Map([
				...connections.map((connection) => [connection.id, connection.name] as const),
				[DELETED_CONNECTION_SCOPE, DELETED_CONNECTION_LABEL] as const,
			]),
		[connections],
	);
	const connectionBars = useMemo(
		() => aggregateConnections(latestConnectionMetrics, connectionNames),
		[latestConnectionMetrics, connectionNames],
	);
	const connectionOptions = useMemo(
		() => buildIdentityConnectionOptions(connections, latestConnectionMetrics),
		[connections, latestConnectionMetrics],
	);
	const observedLabel = latestMetric == null ? undefined : formatDate(instantToDateKey(latestMetric.observedAt));
	const connectionObservedLabel = latestMetric == null ? undefined : formatUTCTimestamp(latestMetric.observedAt);

	const onPresetChange = (preset: IdentityOverviewDatePreset) => {
		const end = latestMetric == null ? todayUTCDateKey() : instantToDateKey(latestMetric.observedAt);
		setSelectedDateRange(preset);
		setDisplayRange(displayRangeForPreset(preset, end));
	};

	const onConnectionChange = (connection: string) => {
		connectionRequestVersion.current += 1;
		selectedConnectionRef.current = connection;
		setSelectedConnection(connection);
		setConnectionError(undefined);
		if (connection === '') {
			setConnectionMetricDays(undefined);
			setIsConnectionLoading(false);
			return;
		}
		const latest = latestMetricRef.current;
		if (latest == null) {
			setIsConnectionLoading(true);
			return;
		}
		void loadConnectionMetrics(latest, connection, displayRange);
	};

	const refreshDashboard = async () => {
		setIsRefreshing(true);
		setError(undefined);
		try {
			await api.workspaces.refreshIdentityMetrics();
			latestMetricRef.current = undefined;
			await loadMetrics({ preserveData: true });
		} catch (err) {
			setError(getErrorMessage(err));
		} finally {
			setIsRefreshing(false);
		}
	};

	const onCustomRangeChange = (selection: SegmentedDateRangeSelection[]) => {
		const range = selection[0];
		if (range == null) return;
		setCustomDateRange(selection);
		setSelectedDateRange('Custom');
		setDisplayRange({
			start: pickerDateToDateKey(range.startDate),
			end: pickerDateToDateKey(range.endDate),
		});
	};

	return (
		<main className='identity-overview'>
			<div className='identity-overview__page-header'>
				<div>
					<h1>Profile unification overview</h1>
					<p>Understand your current identity state and trends.</p>
				</div>
				<div className='identity-overview__page-actions'>
					<SlButton
						className='segmented-date-range-button'
						variant='default'
						size='small'
						onClick={refreshDashboard}
						loading={isRefreshing}
						disabled={isRefreshing || isLoading}
					>
						<SlIcon slot='prefix' name='arrow-clockwise' />
						Refresh
					</SlButton>
					<SegmentedDateRangeControl<IdentityOverviewDatePreset>
						accessibleLabel='Trend range'
						presets={DATE_RANGE_PRESETS}
						value={selectedDateRange}
						customRange={customDateRange}
						onPresetChange={onPresetChange}
						onCustomRangeChange={onCustomRangeChange}
						pickerAlignment='end'
					/>
				</div>
			</div>

			<section className='identity-overview__section identity-overview__section--current-state'>
				<SectionHeading
					title='Current identity state'
					secondary={observedLabel == null ? undefined : <>(as of {observedLabel})</>}
					info='The latest observed workspace identity state, not a sum of daily values.'
				/>
				{error && (
					<StateMessage
						variant='error'
						title='Current identity state is unavailable'
						description={error}
						compact
					/>
				)}
				<div className='identity-overview__kpi-grid'>
					<KpiCard
						title='Total identities'
						value={latestMetric == null ? '—' : formatNumber(latestMetric.total)}
						secondary={observedLabel == null ? undefined : <>As of {observedLabel}</>}
						subtleBackground
						loading={isLoading}
					/>
					<KpiCard
						title='Over the last 7 days'
						info='Compares the latest identity state with the identity state as of exactly seven days earlier.'
						value={formatTrendPercent(sevenDayTrend.changePercent)}
						secondary={
							sevenDayTrend.changePercent == null
								? undefined
								: `from ${formatComparisonDate(sevenDayTrend.referenceDay, sevenDayTrend.currentDay)}`
						}
						sparklinePoints={latestMetric == null ? undefined : sevenDayTrend.points}
						loading={isLoading}
					/>
					<KpiCard
						title='Over the last 30 days'
						info='Compares the latest identity state with the identity state as of exactly 30 days earlier.'
						value={formatTrendPercent(thirtyDayTrend.changePercent)}
						secondary={
							thirtyDayTrend.changePercent == null
								? undefined
								: `from ${formatComparisonDate(thirtyDayTrend.referenceDay, thirtyDayTrend.currentDay)}`
						}
						sparklinePoints={latestMetric == null ? undefined : thirtyDayTrend.points}
						loading={isLoading}
					/>
				</div>
				<div className='identity-overview__current-charts'>
					<IdentitiesChart
						days={chartDays}
						loading={chartLoading}
						error={chartError}
						connectionOptions={connectionOptions}
						selectedConnection={selectedConnection}
						onConnectionChange={onConnectionChange}
					/>
					<ConnectionsChart
						data={connectionBars}
						totalIdentities={latestMetric?.total ?? null}
						observedLabel={connectionObservedLabel}
						loading={isLoading}
						error={error}
					/>
				</div>
			</section>
		</main>
	);
};

export default IdentityOverview;
