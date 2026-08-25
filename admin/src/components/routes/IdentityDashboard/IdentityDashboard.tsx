import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import './IdentityDashboard.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
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
	formatUTCTimestamp,
	hasResolutionDataInRange,
	instantToDateKey,
	pickerDateToDateKey,
	todayUTCDateKey,
} from './IdentityDashboard.helpers';
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
} from './IdentityDashboard.components';

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

interface LoadMetricsOptions {
	preserveData?: boolean;
}

const IdentityDashboard = () => {
	const { api, connections, selectedWorkspace, setTitle } = useContext(AppContext);
	const [displayRange, setDisplayRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [loadedIdentityDisplayRange, setLoadedIdentityDisplayRange] = useState<DisplayDateRange>(initialDisplayRange);
	const [loadedResolutionDisplayRange, setLoadedResolutionDisplayRange] =
		useState<DisplayDateRange>(initialDisplayRange);
	const [selectedDateRange, setSelectedDateRange] = useState<IdentityDashboardDatePreset | 'Custom'>(
		IDENTITY_DASHBOARD_DEFAULT_DATE_PRESET,
	);
	const [customDateRange, setCustomDateRange] = useState<SegmentedDateRangeSelection[]>(initialCustomRange);
	const [latestIdentityMetric, setLatestIdentityMetric] = useState<IdentityMetric>();
	const [identityMetricDays, setIdentityMetricDays] = useState<IdentityMetricDay[]>();
	const [selectedIdentityConnection, setSelectedIdentityConnection] = useState<string>('');
	const [connectionIdentityMetricDays, setConnectionIdentityMetricDays] = useState<IdentityMetricDay[]>();
	const [loadedConnectionDisplayRange, setLoadedConnectionDisplayRange] =
		useState<DisplayDateRange>(initialDisplayRange);
	const [connectionIdentityError, setConnectionIdentityError] = useState<string>();
	const [isConnectionIdentityLoading, setIsConnectionIdentityLoading] = useState<boolean>(false);
	const [latestResolutionMetric, setLatestResolutionMetric] = useState<IdentityResolutionMetric | null>();
	const [resolutionMetricDays, setResolutionMetricDays] = useState<IdentityResolutionMetricDay[]>();
	const [resolutionRuns, setResolutionRuns] = useState<IdentityResolutionRun[]>([]);
	const [identityError, setIdentityError] = useState<string>();
	const [resolutionError, setResolutionError] = useState<string>();
	const [resolutionRunsError, setResolutionRunsError] = useState<string>();
	const [isIdentityLoading, setIsIdentityLoading] = useState<boolean>(true);
	const [isResolutionLoading, setIsResolutionLoading] = useState<boolean>(true);
	const [isResolutionRunsLoading, setIsResolutionRunsLoading] = useState<boolean>(true);
	const [isRefreshing, setIsRefreshing] = useState<boolean>(false);
	const requestVersion = useRef<number>(0);
	const connectionRequestVersion = useRef<number>(0);
	const runsRequestVersion = useRef<number>(0);
	// Keep range changes anchored to one latest snapshot until the next refresh.
	const latestIdentityMetricRef = useRef<IdentityMetric>();
	const latestResolutionMetricRef = useRef<IdentityResolutionMetric | null>();
	const selectedIdentityConnectionRef = useRef<string>('');
	const previousWorkspace = useRef<string>(selectedWorkspace);
	const previousConnectionWorkspace = useRef<string>(selectedWorkspace);
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
		setTitle('Profile Unification / Dashboard');
	}, [setTitle]);

	useLayoutEffect(() => {
		if (previousConnectionWorkspace.current === selectedWorkspace) return;
		previousConnectionWorkspace.current = selectedWorkspace;
		connectionRequestVersion.current += 1;
		selectedIdentityConnectionRef.current = '';
		setSelectedIdentityConnection('');
		setConnectionIdentityMetricDays(undefined);
		setConnectionIdentityError(undefined);
		setIsConnectionIdentityLoading(false);
	}, [selectedWorkspace]);

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

	const loadConnectionIdentityMetrics = useCallback(
		async (latest: IdentityMetric, connection: string, requestedRange: DisplayDateRange) => {
			const version = ++connectionRequestVersion.current;
			setConnectionIdentityError(undefined);
			setIsConnectionIdentityLoading(true);
			try {
				const days = await fetchIdentityMetricDays(latest, requestedRange, connection);
				if (
					version !== connectionRequestVersion.current ||
					connection !== selectedIdentityConnectionRef.current
				)
					return;
				setLoadedConnectionDisplayRange(requestedRange);
				setConnectionIdentityMetricDays(days);
			} catch (err) {
				if (
					version !== connectionRequestVersion.current ||
					connection !== selectedIdentityConnectionRef.current
				)
					return;
				setLoadedConnectionDisplayRange(requestedRange);
				setConnectionIdentityMetricDays(undefined);
				setConnectionIdentityError(getErrorMessage(err));
			} finally {
				if (version === connectionRequestVersion.current) setIsConnectionIdentityLoading(false);
			}
		},
		[fetchIdentityMetricDays],
	);

	const loadMetrics = useCallback(
		async ({ preserveData = false }: LoadMetricsOptions = {}) => {
			const version = ++requestVersion.current;
			const requestedRange = displayRange;
			setIdentityError(undefined);
			setResolutionError(undefined);
			if (!preserveData) {
				setIsIdentityLoading(true);
				setIsResolutionLoading(true);
				setLatestIdentityMetric(undefined);
				setIdentityMetricDays(undefined);
				setLatestResolutionMetric(undefined);
				setResolutionMetricDays(undefined);
			}

			const loadIdentityMetrics = async () => {
				try {
					let latest = latestIdentityMetricRef.current;
					if (latest == null) {
						latest = await api.workspaces.latestIdentityMetric();
						if (version !== requestVersion.current) return;
						latestIdentityMetricRef.current = latest;
					}

					const connection = selectedIdentityConnectionRef.current;
					const [days] = await Promise.all([
						fetchIdentityMetricDays(latest, requestedRange),
						connection === ''
							? Promise.resolve()
							: loadConnectionIdentityMetrics(latest, connection, requestedRange),
					]);
					if (version !== requestVersion.current) return;
					setLoadedIdentityDisplayRange(requestedRange);
					setLatestIdentityMetric(latest);
					setIdentityMetricDays(days);
					setIsIdentityLoading(false);
				} catch (err) {
					if (version !== requestVersion.current) return;
					setLoadedIdentityDisplayRange(requestedRange);
					setLatestIdentityMetric(undefined);
					setIdentityMetricDays(undefined);
					setIdentityError(getErrorMessage(err));
					setIsIdentityLoading(false);
				}
			};
			const loadResolutionMetrics = async () => {
				try {
					let latest = latestResolutionMetricRef.current;
					if (latest === undefined) {
						latest = await api.workspaces.latestIdentityResolutionMetric();
						if (version !== requestVersion.current) return;
						latestResolutionMetricRef.current = latest;
					}

					const days = latest == null ? [] : await fetchResolutionMetricDays(latest, requestedRange);
					if (version !== requestVersion.current) return;
					setLoadedResolutionDisplayRange(requestedRange);
					setLatestResolutionMetric(latest);
					setResolutionMetricDays(days);
					setIsResolutionLoading(false);
				} catch (err) {
					if (version !== requestVersion.current) return;
					setLoadedResolutionDisplayRange(requestedRange);
					setLatestResolutionMetric(undefined);
					setResolutionMetricDays(undefined);
					setResolutionError(getErrorMessage(err));
					setIsResolutionLoading(false);
				}
			};

			await Promise.all([loadIdentityMetrics(), loadResolutionMetrics()]);
		},
		[api, displayRange, fetchIdentityMetricDays, fetchResolutionMetricDays, loadConnectionIdentityMetrics],
	);

	const loadResolutionRuns = useCallback(
		async (preserveData = false) => {
			const version = ++runsRequestVersion.current;
			setResolutionRunsError(undefined);
			setIsResolutionRunsLoading(true);
			if (!preserveData) setResolutionRuns([]);
			try {
				const response = await api.workspaces.identityResolutionRuns();
				if (version !== runsRequestVersion.current) return;
				setResolutionRuns(response.runs);
			} catch (err) {
				if (version !== runsRequestVersion.current) return;
				setResolutionRunsError(getErrorMessage(err));
				if (!preserveData) setResolutionRuns([]);
			} finally {
				if (version === runsRequestVersion.current) setIsResolutionRunsLoading(false);
			}
		},
		[api],
	);

	useEffect(() => {
		const workspaceChanged = previousWorkspace.current !== selectedWorkspace;
		const connectionCatalogChanged = previousSourceConnectionCatalogKey.current !== sourceConnectionCatalogKey;
		previousWorkspace.current = selectedWorkspace;
		previousSourceConnectionCatalogKey.current = sourceConnectionCatalogKey;
		if (workspaceChanged || connectionCatalogChanged) latestIdentityMetricRef.current = undefined;
		if (workspaceChanged) latestResolutionMetricRef.current = undefined;
		loadMetrics({ preserveData: !workspaceChanged });
	}, [loadMetrics, selectedWorkspace, sourceConnectionCatalogKey]);

	useEffect(() => {
		loadResolutionRuns(false);
	}, [loadResolutionRuns, selectedWorkspace]);

	useEffect(() => {
		if (latestIdentityMetric == null || selectedDateRange === 'Custom') return;
		const latestDay = instantToDateKey(latestIdentityMetric.observedAt);
		const nextRange = displayRangeForPreset(selectedDateRange, latestDay);
		setDisplayRange((current) =>
			current.start === nextRange.start && current.end === nextRange.end ? current : nextRange,
		);
	}, [latestIdentityMetric, selectedDateRange]);

	const connectionNames = useMemo(
		() =>
			new Map([
				...connections.map((connection) => [connection.id, connection.name] as const),
				[DELETED_CONNECTION_SCOPE, DELETED_CONNECTION_LABEL] as const,
			]),
		[connections],
	);
	const identityDays = useMemo(
		() => buildIdentityMetricChartDays(identityMetricDays ?? [], loadedIdentityDisplayRange),
		[identityMetricDays, loadedIdentityDisplayRange],
	);
	const selectedConnectionIdentityDays = useMemo(
		() => buildIdentityMetricChartDays(connectionIdentityMetricDays ?? [], loadedConnectionDisplayRange),
		[connectionIdentityMetricDays, loadedConnectionDisplayRange],
	);
	const showAllIdentityConnections = selectedIdentityConnection === '';
	const selectedIdentityChartDays =
		connectionIdentityMetricDays == null ? identityDays : selectedConnectionIdentityDays;
	const identityChartDays = showAllIdentityConnections ? identityDays : selectedIdentityChartDays;
	const identityChartLoading = showAllIdentityConnections
		? isIdentityLoading
		: isConnectionIdentityLoading && connectionIdentityMetricDays == null && identityMetricDays == null;
	const identityChartError = showAllIdentityConnections ? identityError : connectionIdentityError;
	const temporalSemantics = useMemo(
		() =>
			buildTemporalSemantics(
				loadedResolutionDisplayRange,
				latestIdentityMetric?.observedAt,
				latestResolutionMetric?.observedAt,
			),
		[latestIdentityMetric, latestResolutionMetric, loadedResolutionDisplayRange],
	);
	const fixedIdentityTrendEnd = temporalSemantics.latestIdentityDay ?? loadedIdentityDisplayRange.end;
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
		connectionRequestVersion.current += 1;
		selectedIdentityConnectionRef.current = '';
		setSelectedIdentityConnection('');
		setConnectionIdentityMetricDays(undefined);
		setConnectionIdentityError(undefined);
		setIsConnectionIdentityLoading(false);
	}, [identityConnectionOptions, selectedIdentityConnection]);
	const resolutionChartDays = useMemo(
		() => buildIdentityResolutionMetricChartDays(resolutionMetricDays ?? [], loadedResolutionDisplayRange),
		[resolutionMetricDays, loadedResolutionDisplayRange],
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
		() => hasResolutionDataInRange(resolutionMetricDays ?? [], loadedResolutionDisplayRange),
		[resolutionMetricDays, loadedResolutionDisplayRange],
	);
	const resolutionKpiComparison = useMemo(
		() =>
			calculateResolutionKpiComparison(
				latestResolutionMetric ?? null,
				resolutionMetricDays ?? [],
				loadedResolutionDisplayRange,
			),
		[latestResolutionMetric, resolutionMetricDays, loadedResolutionDisplayRange],
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
	const identitiesChangeSinceResolution = calculateIdentityChangeSinceResolution(
		latestIdentityMetric == null ? null : Number(latestIdentityMetric.total),
		latestResolution == null ? null : Number(latestResolution.identities.total),
	);
	const identitiesChangeSinceResolutionPoints = buildIdentityChangeSinceResolutionPoints(
		latestIdentityMetric?.observedAt ?? null,
		latestIdentityMetric == null ? null : Number(latestIdentityMetric.total),
		latestResolution?.observedAt ?? null,
		latestResolution == null ? null : Number(latestResolution.identities.total),
	);
	const resolutionComparisonDate = formatComparisonDate(
		resolutionKpiComparison.previousEnd,
		latestResolution == null ? resolutionKpiComparison.previousEnd : instantToDateKey(latestResolution.observedAt),
	);
	const browserTimeZone = new Intl.DateTimeFormat().resolvedOptions().timeZone;
	const showResolutionKpiComparison = temporalSemantics.showLatestResolutionComparison;
	const unifiedProfilesComparison = showResolutionKpiComparison
		? {
				delta: formatTrendPercent(resolutionKpiComparison.profilesChangePercent),
				referenceDate: resolutionComparisonDate,
			}
		: undefined;
	const identitiesPerProfileComparison = showResolutionKpiComparison
		? {
				delta: formatRatioDelta(resolutionKpiComparison.identitiesPerProfileChange),
				referenceDate: resolutionComparisonDate,
			}
		: undefined;
	const identityLinkRateComparison = showResolutionKpiComparison
		? {
				delta: formatPercentagePointDelta(resolutionKpiComparison.linkedIdentitiesRatePercentagePoints),
				referenceDate: resolutionComparisonDate,
			}
		: undefined;
	const showResolutionStructure = isResolutionLoading || resolutionError != null || latestResolution != null;
	const resolutionDistributionTemporalLabel =
		latestResolution == null ? '' : `Completed ${formatUTCTimestamp(latestResolution.observedAt)} UTC`;
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
		connectionRequestVersion.current += 1;
		selectedIdentityConnectionRef.current = connection;
		setSelectedIdentityConnection(connection);
		setConnectionIdentityError(undefined);
		if (connection === '') {
			setConnectionIdentityMetricDays(undefined);
			setIsConnectionIdentityLoading(false);
			return;
		}

		const latest = latestIdentityMetricRef.current;
		if (latest == null) {
			setIsConnectionIdentityLoading(true);
			return;
		}
		loadConnectionIdentityMetrics(latest, connection, displayRange);
	};

	const refreshDashboard = async () => {
		setIsRefreshing(true);
		setIdentityError(undefined);
		try {
			await api.workspaces.refreshIdentityMetrics();
			latestIdentityMetricRef.current = undefined;
			latestResolutionMetricRef.current = undefined;
			await Promise.all([loadMetrics({ preserveData: true }), loadResolutionRuns(true)]);
		} catch (err) {
			setIdentityError(getErrorMessage(err));
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
		<main className='identity-dashboard'>
			<div className='identity-dashboard__page-header'>
				<div>
					<h1>Identity dashboard</h1>
					<p>Understand the effectiveness of your identity resolution and your current identity state.</p>
				</div>
				<div className='identity-dashboard__page-actions'>
					<SlButton
						className='segmented-date-range-button'
						variant='default'
						size='small'
						onClick={refreshDashboard}
						loading={isRefreshing}
						disabled={isRefreshing || isIdentityLoading || isResolutionLoading}
					>
						<SlIcon slot='prefix' name='arrow-clockwise' />
						Refresh
					</SlButton>
					<SegmentedDateRangeControl<IdentityDashboardDatePreset>
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

			<section className='identity-dashboard__section'>
				{resolutionError && (
					<StateMessage
						variant='error'
						title='Identity resolution metrics are unavailable'
						description={resolutionError}
						compact={true}
					/>
				)}
				{showResolutionStructure && (
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
							loading={isResolutionLoading}
						/>
						<KpiCard
							title='Unified profiles'
							value={latestResolution == null ? '—' : formatNumber(latestResolution.profiles.total)}
							comparison={unifiedProfilesComparison}
							loading={isResolutionLoading}
						/>
						<KpiCard
							title='Identities per profile'
							titleAccentColor={IDENTITIES_PER_PROFILE_COLOR}
							info='Average number of identities linked to each unified profile, based on the latest successful identity resolution run.'
							value={formatNullableRatio(latestResolution?.identitiesPerProfile ?? null)}
							comparison={identitiesPerProfileComparison}
							loading={isResolutionLoading}
						/>
						<KpiCard
							title='Identity link rate'
							titleAccentColor={IDENTITY_LINK_RATE_COLOR}
							info='Share of identities that are part of a unified profile with multiple identities, based on the latest successful identity resolution run.'
							value={latestResolution == null ? '—' : formatRate(latestResolution.linkedIdentitiesRate)}
							comparison={identityLinkRateComparison}
							loading={isResolutionLoading}
						/>
					</div>
				)}
				<div className='identity-dashboard__resolution-layout'>
					<TypeDistributionCard
						processed={processedTypeDistribution}
						unified={unifiedTypeDistribution}
						loading={isResolutionLoading}
						error={resolutionError}
					/>
					{showResolutionStructure ? (
						<>
							{!isResolutionLoading && resolutionError == null && !hasResolutionInPeriod ? (
								<ResolutionPeriodEmptyState />
							) : (
								<ResolutionEffectivenessChart
									data={resolutionEffectivenessData}
									loading={isResolutionLoading}
									error={resolutionError}
								/>
							)}
							<ProfilesChart
								days={unifiedProfileHistoryData}
								loading={isResolutionLoading}
								error={resolutionError}
							/>
							<ProfileComposition
								profiles={latestResolution?.profiles.total ?? 0}
								composition={latestResolution?.composition}
								loading={isResolutionLoading}
								error={resolutionError}
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
				{identityError && (
					<StateMessage
						variant='error'
						title='Current identity state is unavailable'
						description={identityError}
						compact={true}
					/>
				)}
				<div className='identity-dashboard__kpi-grid'>
					<KpiCard
						title='Total identities'
						value={latestIdentityMetric == null ? '—' : formatNumber(latestIdentityMetric.total)}
						secondary={observedLabel == null ? undefined : <>As of {observedLabel}</>}
						subtleBackground={true}
						loading={isIdentityLoading}
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
						loading={isIdentityLoading || isResolutionLoading}
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
						loading={isIdentityLoading}
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
						loading={isIdentityLoading}
					/>
				</div>
				<div className='identity-dashboard__current-charts'>
					<IdentitiesChart
						days={identityChartDays}
						loading={identityChartLoading}
						error={identityChartError}
						connectionOptions={identityConnectionOptions}
						selectedConnection={selectedIdentityConnection}
						onConnectionChange={onIdentityConnectionChange}
					/>
					<ConnectionsChart
						data={connectionBars}
						totalIdentities={latestIdentityMetric?.total ?? null}
						observedLabel={connectionObservedLabel}
						loading={isIdentityLoading}
						error={identityError}
					/>
				</div>
			</section>

			<HistorySection runs={resolutionRuns} loading={isResolutionRunsLoading} error={resolutionRunsError} />
		</main>
	);
};

export default IdentityDashboard;
