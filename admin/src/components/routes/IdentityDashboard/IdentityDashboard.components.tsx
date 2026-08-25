import React, { ReactNode } from 'react';
import {
	Area,
	AreaChart,
	Bar,
	BarChart,
	CartesianGrid,
	LabelList,
	Legend,
	Line,
	LineChart,
	Cell,
	Pie,
	PieChart,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from 'recharts';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import type SlSelectElement from '@shoelace-style/shoelace/dist/components/select/select.component.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { Link } from '../../base/Link/Link';
import { formatNumber } from '../../../utils/formatNumber';
import {
	ConnectionBar,
	DELETED_CONNECTION_SCOPE,
	IdentityTrend,
	ProfileCompositionBucket,
	ResolutionEffectivenessPoint,
	UnifiedProfileHistoryPoint,
	buildProfileComposition,
	buildTypeDistribution,
	chartDomainTicks,
	formatChartDate,
	formatRatio,
	formatResolutionUTCTimestamp,
	formatRunDuration,
	formatSharePercent,
	identityLinkRateChartDomain,
	ratioChartDomain,
	sparklineDomain,
} from './IdentityDashboard.helpers';
import { IdentityResolutionComposition } from '../../../lib/api/types/metrics';
import { IdentityResolutionRun } from '../../../lib/api/types/workspace';

const CHART_COLOR = '#6062d0';
const CONNECTION_ANONYMOUS_COLOR = '#b9bae9';
const IDENTITY_LINK_RATE_COLOR = '#0EA5E9';
const IDENTITIES_PER_PROFILE_COLOR = '#0f7c82';
const UNIFIED_PROFILES_RECOGNIZED_COLOR = '#f2822e';
const UNIFIED_PROFILES_ANONYMOUS_COLOR = '#fce4d3';
const TYPE_DISTRIBUTION_COLORS = {
	recognized: '#5a5f6b',
	anonymous: '#d9dde3',
};
const GRID_COLOR = '#e8e8ee';
const TREND_CHART_ANIMATION_DURATION = 450;
const DASHBOARD_DONUT_INNER_RADIUS = 48;
const DASHBOARD_DONUT_OUTER_RADIUS = 68;
const CONNECTION_AXIS_WIDTH = 180;
const CONNECTION_AXIS_SHARE_OFFSET = 52;

interface InfoTooltipProps {
	content: string;
	label: string;
}

const InfoTooltip = ({ content, label }: InfoTooltipProps) => (
	<SlTooltip className='identity-dashboard__tooltip' content={content} placement='top' hoist>
		<button className='identity-dashboard__info' type='button' aria-label={label}>
			<SlIcon name='info-circle' />
		</button>
	</SlTooltip>
);

interface SectionHeadingProps {
	title: string;
	secondary?: ReactNode;
	info: string;
}

const SectionHeading = ({ title, secondary, info }: SectionHeadingProps) => (
	<div className='identity-dashboard__section-heading'>
		<h2>{title}</h2>
		{secondary && <span>{secondary}</span>}
		<InfoTooltip content={info} label={`About ${title}`} />
	</div>
);

interface DashboardCardProps {
	title?: string;
	temporalLabel?: string;
	info?: string;
	headerAction?: ReactNode;
	className?: string;
	children: ReactNode;
}

const DashboardCard = ({ title, temporalLabel, info, headerAction, className, children }: DashboardCardProps) => (
	<div className={`identity-dashboard__card${className ? ` ${className}` : ''}`}>
		{title && (
			<div className='identity-dashboard__card-heading'>
				<h3>{title}</h3>
				{temporalLabel && <span>{temporalLabel}</span>}
				{info && <InfoTooltip content={info} label={`About ${title}`} />}
				{headerAction && <div className='identity-dashboard__card-heading-action'>{headerAction}</div>}
			</div>
		)}
		{children}
	</div>
);

const DonutCenter = ({ value, label }: { value: ReactNode; label: string }) => (
	<div className='identity-dashboard__donut-center'>
		<strong>{value}</strong>
		<span>{label}</span>
	</div>
);

interface ChartTooltipRow {
	label: ReactNode;
	value: ReactNode;
}

interface ChartTooltipProps {
	active?: boolean;
	title?: ReactNode;
	rows: ChartTooltipRow[];
}

const ChartTooltip = ({ active, title, rows }: ChartTooltipProps) => {
	if (!active || rows.length === 0) return null;

	return (
		<div className='identity-dashboard__chart-tooltip' role='tooltip'>
			{title != null && <strong>{title}</strong>}
			<dl>
				{rows.map((row, index) => (
					<div key={index}>
						<dt>{row.label}</dt>
						<dd>{row.value}</dd>
					</div>
				))}
			</dl>
		</div>
	);
};

interface RechartsTooltipEntry {
	name?: unknown;
	value?: unknown;
	payload?: unknown;
}

interface TimeSeriesTooltipProps {
	active?: boolean;
	label?: unknown;
	payload?: readonly RechartsTooltipEntry[];
	formatValue: (value: number) => string;
}

const TimeSeriesTooltip = ({ active, label, payload, formatValue }: TimeSeriesTooltipProps) => (
	<ChartTooltip
		active={active}
		title={label == null ? undefined : formatChartDate(String(label))}
		rows={(payload ?? []).flatMap((entry) =>
			entry.value == null ? [] : [{ label: String(entry.name ?? ''), value: formatValue(Number(entry.value)) }],
		)}
	/>
);

const RecognizedAnonymousHistoryTooltip = ({ active, label, payload }: Omit<TimeSeriesTooltipProps, 'formatValue'>) => {
	const entries = (payload ?? []).flatMap((entry) =>
		entry.value == null ? [] : [{ label: String(entry.name ?? ''), value: Number(entry.value) }],
	);
	const total = entries.reduce((sum, entry) => sum + entry.value, 0);

	return (
		<ChartTooltip
			active={active}
			title={label == null ? undefined : formatChartDate(String(label))}
			rows={entries.map((entry) => ({
				label: entry.label,
				value: `${formatNumber(entry.value)} (${formatSharePercent(entry.value, total, 2)})`,
			}))}
		/>
	);
};

const DonutTooltip = ({ active, payload }: Omit<TimeSeriesTooltipProps, 'label' | 'formatValue'>) => {
	const entry = payload?.[0];
	const datum = entry?.payload as { label?: string } | undefined;
	const label = datum?.label ?? (entry?.name == null ? undefined : String(entry.name));
	return (
		<ChartTooltip
			active={active}
			title={label}
			rows={entry?.value == null ? [] : [{ label: 'Total', value: formatNumber(Number(entry.value)) }]}
		/>
	);
};

interface StateMessageProps {
	variant: 'empty' | 'error' | 'unavailable';
	title: string;
	description?: string;
	action?: ReactNode;
	compact?: boolean;
}

const StateMessage = ({ variant, title, description, action, compact }: StateMessageProps) => (
	<div
		className={`identity-dashboard__state${variant === 'error' ? ' identity-dashboard__state--error' : ''}${compact ? ' identity-dashboard__state--compact' : ''}`}
		role={variant === 'error' ? 'alert' : 'status'}
	>
		<SlIcon name={variant === 'error' ? 'exclamation-triangle' : 'info-circle'} />
		<div>
			<div className='identity-dashboard__state-title'>{title}</div>
			{description && <div className='identity-dashboard__state-description'>{description}</div>}
			{action}
		</div>
	</div>
);

const CardLoading = ({ chart = false }: { chart?: boolean }) => (
	<div
		className={`identity-dashboard__loading${chart ? ' identity-dashboard__loading--chart' : ''}`}
		aria-label='Loading'
	>
		{chart ? (
			<SlSpinner style={{ fontSize: '2rem', '--track-width': '4px' } as React.CSSProperties} />
		) : (
			<>
				<div className='identity-dashboard__skeleton identity-dashboard__skeleton--value' />
				<div className='identity-dashboard__skeleton identity-dashboard__skeleton--label' />
			</>
		)}
	</div>
);

interface KpiCardProps {
	title: string;
	titleAccentColor?: string;
	value: ReactNode;
	secondary?: ReactNode;
	loading: boolean;
	sparklinePoints?: IdentityTrend['points'];
	subtleBackground?: boolean;
	info?: string;
	comparison?: {
		delta: string;
		referenceDate: string;
	};
}

const KpiCard = ({
	title,
	titleAccentColor,
	value,
	secondary,
	loading,
	sparklinePoints,
	subtleBackground,
	info,
	comparison,
}: KpiCardProps) => (
	<DashboardCard className={`identity-dashboard__kpi${subtleBackground ? ' identity-dashboard__kpi--subtle' : ''}`}>
		<div className='identity-dashboard__kpi-title'>
			<span className='identity-dashboard__kpi-title-label'>
				{titleAccentColor && (
					<span
						className='identity-dashboard__kpi-title-accent'
						style={{ backgroundColor: titleAccentColor }}
						aria-hidden='true'
					/>
				)}
				<span>{title}</span>
			</span>
			{info && <InfoTooltip content={info} label={`About ${title}`} />}
		</div>
		{loading ? (
			<CardLoading />
		) : (
			<>
				<div>
					<div className='identity-dashboard__kpi-value'>{value}</div>
					{secondary != null && <div className='identity-dashboard__kpi-secondary'>{secondary}</div>}
					{comparison && (
						<div className='identity-dashboard__kpi-comparison'>
							<span>{comparison.delta}</span>
							<span>from {comparison.referenceDate}</span>
						</div>
					)}
				</div>
				{sparklinePoints != null && sparklinePoints.length > 1 && <TrendSparkline points={sparklinePoints} />}
			</>
		)}
	</DashboardCard>
);

const TrendSparkline = ({ points }: { points: IdentityTrend['points'] }) => {
	const domain = sparklineDomain(points);
	return (
		<div className='identity-dashboard__sparkline' aria-hidden='true'>
			<ResponsiveContainer width='100%' height='100%'>
				<LineChart data={points}>
					<XAxis dataKey='day' hide />
					<YAxis hide domain={domain} />
					<Line
						type='linear'
						dataKey='total'
						stroke={CHART_COLOR}
						strokeWidth={2}
						dot={false}
						connectNulls={false}
						isAnimationActive={false}
					/>
				</LineChart>
			</ResponsiveContainer>
		</div>
	);
};

const compactNumber = (value: number): string =>
	new Intl.NumberFormat('en-US', { notation: 'compact', compactDisplay: 'short', maximumFractionDigits: 1 }).format(
		value,
	);

interface RecognizedAnonymousHistoryPoint {
	day: string;
	recognized: number | null;
	anonymous: number | null;
}

interface RecognizedAnonymousLegendProps {
	recognizedColor: string;
	anonymousColor: string;
}

const RecognizedAnonymousLegend = ({ recognizedColor, anonymousColor }: RecognizedAnonymousLegendProps) => (
	<ul className='identity-dashboard__connection-legend' aria-label='Identity types'>
		<li>
			<span style={{ background: recognizedColor }} />
			Recognized
		</li>
		<li>
			<span style={{ background: anonymousColor }} />
			Anonymous
		</li>
	</ul>
);

interface RecognizedAnonymousHistoryChartProps {
	title: string;
	info: string;
	headerAction?: ReactNode;
	days: RecognizedAnonymousHistoryPoint[];
	recognizedColor: string;
	anonymousColor: string;
	loading: boolean;
	error?: string;
	errorTitle: string;
	emptyTitle: string;
}

const RecognizedAnonymousHistoryChart = ({
	title,
	info,
	headerAction,
	days,
	recognizedColor,
	anonymousColor,
	loading,
	error,
	errorTitle,
	emptyTitle,
}: RecognizedAnonymousHistoryChartProps) => (
	<DashboardCard title={title} info={info} headerAction={headerAction} className='identity-dashboard__chart-card'>
		{loading ? (
			<CardLoading chart />
		) : error ? (
			<StateMessage variant='error' title={errorTitle} description={error} compact />
		) : days.every((day) => day.recognized == null && day.anonymous == null) ? (
			<StateMessage
				variant='empty'
				title={emptyTitle}
				description='Choose another start date or refresh the dashboard.'
				compact
			/>
		) : (
			<div className='identity-dashboard__chart'>
				<ResponsiveContainer width='100%' height='100%'>
					<AreaChart data={days} margin={{ top: 8, right: 18, bottom: 2, left: 4 }}>
						<CartesianGrid stroke={GRID_COLOR} vertical={false} />
						<XAxis dataKey='day' tickFormatter={formatChartDate} minTickGap={28} tickLine={false} />
						<YAxis tickFormatter={compactNumber} allowDecimals={false} tickLine={false} width={54} />
						<Tooltip
							content={({ active, label, payload }) => (
								<RecognizedAnonymousHistoryTooltip active={active} label={label} payload={payload} />
							)}
						/>
						<Legend
							content={
								<RecognizedAnonymousLegend
									recognizedColor={recognizedColor}
									anonymousColor={anonymousColor}
								/>
							}
						/>
						<Area
							type='linear'
							dataKey='recognized'
							name='Recognized'
							stackId='identity-types'
							stroke='none'
							fill={recognizedColor}
							fillOpacity={1}
							dot={false}
							activeDot={{ r: 4, fill: recognizedColor, stroke: 'none' }}
							connectNulls={false}
							isAnimationActive
							animationDuration={TREND_CHART_ANIMATION_DURATION}
							animationEasing='ease-out'
						/>
						<Area
							type='linear'
							dataKey='anonymous'
							name='Anonymous'
							stackId='identity-types'
							stroke='none'
							fill={anonymousColor}
							fillOpacity={1}
							dot={false}
							activeDot={{ r: 4, fill: anonymousColor, stroke: 'none' }}
							connectNulls={false}
							isAnimationActive
							animationDuration={TREND_CHART_ANIMATION_DURATION}
							animationEasing='ease-out'
						/>
					</AreaChart>
				</ResponsiveContainer>
			</div>
		)}
	</DashboardCard>
);

interface IdentitiesChartProps {
	days: RecognizedAnonymousHistoryPoint[];
	loading: boolean;
	error?: string;
	connectionOptions: { id: string; name: string }[];
	selectedConnection: string;
	onConnectionChange: (connection: string) => void;
}

const IdentitiesChart = ({
	days,
	loading,
	error,
	connectionOptions,
	selectedConnection,
	onConnectionChange,
}: IdentitiesChartProps) => (
	<RecognizedAnonymousHistoryChart
		title='Identities over time (daily)'
		info='Counts of recognized and anonymous identities at the end of each UTC day over the selected date range. Days with no data are shown as gaps.'
		headerAction={
			<SlSelect
				className='identity-dashboard__connection-select'
				size='small'
				value={selectedConnection}
				hoist
				aria-label='Connection'
				onSlChange={(event) => onConnectionChange(String((event.currentTarget as SlSelectElement).value))}
			>
				<SlOption className='identity-dashboard__connection-option' value=''>
					All connections
				</SlOption>
				{connectionOptions.map((connection) => (
					<SlOption
						className={`identity-dashboard__connection-option${
							connection.id === DELETED_CONNECTION_SCOPE
								? ' identity-dashboard__connection-option--deleted'
								: ''
						}`}
						key={connection.id}
						value={connection.id}
					>
						<span className='identity-dashboard__connection-option-name'>{connection.name}</span>
					</SlOption>
				))}
			</SlSelect>
		}
		days={days}
		recognizedColor={CHART_COLOR}
		anonymousColor={CONNECTION_ANONYMOUS_COLOR}
		loading={loading}
		error={error}
		errorTitle='Identity metrics could not be loaded'
		emptyTitle='No identity data in this period'
	/>
);

interface ProfilesChartProps {
	days: UnifiedProfileHistoryPoint[];
	loading: boolean;
	error?: string;
}

const ProfilesChart = ({ days, loading, error }: ProfilesChartProps) => (
	<RecognizedAnonymousHistoryChart
		title='Profiles over time (daily)'
		info='Daily end-of-day counts of recognized and anonymous unified profiles over the selected date range. Values carry forward until the next successful identity resolution run; days before the first available data point are shown as gaps.'
		days={days}
		recognizedColor={UNIFIED_PROFILES_RECOGNIZED_COLOR}
		anonymousColor={UNIFIED_PROFILES_ANONYMOUS_COLOR}
		loading={loading}
		error={error}
		errorTitle='Unified profile metrics could not be loaded'
		emptyTitle='No unified profile data in this period'
	/>
);

interface ConnectionsChartProps {
	data: ConnectionBar[];
	totalIdentities: number | null;
	observedLabel?: string;
	loading: boolean;
	error?: string;
}

interface ConnectionTooltipProps {
	active?: boolean;
	connection?: ConnectionBar;
}

const ConnectionTooltip = ({ active, connection }: ConnectionTooltipProps) => {
	if (!active || connection == null) return null;

	return (
		<ChartTooltip
			active
			title={connection.name}
			rows={[
				{ label: 'Total', value: formatNumber(connection.total) },
				{
					label: 'Recognized',
					value: `${formatNumber(connection.recognized)} (${formatSharePercent(connection.recognized, connection.total, 2)})`,
				},
				{
					label: 'Anonymous',
					value: `${formatNumber(connection.anonymous)} (${formatSharePercent(connection.anonymous, connection.total, 2)})`,
				},
			]}
		/>
	);
};

interface ConnectionAxisTickProps {
	x?: number;
	y?: number;
	index?: number;
	data: ConnectionBar[];
	totalIdentities: number | null;
}

const ConnectionAxisTick = ({ x = 0, y = 0, index = 0, data, totalIdentities }: ConnectionAxisTickProps) => {
	const connection = data[index];
	if (connection == null) return <g />;

	return (
		<g transform={`translate(0 ${y})`}>
			<text
				className={`identity-dashboard__connection-axis-name${
					connection.connection === DELETED_CONNECTION_SCOPE
						? ' identity-dashboard__connection-axis-name--deleted'
						: ''
				}`}
				x={x - CONNECTION_AXIS_SHARE_OFFSET}
				y={0}
				dominantBaseline='central'
				textAnchor='end'
			>
				{connection.name}
			</text>
			<text
				className={`identity-dashboard__connection-axis-share${
					connection.connection === DELETED_CONNECTION_SCOPE
						? ' identity-dashboard__connection-axis-share--deleted'
						: ''
				}`}
				x={x - CONNECTION_AXIS_SHARE_OFFSET / 2}
				y={0}
				dominantBaseline='central'
				textAnchor='middle'
			>
				{totalIdentities == null ? '—' : formatSharePercent(connection.total, totalIdentities, 0)}
			</text>
		</g>
	);
};

const ConnectionsChart = ({ data, totalIdentities, observedLabel, loading, error }: ConnectionsChartProps) => (
	<DashboardCard
		title='Identities by connection'
		temporalLabel={observedLabel == null ? undefined : `(as of ${observedLabel})`}
		info='Latest identity state grouped by active source connection. Identities from deleted connections are grouped under Other. Identities without a profile are included in the appropriate recognized or anonymous count.'
		className='identity-dashboard__chart-card'
	>
		{loading ? (
			<CardLoading chart />
		) : error ? (
			<StateMessage variant='error' title='Connection metrics could not be loaded' description={error} compact />
		) : data.length === 0 ? (
			<StateMessage
				variant='empty'
				title='No connections with identities'
				description='Connection contributions will appear after identities are imported.'
				compact
			/>
		) : (
			<div
				className='identity-dashboard__chart identity-dashboard__connections-chart'
				style={{ height: 258 + (data.length > 1 ? data.length * 2 + 2 : 0) }}
			>
				<ResponsiveContainer width='100%' height='100%'>
					<BarChart data={data} layout='vertical' margin={{ top: 8, right: 58, bottom: 2, left: 4 }}>
						<CartesianGrid stroke={GRID_COLOR} horizontal={false} />
						<XAxis type='number' tickFormatter={compactNumber} allowDecimals={false} tickLine={false} />
						<YAxis
							type='category'
							dataKey='name'
							width={CONNECTION_AXIS_WIDTH}
							tickLine={false}
							tickSize={0}
							tickMargin={0}
							interval={0}
							tick={<ConnectionAxisTick data={data} totalIdentities={totalIdentities} />}
						/>
						<Tooltip
							cursor={{ fill: '#f6f6f8' }}
							content={({ active, payload }) => (
								<ConnectionTooltip
									active={active}
									connection={payload?.[0]?.payload as ConnectionBar | undefined}
								/>
							)}
						/>
						<Legend
							content={
								<RecognizedAnonymousLegend
									recognizedColor={CHART_COLOR}
									anonymousColor={CONNECTION_ANONYMOUS_COLOR}
								/>
							}
						/>
						<Bar
							dataKey='recognized'
							name='Recognized'
							fill={CHART_COLOR}
							stackId='identity-types'
							barSize={17}
							isAnimationActive={false}
						/>
						<Bar
							dataKey='anonymous'
							name='Anonymous'
							fill={CONNECTION_ANONYMOUS_COLOR}
							stackId='identity-types'
							barSize={17}
							radius={[0, 4, 4, 0]}
							isAnimationActive={false}
						>
							<LabelList
								dataKey='total'
								position='right'
								formatter={(value) => formatNumber(Number(value))}
							/>
						</Bar>
					</BarChart>
				</ResponsiveContainer>
			</div>
		)}
	</DashboardCard>
);

interface ResolutionEffectivenessChartProps {
	data: ResolutionEffectivenessPoint[];
	loading: boolean;
	error?: string;
}

const ResolutionEffectivenessChart = ({ data, loading, error }: ResolutionEffectivenessChartProps) => {
	const hasIdentityLinkRate = data.some((day) => day.linkedIdentitiesRatePercent != null);
	const hasIdentitiesPerProfile = data.some((day) => day.identitiesPerProfile != null);
	const hasData = hasIdentityLinkRate || hasIdentitiesPerProfile;
	const identityLinkRateDomain = identityLinkRateChartDomain(data);
	const ratioDomain = ratioChartDomain(data);
	const identityLinkRateTicks = chartDomainTicks(identityLinkRateDomain);
	const ratioTicks = chartDomainTicks(ratioDomain);

	return (
		<DashboardCard
			title='Resolution effectiveness over time (daily)'
			className='identity-dashboard__chart-card identity-dashboard__resolution-chart-card'
		>
			{loading ? (
				<CardLoading chart />
			) : error ? (
				<StateMessage
					variant='error'
					title='Resolution metrics could not be loaded'
					description={error}
					compact
				/>
			) : !hasData ? (
				<StateMessage
					variant='empty'
					title='No identity resolution data in this period'
					description='Run identity resolution to populate the daily series.'
					compact
				/>
			) : (
				<div className='identity-dashboard__effectiveness-content'>
					<div className='identity-dashboard__effectiveness-metric'>
						<div className='identity-dashboard__effectiveness-metric-heading'>
							<h4>Identities per profile</h4>
							<InfoTooltip
								content='Average number of identities per unified profile over the selected date range. Each day shows the value as of the latest successful identity resolution run.'
								label='About identities per profile'
							/>
						</div>
						{hasIdentitiesPerProfile ? (
							<div className='identity-dashboard__effectiveness-chart'>
								<ResponsiveContainer width='100%' height='100%'>
									<LineChart
										data={data}
										margin={{ top: 10, right: 12, bottom: 10, left: 0 }}
										syncId='identity-resolution-effectiveness'
										syncMethod='value'
									>
										<CartesianGrid stroke={GRID_COLOR} vertical={false} />
										<XAxis dataKey='day' hide />
										<YAxis
											domain={ratioDomain}
											ticks={ratioTicks}
											tickFormatter={(value) => formatRatio(Number(value))}
											tickLine={false}
											axisLine={{ stroke: GRID_COLOR }}
											interval={0}
											width={42}
										/>
										<Tooltip
											content={({ active, label, payload }) => (
												<TimeSeriesTooltip
													active={active}
													label={label}
													payload={payload}
													formatValue={formatRatio}
												/>
											)}
										/>
										<Line
											type='stepAfter'
											dataKey='identitiesPerProfile'
											name='Identities per profile (ratio)'
											stroke={IDENTITIES_PER_PROFILE_COLOR}
											strokeWidth={2}
											dot={false}
											activeDot={{ r: 3 }}
											connectNulls={false}
											isAnimationActive
											animationDuration={TREND_CHART_ANIMATION_DURATION}
											animationEasing='ease-out'
										/>
									</LineChart>
								</ResponsiveContainer>
							</div>
						) : (
							<div className='identity-dashboard__effectiveness-empty'>No data available</div>
						)}
					</div>
					<div className='identity-dashboard__effectiveness-metric'>
						<div className='identity-dashboard__effectiveness-metric-heading'>
							<h4>Identity link rate</h4>
							<InfoTooltip
								content='Share of identities that are part of a unified profile with multiple identities over the selected date range. Each day shows the value as of the latest successful identity resolution run.'
								label='About identity link rate'
							/>
						</div>
						{hasIdentityLinkRate ? (
							<div className='identity-dashboard__effectiveness-chart'>
								<ResponsiveContainer width='100%' height='100%'>
									<LineChart
										data={data}
										margin={{ top: 10, right: 12, bottom: 2, left: 0 }}
										syncId='identity-resolution-effectiveness'
										syncMethod='value'
									>
										<CartesianGrid stroke={GRID_COLOR} vertical={false} />
										<XAxis
											dataKey='day'
											tickFormatter={formatChartDate}
											minTickGap={28}
											tickLine={false}
											axisLine={{ stroke: GRID_COLOR }}
										/>
										<YAxis
											domain={identityLinkRateDomain}
											ticks={identityLinkRateTicks}
											tickFormatter={(value) => `${Number(value).toFixed(1)}%`}
											tickLine={false}
											axisLine={{ stroke: GRID_COLOR }}
											interval={0}
											width={42}
										/>
										<Tooltip
											content={({ active, label, payload }) => (
												<TimeSeriesTooltip
													active={active}
													label={label}
													payload={payload}
													formatValue={(value) => `${value.toFixed(1)}%`}
												/>
											)}
										/>
										<Line
											type='stepAfter'
											dataKey='linkedIdentitiesRatePercent'
											name='Identity link rate (%)'
											stroke={IDENTITY_LINK_RATE_COLOR}
											strokeWidth={2}
											dot={false}
											activeDot={{
												r: 3,
												fill: IDENTITY_LINK_RATE_COLOR,
												stroke: IDENTITY_LINK_RATE_COLOR,
											}}
											connectNulls={false}
											isAnimationActive
											animationDuration={TREND_CHART_ANIMATION_DURATION}
											animationEasing='ease-out'
										/>
									</LineChart>
								</ResponsiveContainer>
							</div>
						) : (
							<div className='identity-dashboard__effectiveness-empty'>No data available</div>
						)}
					</div>
				</div>
			)}
		</DashboardCard>
	);
};

const COMPOSITION_COLORS = ['#5557be', '#686ac7', '#7778cf', '#999ade', '#b9bae9', '#d9d9f2'];
interface TypeDistributionSnapshot {
	total: number;
	recognized: number;
	anonymous: number;
	temporalLabel: string;
}

interface TypeDistributionBlockProps {
	title: string;
	snapshot?: TypeDistributionSnapshot;
	loading: boolean;
	error?: string;
	recognizedColor?: string;
	anonymousColor?: string;
}

const distributionPercentage = (value: number | null): string => (value == null ? '—' : `${value.toFixed(1)}%`);

const TypeDistributionBlock = ({
	title,
	snapshot,
	loading,
	error,
	recognizedColor = TYPE_DISTRIBUTION_COLORS.recognized,
	anonymousColor = TYPE_DISTRIBUTION_COLORS.anonymous,
}: TypeDistributionBlockProps) => {
	const segments =
		snapshot == null ? [] : buildTypeDistribution(snapshot.recognized, snapshot.anonymous, snapshot.total);
	const total = snapshot?.total ?? 0;
	const hasData = snapshot != null && total > 0 && error == null;
	const colors = { recognized: recognizedColor, anonymous: anonymousColor };

	return (
		<div className='identity-dashboard__distribution-block'>
			<div className='identity-dashboard__distribution-heading'>
				<h4>{title}</h4>
				{snapshot && <p>{snapshot.temporalLabel}</p>}
			</div>
			{loading ? (
				<CardLoading chart />
			) : !hasData ? (
				<div className='identity-dashboard__distribution-empty' role='status'>
					<SlIcon name='info-circle' />
					<span>No data available</span>
				</div>
			) : (
				<div className='identity-dashboard__distribution-body'>
					<div className='identity-dashboard__donut-ring' aria-label={`${title} type distribution`}>
						<ResponsiveContainer width='100%' height='100%'>
							<PieChart>
								<Pie
									data={segments}
									dataKey='count'
									nameKey='label'
									innerRadius={DASHBOARD_DONUT_INNER_RADIUS}
									outerRadius={DASHBOARD_DONUT_OUTER_RADIUS}
									startAngle={90}
									endAngle={-270}
									stroke='none'
									isAnimationActive={false}
								>
									{segments.map((segment) => (
										<Cell key={segment.key} fill={colors[segment.key]} />
									))}
								</Pie>
								<Tooltip
									content={({ active, payload }) => (
										<DonutTooltip active={active} payload={payload} />
									)}
								/>
							</PieChart>
						</ResponsiveContainer>
						<DonutCenter value={formatNumber(total)} label='Total' />
					</div>
					<div className='identity-dashboard__distribution-legend'>
						{segments.map((segment) => (
							<div className='identity-dashboard__distribution-legend-item' key={segment.key}>
								<i style={{ background: colors[segment.key] }} />
								<div>
									<span>{segment.label}</span>
									<strong>{distributionPercentage(segment.percentage)}</strong>
									<small>{formatNumber(segment.count)}</small>
								</div>
							</div>
						))}
					</div>
				</div>
			)}
		</div>
	);
};

interface TypeDistributionCardProps {
	processed?: TypeDistributionSnapshot;
	unified?: TypeDistributionSnapshot;
	loading: boolean;
	error?: string;
}

const TypeDistributionCard = ({ processed, unified, loading, error }: TypeDistributionCardProps) => (
	<DashboardCard
		title='Distribution per type'
		info='Both charts show the recognized vs. anonymous distribution from the latest successful identity resolution run.'
		className='identity-dashboard__distribution-card'
	>
		<div className='identity-dashboard__distribution-panel'>
			<TypeDistributionBlock title='Identities processed' snapshot={processed} loading={loading} error={error} />
			<div className='identity-dashboard__distribution-divider' aria-hidden='true'>
				<span>↔</span>
			</div>
			<TypeDistributionBlock
				title='Unified profiles'
				snapshot={unified}
				loading={loading}
				error={error}
				recognizedColor={UNIFIED_PROFILES_RECOGNIZED_COLOR}
				anonymousColor={UNIFIED_PROFILES_ANONYMOUS_COLOR}
			/>
		</div>
		<div className='identity-dashboard__distribution-footer'>
			<span>
				<SlIcon name='people' />
				<strong>Recognized:</strong> has at least one recognized identity
			</span>
			<i aria-hidden='true' />
			<span>
				<SlIcon name='person' />
				<strong>Anonymous:</strong> only anonymous identities
			</span>
		</div>
	</DashboardCard>
);

interface ProfileCompositionProps {
	profiles: number;
	composition?: IdentityResolutionComposition;
	loading: boolean;
	error?: string;
}

const compositionPercentage = (bucket: ProfileCompositionBucket): string =>
	bucket.percentage == null ? '—' : `${bucket.percentage.toFixed(1)}%`;

const ProfileComposition = ({ profiles, composition, loading, error }: ProfileCompositionProps) => {
	const buckets = composition == null ? [] : buildProfileComposition(composition, profiles);
	const hasValues = buckets.some((bucket) => bucket.count > 0);
	return (
		<DashboardCard
			title='Profiles by number of identities'
			info='Shows how unified profiles are distributed by the number of identities they contain, based on the latest successful identity resolution run.'
			className='identity-dashboard__composition-card'
		>
			{loading ? (
				<CardLoading chart />
			) : error ? (
				<StateMessage variant='error' title='Composition could not be loaded' description={error} compact />
			) : composition == null ? (
				<StateMessage variant='unavailable' title='Profile composition is unavailable' compact />
			) : (
				<div className='identity-dashboard__composition-content'>
					<div className='identity-dashboard__donut-ring' aria-label='Profiles by number of identities'>
						{hasValues ? (
							<ResponsiveContainer width='100%' height='100%'>
								<PieChart>
									<Pie
										data={buckets}
										dataKey='count'
										nameKey='label'
										innerRadius={DASHBOARD_DONUT_INNER_RADIUS}
										outerRadius={DASHBOARD_DONUT_OUTER_RADIUS}
										stroke='none'
										isAnimationActive={false}
									>
										{buckets.map((bucket, index) => (
											<Cell key={bucket.key} fill={COMPOSITION_COLORS[index]} />
										))}
									</Pie>
									<Tooltip
										content={({ active, payload }) => (
											<DonutTooltip active={active} payload={payload} />
										)}
									/>
								</PieChart>
							</ResponsiveContainer>
						) : (
							<div className='identity-dashboard__composition-ring-placeholder' />
						)}
						<DonutCenter value={formatNumber(profiles)} label='Profiles' />
					</div>
					<div className='identity-dashboard__composition-legend'>
						{buckets.map((bucket, index) => (
							<div key={bucket.key}>
								<i style={{ background: COMPOSITION_COLORS[index] }} />
								<span>{bucket.label}</span>
								<strong>{compositionPercentage(bucket)}</strong>
							</div>
						))}
					</div>
				</div>
			)}
		</DashboardCard>
	);
};

const GoToRun = () => (
	<Link path='profile-unification/profiles'>
		<SlButton size='small' variant='default'>
			Go to Run
		</SlButton>
	</Link>
);

const ResolutionPeriodEmptyState = () => (
	<DashboardCard className='identity-dashboard__period-empty-card'>
		<StateMessage
			variant='empty'
			title='No identity resolution completed in this period'
			description='Run identity resolution to see results here.'
			action={<GoToRun />}
		/>
	</DashboardCard>
);

const NoResolutionState = () => (
	<DashboardCard className='identity-dashboard__no-resolution-card'>
		<StateMessage
			variant='empty'
			title='No identity resolution completed in this period'
			description='Run identity resolution to see results here.'
			action={<GoToRun />}
		/>
	</DashboardCard>
);

const HISTORY_COLUMNS: GridColumn[] = [
	{ name: 'Status' },
	{ name: 'Started at' },
	{ name: 'Completed at' },
	{ name: 'Duration' },
	{ name: 'Error' },
];

interface HistorySectionProps {
	runs: IdentityResolutionRun[];
	loading: boolean;
	error?: string;
}

const historyStatusBadge = (status: IdentityResolutionRun['status']): ReactNode => {
	const labels = { running: 'Running', successful: 'Successful', failed: 'Failed' } as const;
	const variants = { running: 'primary', successful: 'success', failed: 'danger' } as const;
	return (
		<SlBadge className='identity-dashboard__history-status' variant={variants[status]} pill>
			{labels[status]}
		</SlBadge>
	);
};

const historyRows = (runs: IdentityResolutionRun[]): GridRow[] =>
	runs.map((run) => ({
		key: run.id,
		cells: [
			historyStatusBadge(run.status),
			formatResolutionUTCTimestamp(run.startTime),
			run.endTime == null ? '—' : formatResolutionUTCTimestamp(run.endTime),
			formatRunDuration(run.startTime, run.endTime),
			run.error == null ? (
				'—'
			) : (
				<span className='identity-dashboard__history-error' title={run.error}>
					{run.error}
				</span>
			),
		],
	}));

const HistorySection = ({ runs, loading, error }: HistorySectionProps) => (
	<section className='identity-dashboard__section'>
		<SectionHeading
			title='Identity resolution history'
			info='Identity resolution runs, including those in progress, successful, and failed.'
		/>
		<DashboardCard className='identity-dashboard__history-card'>
			{error == null ? (
				<div className='identity-dashboard__history-grid'>
					<Grid
						columns={HISTORY_COLUMNS}
						rows={historyRows(runs)}
						gridColumnsWidths='minmax(120px, 0.7fr) minmax(210px, 1.2fr) minmax(210px, 1.2fr) minmax(110px, 0.6fr) minmax(260px, 1.8fr)'
						isLoading={loading}
						loadingText='Loading identity resolution history'
						noRowsMessage='No identity resolution runs yet'
					/>
				</div>
			) : (
				<StateMessage
					variant='error'
					title='Identity resolution run history could not be loaded'
					description={error}
				/>
			)}
		</DashboardCard>
	</section>
);

export {
	ConnectionsChart,
	HistorySection,
	IDENTITIES_PER_PROFILE_COLOR,
	IdentitiesChart,
	KpiCard,
	NoResolutionState,
	ProfileComposition,
	ResolutionEffectivenessChart,
	IDENTITY_LINK_RATE_COLOR,
	ResolutionPeriodEmptyState,
	SectionHeading,
	StateMessage,
	TypeDistributionCard,
	ProfilesChart,
};
