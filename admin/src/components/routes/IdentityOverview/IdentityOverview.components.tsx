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
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from 'recharts';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import type SlSelectElement from '@shoelace-style/shoelace/dist/components/select/select.component.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { formatNumber } from '../../../utils/formatNumber';
import {
	ConnectionBar,
	DELETED_CONNECTION_SCOPE,
	IdentityTrend,
	formatChartDate,
	formatSharePercent,
	sparklineDomain,
} from './IdentityOverview.helpers';

const CHART_COLOR = '#6062d0';
const CONNECTION_ANONYMOUS_COLOR = '#b9bae9';
const GRID_COLOR = '#e8e8ee';
const TREND_CHART_ANIMATION_DURATION = 450;
const CONNECTION_AXIS_WIDTH = 180;
const CONNECTION_AXIS_SHARE_OFFSET = 52;

interface InfoTooltipProps {
	content: string;
	label: string;
}

const InfoTooltip = ({ content, label }: InfoTooltipProps) => (
	<SlTooltip content={content} placement='top' hoist>
		<button className='identity-overview__info' type='button' aria-label={label}>
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
	<div className='identity-overview__section-heading'>
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
	<div className={`identity-overview__card${className ? ` ${className}` : ''}`}>
		{title && (
			<div className='identity-overview__card-heading'>
				<h3>{title}</h3>
				{temporalLabel && <span>{temporalLabel}</span>}
				{info && <InfoTooltip content={info} label={`About ${title}`} />}
				{headerAction && <div className='identity-overview__card-heading-action'>{headerAction}</div>}
			</div>
		)}
		{children}
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
		<div className='identity-overview__chart-tooltip' role='tooltip'>
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

interface RecognizedAnonymousHistoryTooltipProps {
	active?: boolean;
	label?: unknown;
	payload?: readonly RechartsTooltipEntry[];
}

const RecognizedAnonymousHistoryTooltip = ({ active, label, payload }: RecognizedAnonymousHistoryTooltipProps) => {
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

interface StateMessageProps {
	variant: 'empty' | 'error' | 'unavailable';
	title: string;
	description?: string;
	action?: ReactNode;
	compact?: boolean;
}

const StateMessage = ({ variant, title, description, action, compact }: StateMessageProps) => (
	<div
		className={`identity-overview__state identity-overview__state--${variant}${compact ? ' identity-overview__state--compact' : ''}`}
		role={variant === 'error' ? 'alert' : 'status'}
	>
		<SlIcon name={variant === 'error' ? 'exclamation-triangle' : 'info-circle'} />
		<div>
			<div className='identity-overview__state-title'>{title}</div>
			{description && <div className='identity-overview__state-description'>{description}</div>}
			{action}
		</div>
	</div>
);

const CardLoading = ({ chart = false }: { chart?: boolean }) => (
	<div
		className={`identity-overview__loading${chart ? ' identity-overview__loading--chart' : ''}`}
		aria-label='Loading'
	>
		{chart ? (
			<SlSpinner style={{ fontSize: '2rem', '--track-width': '4px' } as React.CSSProperties} />
		) : (
			<>
				<div className='identity-overview__skeleton identity-overview__skeleton--value' />
				<div className='identity-overview__skeleton identity-overview__skeleton--label' />
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
	<DashboardCard className={`identity-overview__kpi${subtleBackground ? ' identity-overview__kpi--subtle' : ''}`}>
		<div className='identity-overview__kpi-title'>
			<span className='identity-overview__kpi-title-label'>
				{titleAccentColor && (
					<span
						className='identity-overview__kpi-title-accent'
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
				<div className='identity-overview__kpi-body'>
					<div className='identity-overview__kpi-value'>{value}</div>
					{secondary != null && <div className='identity-overview__kpi-secondary'>{secondary}</div>}
					{comparison && (
						<div className='identity-overview__kpi-comparison'>
							<span className='identity-overview__kpi-comparison-delta'>{comparison.delta}</span>
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
		<div className='identity-overview__sparkline' aria-hidden='true'>
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
	<ul className='identity-overview__connection-legend' aria-label='Identity types'>
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
	<DashboardCard title={title} info={info} headerAction={headerAction} className='identity-overview__chart-card'>
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
			<div className='identity-overview__chart'>
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
		info='Daily end-state recognized and anonymous identities for the Trend range. Missing observations remain gaps.'
		headerAction={
			<SlSelect
				className='identity-overview__connection-select'
				size='small'
				value={selectedConnection}
				hoist
				aria-label='Connection'
				onSlChange={(event) => onConnectionChange(String((event.currentTarget as SlSelectElement).value))}
			>
				<SlOption className='identity-overview__connection-option' value=''>
					All connections
				</SlOption>
				{connectionOptions.map((connection) => (
					<SlOption
						className={`identity-overview__connection-option${
							connection.id === DELETED_CONNECTION_SCOPE
								? ' identity-overview__connection-option--deleted'
								: ''
						}`}
						key={connection.id}
						value={connection.id}
					>
						<span className='identity-overview__connection-option-name'>{connection.name}</span>
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
				className={`identity-overview__connection-axis-name${
					connection.connection === DELETED_CONNECTION_SCOPE
						? ' identity-overview__connection-axis-name--deleted'
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
				className={`identity-overview__connection-axis-share${
					connection.connection === DELETED_CONNECTION_SCOPE
						? ' identity-overview__connection-axis-share--deleted'
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
		info='Latest identity state grouped by live source connection, with deleted connections shown as an aggregate. Identities without a profile are a subset of the anonymous and recognized counts.'
		className='identity-overview__chart-card'
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
				className='identity-overview__chart identity-overview__connections-chart'
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
export { ConnectionsChart, IdentitiesChart, KpiCard, SectionHeading, StateMessage };
