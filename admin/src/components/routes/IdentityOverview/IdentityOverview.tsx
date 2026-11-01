import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import './IdentityOverview.css';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AppContext from '../../../context/AppContext';
import SegmentedDateRangeControl, {
	SegmentedDateRangePreset,
	SegmentedDateRangeSelection,
} from '../../base/SegmentedDateRangeControl/SegmentedDateRangeControl';
import {
	IdentityMetric,
	IdentityMetricDay,
	IdentityResolutionMetric,
	IdentityResolutionMetricDay,
} from '../../../lib/api/types/metrics';
import { IdentityResolutionRun } from '../../../lib/api/types/workspace';
import { NotFoundError } from '../../../lib/api/errors';
import { formatNumber } from '../../../utils/formatNumber';
import {
	DisplayDateRange,
	DELETED_CONNECTION_LABEL,
	DELETED_CONNECTION_SCOPE,
	IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET,
	IdentityDashboardDatePreset,
	aggregateConnections,
	buildDeletedConnectionMetric,
	buildIdentityChangeSinceResolutionPoints,
	buildIdentityConnectionOptions,
	buildIdentityMetricChartDays,
	buildIdentityResolutionMetricChartDays,
	buildResolutionEffectivenessData,
	buildTemporalSemantics,
	buildUnifiedProfileHistoryData,
	calculateIdentityChangeSinceResolution,
	calculateIdentityTrend,
	calculateResolutionKpiComparison,
	completeIdentityMetricDays,
	completeIdentityResolutionMetricDays,
	computeFetchRange,
	dateKeyToPickerDate,
	displayRangeForPreset,
	formatComparisonDate,
	formatDate,
	formatNullableRatio,
	formatPercentagePointDelta,
	formatRate,
	formatRatioDelta,
	formatResolutionComparisonUTCTimestamp,
	formatResolutionLocalTimeZoneDetails,
	formatResolutionLocalTimestamp,
	formatResolutionUTCTimestamp,
	formatSignedIntegerDelta,
	formatTrendPercent,
	hasResolutionDataInRange,
	instantToDateKey,
	pickerDateToDateKey,
	todayUTCDateKey,
} from './IdentityOverview.helpers';
import {
	ConnectionsChart,
	HistorySection,
	IDENTITY_LINK_RATE_COLOR,
	IDENTITIES_PER_PROFILE_COLOR,
	IdentitiesChart,
	KpiCard,
	NoResolutionState,
	ProfilesChart,
	ProfileComposition,
	ResolutionEffectivenessChart,
	ResolutionPeriodEmptyState,
	SectionHeading,
	StateMessage,
	TypeDistributionCard,
} from './IdentityOverview.components';

const initialDisplayRange = (): DisplayDateRange => {
	const end = todayUTCDateKey();
	return displayRangeForPreset(IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET, end);
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

const DATE_RANGE_PRESETS: SegmentedDateRangePreset<IdentityDashboardDatePreset>[] = [
	{ value: 'last7Days', label: 'Last 7 days' },
	{ value: 'last30Days', label: 'Last 30 days' },
	{ value: 'last90Days', label: 'Last 90 days' },
];

const getErrorMessage = (error: unknown): string => {
	if (error instanceof Error) return error.message;
	return String(error);
};

interface DashboardData {
	displayRange: DisplayDateRange;
	latestIdentityMetric: IdentityMetric;
	identityMetricDays: IdentityMetricDay[];
	latestResolutionMetric: IdentityResolutionMetric | null;
	resolutionMetricDays: IdentityResolutionMetricDay[];
	resolutionRuns: IdentityResolutionRun[];
	selectedIdentityConnection: string;
	connectionIdentityMetricDays?: IdentityMetricDay[];
}

const IdentityOverview = () => {
	const { api, connections, selectedWorkspace, setTitle } = useContext(AppContext);
	const [displayRange, setDisplayRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [selectedDateRange, setSelectedDateRange] = useState<IdentityDashboardDatePreset | 'Custom'>(
		IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET,
	);
	const [customDateRange, setCustomDateRange] = useState<SegmentedDateRangeSelection[]>(initialCustomRange);
	const [today, setToday] = useState<string>(todayUTCDateKey);
	const [dashboardData, setDashboardData] = useState<DashboardData>();
	const [dashboardError, setDashboardError] = useState<string>();
	const [selectedIdentityConnection, setSelectedIdentityConnection] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const requestVersion = useRef<number>(0);
	// Only completed loads can supply cached snapshots for range and connection changes.
	const dashboardDataRef = useRef<DashboardData>();
	const previousWorkspace = useRef<string>(selectedWorkspace);
	const {
		latestIdentityMetric,
		identityMetricDays,
		latestResolutionMetric,
		resolutionMetricDays,
		resolutionRuns = [],
		connectionIdentityMetricDays,
		selectedIdentityConnection: loadedIdentityConnection = '',
		displayRange: loadedDisplayRange = displayRange,
	} = dashboardData ?? {};
	const isInitialLoading = isLoading && dashboardData == null;
	const selectedDisplayRange =
		selectedDateRange === 'Custom' || latestIdentityMetric == null
			? displayRange
			: displayRangeForPreset(selectedDateRange, instantToDateKey(latestIdentityMetric.observedAt));
	const isReloading = isLoading && dashboardData != null;
	const isRangeReloading =
		isReloading &&
		(selectedDisplayRange.start !== loadedDisplayRange.start ||
			selectedDisplayRange.end !== loadedDisplayRange.end);
	const isIdentityHistoryReloading =
		isReloading && (isRangeReloading || selectedIdentityConnection !== loadedIdentityConnection);
	const chartError = dashboardData == null ? dashboardError : undefined;
	const sourceConnectionCatalogKey = useMemo(
		() =>
			connections
				.filter((connection) => connection.role === 'Source')
				.map((connection) => connection.id)
				.sort()
				.join(','),
		[connections],
	);
	const previousSourceConnectionCatalogKey = useRef<string>(sourceConnectionCatalogKey);

	useLayoutEffect(() => {
		setTitle('Profile Unification / Identity dashboard');
	}, [setTitle]);

	useEffect(() => {
		let timeout: number;
		const updateToday = () => {
			const now = new Date();
			setToday(todayUTCDateKey(now));
			const nextUTCDay = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + 1);
			timeout = window.setTimeout(updateToday, nextUTCDay - now.getTime() + 1);
		};
		updateToday();
		return () => window.clearTimeout(timeout);
	}, []);

	const fetchIdentityMetricDays = useCallback(
		async (latest: IdentityMetric, requestedRange: DisplayDateRange, connectionSelection?: string) => {
			const latestDay = instantToDateKey(latest.observedAt);
			const fetchRange = computeFetchRange(requestedRange, latestDay);
			// Exclude the latest day so observations committed after Latest cannot change the queried history.
			const days =
				fetchRange.start < latestDay
					? await api.workspaces.identityMetricsPerDate(fetchRange.start, latestDay, connectionSelection)
					: [];
			return completeIdentityMetricDays(days, latest, fetchRange, connectionSelection);
		},
		[api],
	);

	const fetchResolutionMetricDays = useCallback(
		async (latest: IdentityResolutionMetric, requestedRange: DisplayDateRange) => {
			const latestDay = instantToDateKey(latest.observedAt);
			const fetchRange = computeFetchRange(requestedRange);
			const historyEnd = fetchRange.end < latestDay ? fetchRange.end : latestDay;
			// Exclude the latest day so observations committed after Latest cannot change the queried history.
			const days =
				fetchRange.start < historyEnd
					? await api.workspaces.identityResolutionMetricsPerDate(fetchRange.start, historyEnd)
					: [];
			return completeIdentityResolutionMetricDays(days, latest, fetchRange);
		},
		[api],
	);

	const loadMetrics = useCallback(async () => {
		const version = ++requestVersion.current;
		setDashboardError(undefined);
		setIsLoading(true);
		try {
			const cached = dashboardDataRef.current;
			const [identity, resolution, runs] = await Promise.all([
				cached == null ? api.workspaces.latestIdentityMetric() : cached.latestIdentityMetric,
				cached == null ? api.workspaces.latestIdentityResolutionMetric() : cached.latestResolutionMetric,
				cached == null ? api.workspaces.identityResolutionRuns() : { runs: cached.resolutionRuns },
			]);
			if (version !== requestVersion.current) return;

			const requestedRange =
				selectedDateRange === 'Custom'
					? displayRange
					: displayRangeForPreset(selectedDateRange, instantToDateKey(identity.observedAt));
			const hasCachedRange =
				cached != null &&
				cached.displayRange.start === requestedRange.start &&
				cached.displayRange.end === requestedRange.end;
			const [identityDays, resolutionDays, connectionDays] = await Promise.all([
				hasCachedRange ? cached.identityMetricDays : fetchIdentityMetricDays(identity, requestedRange),
				hasCachedRange
					? cached.resolutionMetricDays
					: resolution == null
						? []
						: fetchResolutionMetricDays(resolution, requestedRange),
				selectedIdentityConnection === ''
					? undefined
					: fetchIdentityMetricDays(identity, requestedRange, selectedIdentityConnection).catch((err) => {
							if (
								version === requestVersion.current &&
								selectedIdentityConnection !== DELETED_CONNECTION_SCOPE &&
								err instanceof NotFoundError
							) {
								// Discard this load before retrying with a valid selection.
								requestVersion.current += 1;
								dashboardDataRef.current = undefined;
								setSelectedIdentityConnection('');
							}
							throw err;
						}),
			]);
			if (version !== requestVersion.current) return;

			const data: DashboardData = {
				displayRange: requestedRange,
				latestIdentityMetric: identity,
				identityMetricDays: identityDays,
				latestResolutionMetric: resolution,
				resolutionMetricDays: resolutionDays,
				resolutionRuns: runs.runs,
				selectedIdentityConnection,
				connectionIdentityMetricDays: connectionDays,
			};
			dashboardDataRef.current = data;
			setDashboardData(data);
		} catch (err) {
			if (version !== requestVersion.current) return;
			setDashboardError(getErrorMessage(err));
		} finally {
			if (version === requestVersion.current) setIsLoading(false);
		}
	}, [
		api,
		displayRange,
		selectedDateRange,
		selectedIdentityConnection,
		fetchIdentityMetricDays,
		fetchResolutionMetricDays,
	]);

	useLayoutEffect(() => {
		const workspaceChanged = previousWorkspace.current !== selectedWorkspace;
		const connectionCatalogChanged = previousSourceConnectionCatalogKey.current !== sourceConnectionCatalogKey;
		previousWorkspace.current = selectedWorkspace;
		previousSourceConnectionCatalogKey.current = sourceConnectionCatalogKey;
		if (workspaceChanged || connectionCatalogChanged) dashboardDataRef.current = undefined;
		if (workspaceChanged) {
			setDashboardData(undefined);
			setSelectedIdentityConnection('');
		}
		loadMetrics();
		return () => {
			requestVersion.current += 1;
		};
	}, [loadMetrics, selectedWorkspace, sourceConnectionCatalogKey]);

	const connectionNames = useMemo(
		() =>
			new Map([
				...connections.map((connection) => [connection.id, connection.name] as const),
				[DELETED_CONNECTION_SCOPE, DELETED_CONNECTION_LABEL] as const,
			]),
		[connections],
	);
	const identityDays = useMemo(
		() => buildIdentityMetricChartDays(identityMetricDays ?? [], loadedDisplayRange),
		[identityMetricDays, loadedDisplayRange],
	);
	const selectedConnectionIdentityDays = useMemo(
		() => buildIdentityMetricChartDays(connectionIdentityMetricDays ?? [], loadedDisplayRange),
		[connectionIdentityMetricDays, loadedDisplayRange],
	);
	const identityChartDays = loadedIdentityConnection === '' ? identityDays : selectedConnectionIdentityDays;
	const temporalSemantics = useMemo(
		() =>
			buildTemporalSemantics(
				loadedDisplayRange,
				latestIdentityMetric?.observedAt,
				latestResolutionMetric?.observedAt,
			),
		[latestIdentityMetric, latestResolutionMetric, loadedDisplayRange],
	);
	const fixedIdentityTrendEnd = temporalSemantics.latestIdentityDay ?? loadedDisplayRange.end;
	const sevenDayTrend = useMemo(
		() => calculateIdentityTrend(identityMetricDays ?? [], fixedIdentityTrendEnd, 7),
		[fixedIdentityTrendEnd, identityMetricDays],
	);
	const thirtyDayTrend = useMemo(
		() => calculateIdentityTrend(identityMetricDays ?? [], fixedIdentityTrendEnd, 30),
		[fixedIdentityTrendEnd, identityMetricDays],
	);
	const deletedConnectionMetric = useMemo(
		() => buildDeletedConnectionMetric(latestIdentityMetric ?? null),
		[latestIdentityMetric],
	);
	const latestConnectionMetrics = useMemo(() => {
		const metrics = [...(latestIdentityMetric?.connections ?? [])];
		if (deletedConnectionMetric == null) return metrics;
		const deletedTotal = deletedConnectionMetric.anonymous + deletedConnectionMetric.recognized;
		if (deletedTotal > 0) metrics.push(deletedConnectionMetric);
		return metrics;
	}, [deletedConnectionMetric, latestIdentityMetric]);
	const connectionBars = useMemo(
		() => aggregateConnections(latestConnectionMetrics, connectionNames),
		[latestConnectionMetrics, connectionNames],
	);
	const identityConnectionOptions = useMemo(
		() => buildIdentityConnectionOptions(connections, latestConnectionMetrics),
		[connections, latestConnectionMetrics],
	);
	useEffect(() => {
		if (
			selectedIdentityConnection === '' ||
			identityConnectionOptions.some((connection) => connection.id === selectedIdentityConnection)
		) {
			return;
		}
		setSelectedIdentityConnection('');
	}, [identityConnectionOptions, selectedIdentityConnection]);
	const resolutionChartDays = useMemo(
		() => buildIdentityResolutionMetricChartDays(resolutionMetricDays ?? [], loadedDisplayRange),
		[resolutionMetricDays, loadedDisplayRange],
	);
	const resolutionEffectivenessData = useMemo(
		() => buildResolutionEffectivenessData(resolutionChartDays),
		[resolutionChartDays],
	);
	const unifiedProfileHistoryData = useMemo(
		() => buildUnifiedProfileHistoryData(resolutionChartDays),
		[resolutionChartDays],
	);
	const hasResolutionInPeriod = useMemo(
		() => hasResolutionDataInRange(resolutionMetricDays ?? [], loadedDisplayRange),
		[resolutionMetricDays, loadedDisplayRange],
	);
	const resolutionKpiComparison = useMemo(
		() =>
			calculateResolutionKpiComparison(
				latestResolutionMetric ?? null,
				resolutionMetricDays ?? [],
				loadedDisplayRange,
			),
		[latestResolutionMetric, resolutionMetricDays, loadedDisplayRange],
	);

	const observedLabel =
		temporalSemantics.latestIdentityDay == null ? undefined : formatDate(temporalSemantics.latestIdentityDay);
	const connectionObservedLabel =
		temporalSemantics.latestIdentityDay == null
			? undefined
			: formatComparisonDate(temporalSemantics.latestIdentityDay, todayUTCDateKey());
	const latestResolution = latestResolutionMetric;
	const latestResolutionUTCTimestamp =
		latestResolution == null ? undefined : formatResolutionUTCTimestamp(latestResolution.observedAt);
	const latestResolutionComparisonUTCTimestamp =
		latestResolution == null || temporalSemantics.latestIdentityDay == null
			? latestResolutionUTCTimestamp
			: formatResolutionComparisonUTCTimestamp(latestResolution.observedAt, temporalSemantics.latestIdentityDay);
	const canCompareIdentitiesSinceResolution =
		latestIdentityMetric != null &&
		latestResolution != null &&
		Date.parse(latestIdentityMetric.observedAt) >= Date.parse(latestResolution.observedAt);
	const identitiesChangeSinceResolution = canCompareIdentitiesSinceResolution
		? calculateIdentityChangeSinceResolution(
				Number(latestIdentityMetric.total),
				Number(latestResolution.identities.total),
			)
		: null;
	const identitiesChangeSinceResolutionPoints = canCompareIdentitiesSinceResolution
		? buildIdentityChangeSinceResolutionPoints(
				latestIdentityMetric.observedAt,
				Number(latestIdentityMetric.total),
				latestResolution.observedAt,
				Number(latestResolution.identities.total),
			)
		: [];
	const resolutionComparisonDate = formatComparisonDate(
		resolutionKpiComparison.previousEnd,
		latestResolution == null ? resolutionKpiComparison.previousEnd : instantToDateKey(latestResolution.observedAt),
	);
	const browserTimeZone = new Intl.DateTimeFormat().resolvedOptions().timeZone;
	const shouldShowResolutionKpiComparison = temporalSemantics.shouldShowLatestResolutionComparison;
	const unifiedProfilesComparison =
		shouldShowResolutionKpiComparison && resolutionKpiComparison.profilesChangePercent != null
			? {
					delta: formatTrendPercent(resolutionKpiComparison.profilesChangePercent),
					referenceDate: resolutionComparisonDate,
				}
			: undefined;
	const identitiesPerProfileComparison =
		shouldShowResolutionKpiComparison && resolutionKpiComparison.identitiesPerProfileChange != null
			? {
					delta: formatRatioDelta(resolutionKpiComparison.identitiesPerProfileChange),
					referenceDate: resolutionComparisonDate,
				}
			: undefined;
	const identityLinkRateComparison =
		shouldShowResolutionKpiComparison && resolutionKpiComparison.linkedIdentitiesRatePercentagePoints != null
			? {
					delta: formatPercentagePointDelta(resolutionKpiComparison.linkedIdentitiesRatePercentagePoints),
					referenceDate: resolutionComparisonDate,
				}
			: undefined;
	const shouldShowResolutionStructure = isInitialLoading || chartError != null || latestResolution != null;
	const resolutionDistributionTemporalLabel =
		latestResolutionUTCTimestamp == null ? '' : `Completed ${latestResolutionUTCTimestamp}`;
	const processedTypeDistribution =
		latestResolution == null
			? undefined
			: {
					total: Number(latestResolution.identities.total),
					recognized: Number(latestResolution.identities.recognized),
					anonymous: Number(latestResolution.identities.anonymous),
					temporalLabel: resolutionDistributionTemporalLabel,
				};
	const unifiedTypeDistribution =
		latestResolution == null
			? undefined
			: {
					total: Number(latestResolution.profiles.total),
					recognized: Number(latestResolution.profiles.recognized),
					anonymous: Number(latestResolution.profiles.anonymous),
					temporalLabel: resolutionDistributionTemporalLabel,
				};

	const onPresetChange = (preset: IdentityDashboardDatePreset) => {
		const end =
			latestIdentityMetric == null ? todayUTCDateKey() : instantToDateKey(latestIdentityMetric.observedAt);
		setSelectedDateRange(preset);
		setDisplayRange(displayRangeForPreset(preset, end));
	};

	const onIdentityConnectionChange = (connection: string) => {
		if (connection === selectedIdentityConnection) {
			loadMetrics();
			return;
		}
		setSelectedIdentityConnection(connection);
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
		<main className='identity-dashboard'>
			<div className='identity-dashboard__page-header'>
				<div>
					<h1>Identity dashboard</h1>
					<p>Understand the effectiveness of your identity resolution and your current identity state.</p>
				</div>
				<div className='identity-dashboard__page-actions'>
					<SegmentedDateRangeControl<IdentityDashboardDatePreset>
						accessibleLabel='Trend range'
						presets={DATE_RANGE_PRESETS}
						value={selectedDateRange}
						customRange={customDateRange}
						onPresetChange={onPresetChange}
						onCustomRangeChange={onCustomRangeChange}
						maxDate={dateKeyToPickerDate(today)}
						pickerAlignment='end'
					/>
				</div>
			</div>

			<section className='identity-dashboard__section'>
				{dashboardError != null && (
					<StateMessage
						variant='error'
						title='Dashboard data could not be loaded'
						description={dashboardError}
						compact={true}
					/>
				)}
				{shouldShowResolutionStructure && (
					<div className='identity-dashboard__kpi-grid'>
						<KpiCard
							title='Last successful identity resolution'
							info='The most recent successful identity resolution run, regardless of the selected date range.'
							value={latestResolutionUTCTimestamp == null ? '—' : latestResolutionUTCTimestamp}
							secondary={
								latestResolution == null ? (
									'—'
								) : (
									<SlTooltip className='identity-dashboard__tooltip' placement='top' hoist={true}>
										<div slot='content' className='identity-dashboard__local-time-tooltip'>
											<strong>Your local time</strong>
											<span>
												{formatResolutionLocalTimeZoneDetails(
													latestResolution.observedAt,
													browserTimeZone,
												)}
											</span>
										</div>
										<span className='identity-dashboard__local-time-trigger' tabIndex={0}>
											{formatResolutionLocalTimestamp(
												latestResolution.observedAt,
												browserTimeZone,
											)}
										</span>
									</SlTooltip>
								)
							}
							subtleBackground={true}
							loading={isInitialLoading}
						/>
						<KpiCard
							title='Unified profiles'
							value={latestResolution == null ? '—' : formatNumber(latestResolution.profiles.total)}
							comparison={unifiedProfilesComparison}
							isComparisonReloading={isRangeReloading}
							loading={isInitialLoading}
						/>
						<KpiCard
							title='Identities per profile'
							titleAccentColor={IDENTITIES_PER_PROFILE_COLOR}
							info='Average number of identities linked to each unified profile, based on the latest successful identity resolution run.'
							value={formatNullableRatio(latestResolution?.identitiesPerProfile ?? null)}
							comparison={identitiesPerProfileComparison}
							isComparisonReloading={isRangeReloading}
							loading={isInitialLoading}
						/>
						<KpiCard
							title='Identity link rate'
							titleAccentColor={IDENTITY_LINK_RATE_COLOR}
							info='Share of identities that are part of a unified profile with multiple identities, based on the latest successful identity resolution run.'
							value={latestResolution == null ? '—' : formatRate(latestResolution.linkedIdentitiesRate)}
							comparison={identityLinkRateComparison}
							isComparisonReloading={isRangeReloading}
							loading={isInitialLoading}
						/>
					</div>
				)}
				<div className='identity-dashboard__resolution-layout'>
					<TypeDistributionCard
						processed={processedTypeDistribution}
						unified={unifiedTypeDistribution}
						loading={isInitialLoading}
						error={chartError}
					/>
					{shouldShowResolutionStructure ? (
						<>
							{!isInitialLoading && chartError == null && !hasResolutionInPeriod ? (
								<ResolutionPeriodEmptyState isReloading={isRangeReloading} />
							) : (
								<ResolutionEffectivenessChart
									data={resolutionEffectivenessData}
									loading={isInitialLoading}
									isReloading={isRangeReloading}
									error={chartError}
								/>
							)}
							<ProfilesChart
								days={unifiedProfileHistoryData}
								loading={isInitialLoading}
								isReloading={isRangeReloading}
								error={chartError}
							/>
							<ProfileComposition
								profiles={latestResolution?.profiles.total ?? 0}
								composition={latestResolution?.composition}
								loading={isInitialLoading}
								error={chartError}
							/>
						</>
					) : (
						<NoResolutionState />
					)}
				</div>
			</section>

			<section className='identity-dashboard__section identity-dashboard__section--current-state'>
				<SectionHeading
					title='Current identity state'
					secondary={observedLabel == null ? undefined : <>(as of {observedLabel})</>}
					info='The latest available identity state for your workspace. Values are point-in-time, not cumulative.'
				/>
				<div className='identity-dashboard__kpi-grid'>
					<KpiCard
						title='Total identities'
						value={latestIdentityMetric == null ? '—' : formatNumber(latestIdentityMetric.total)}
						secondary={observedLabel == null ? undefined : <>As of {observedLabel}</>}
						subtleBackground={true}
						loading={isInitialLoading}
					/>
					<KpiCard
						title='Since last resolution'
						info='Net change in the total number of identities since the latest successful identity resolution run.'
						value={
							identitiesChangeSinceResolution == null ? (
								'—'
							) : (
								<>
									{formatSignedIntegerDelta(identitiesChangeSinceResolution)}{' '}
									<span className='identity-dashboard__kpi-value-unit'>identities</span>
								</>
							)
						}
						secondary={
							latestResolutionComparisonUTCTimestamp == null ? undefined : (
								<>Completed {latestResolutionComparisonUTCTimestamp}</>
							)
						}
						sparklinePoints={identitiesChangeSinceResolutionPoints}
						loading={isInitialLoading}
					/>
					<KpiCard
						title='Over the last 7 days'
						info='Percentage change in the total number of identities compared with exactly seven days earlier.'
						value={formatTrendPercent(sevenDayTrend.changePercent)}
						secondary={
							sevenDayTrend.changePercent == null
								? undefined
								: `from ${formatComparisonDate(sevenDayTrend.referenceDay, sevenDayTrend.currentDay)}`
						}
						sparklinePoints={latestIdentityMetric == null ? undefined : sevenDayTrend.points}
						loading={isInitialLoading}
					/>
					<KpiCard
						title='Over the last 30 days'
						info='Percentage change in the total number of identities compared with exactly 30 days earlier.'
						value={formatTrendPercent(thirtyDayTrend.changePercent)}
						secondary={
							thirtyDayTrend.changePercent == null
								? undefined
								: `from ${formatComparisonDate(thirtyDayTrend.referenceDay, thirtyDayTrend.currentDay)}`
						}
						sparklinePoints={latestIdentityMetric == null ? undefined : thirtyDayTrend.points}
						loading={isInitialLoading}
					/>
				</div>
				<div className='identity-dashboard__current-charts'>
					<IdentitiesChart
						days={identityChartDays}
						loading={isInitialLoading}
						isReloading={isIdentityHistoryReloading}
						error={selectedIdentityConnection !== loadedIdentityConnection ? dashboardError : chartError}
						connectionOptions={identityConnectionOptions}
						selectedConnection={selectedIdentityConnection}
						onConnectionChange={onIdentityConnectionChange}
					/>
					<ConnectionsChart
						data={connectionBars}
						totalIdentities={latestIdentityMetric?.total ?? null}
						observedLabel={connectionObservedLabel}
						loading={isInitialLoading}
						error={chartError}
					/>
				</div>
			</section>

			<HistorySection runs={resolutionRuns} loading={isInitialLoading} error={chartError} />
		</main>
	);
};

export default IdentityOverview;
