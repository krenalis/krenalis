import React, { useState, useEffect, useContext, useMemo, useRef, ReactNode } from 'react';
import './ConnectionOverview.css';
import 'react-date-range/dist/styles.css';
import 'react-date-range/dist/theme/default.css';
import Grid from '../../base/Grid/Grid';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { ComposedChart, Line, Bar, Legend, XAxis, Tooltip, YAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import Arrow from '../../base/Arrow/Arrow';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { DateRange } from 'react-date-range';
import { ActionError, ActionErrorsResponse } from '../../../lib/api/types/responses';
import { ActionMetrics } from '../../../lib/api/types/action';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';
import considerAsUTC from '../../../utils/considerUTC';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { formatNumber } from '../../../utils/formatNumber';

interface ActionMetricsPoint {
	time: string;
	passed: number;
	failed: number;
	total: number;
}

interface FunnelPoint {
	passed: number;
	failed: number;
}

type FunnelData = FunnelPoint[];

type metricsRange = 'last15Minutes' | 'last24Hours' | 'last7Days' | 'Custom';

type StepIdentifier = 'RECEIVE' | 'INPUT_VALIDATION' | 'FILTER' | 'TRANSFORMATION' | 'OUTPUT_VALIDATION' | 'FINALIZE';

const MINUTES_COUNT = 15;
const HOURS_COUNT = 24;
const DAYS_COUNT = 7;

const ERRORS_COLUMNS: GridColumn[] = [
	{ name: 'Action' },
	{ name: 'Step' },
	{ name: 'Count', alignment: 'center' },
	{ name: 'Last occurred' },
	{ name: 'Error' },
];

const STEP_NAMES: string[] = [
	'Receive',
	'Input validation',
	'Filter',
	'Transformation',
	'Output validation',
	'Finalize',
];

const STEP_IDENTIFIERS: StepIdentifier[] = [
	'RECEIVE',
	'INPUT_VALIDATION',
	'FILTER',
	'TRANSFORMATION',
	'OUTPUT_VALIDATION',
	'FINALIZE',
];

const STEP_NAME_BY_IDENTIFIER: Record<StepIdentifier, string> = {
	RECEIVE: 'Receive',
	INPUT_VALIDATION: 'Input validation',
	FILTER: 'Filter',
	TRANSFORMATION: 'Transformation',
	OUTPUT_VALIDATION: 'Output validation',
	FINALIZE: 'Finalize',
};

const ConnectionOverview = () => {
	const [userActionsMetrics, setUserActionsMetrics] = useState<ActionMetrics>();
	const [eventActionsMetrics, setEventActionsMetrics] = useState<ActionMetrics>();
	const [userActionsErrors, setUserActionsErrors] = useState<ActionError[]>([]);
	const [eventActionsErrors, setEventActionsErrors] = useState<ActionError[]>([]);
	const [funnelArrows, setFunnelArrows] = useState<ReactNode[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [selectedTarget, setSelectedTarget] = useState<'Users' | 'Events'>('Users');
	const [selectedMetricsRange, setSelectedMetricsRange] = useState<metricsRange>('last15Minutes');
	const [isCustomMetricsRangePickerOpen, setIsCustomMetricsRangePickerOpen] = useState<boolean>(false);
	const [customMetricsRange, setCustomMetricsRange] = useState([
		{
			startDate: new Date(),
			endDate: new Date(),
			key: 'selection',
		},
	]);

	const { api, handleError } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	const supportedTargets = useRef([]);
	const currentMetricsIntervalID = useRef<any>();

	const { userActionErrorRows, eventActionErrorRows } = useMemo(() => {
		return {
			userActionErrorRows: computeActionErrorRows(c, userActionsErrors),
			eventActionErrorRows: computeActionErrorRows(c, eventActionsErrors),
		};
	}, [userActionsErrors, eventActionsErrors]);

	const { userActionMetricsData, eventActionMetricsData } = useMemo(() => {
		return {
			userActionMetricsData: computeActionMetricsData(userActionsMetrics, selectedMetricsRange),
			eventActionMetricsData: computeActionMetricsData(eventActionsMetrics, selectedMetricsRange),
		};
	}, [userActionsMetrics, eventActionsMetrics]);

	const { userFunnelData, eventFunnelData } = useMemo(() => {
		return {
			userFunnelData: computeFunnelData(userActionsMetrics),
			eventFunnelData: computeFunnelData(eventActionsMetrics),
		};
	}, [userActionsMetrics, eventActionsMetrics]);

	const steps = useMemo(() => {
		let steps: StepIdentifier[] = [...STEP_IDENTIFIERS];
		switch (c.connector.type) {
			case 'App':
				if (c.role == 'Destination') {
					if (selectedTarget == 'Events') {
						steps = steps.filter((v) => v !== 'INPUT_VALIDATION'); // No Input Validation.
					}
				}
				break;
			case 'Database':
				steps = steps.filter((v) => v !== 'FILTER'); // No Filter.
				break;
			case 'FileStorage':
				if (c.role == 'Destination') {
					steps = ['RECEIVE', 'INPUT_VALIDATION', 'FINALIZE'];
				}
				break;
			case 'Mobile':
			case 'Server':
			case 'Website':
				if (selectedTarget == 'Users') {
					steps = steps.filter((v) => v !== 'INPUT_VALIDATION'); // No Input Validation.
				} else {
					steps = ['RECEIVE', 'FILTER', 'FINALIZE'];
				}
		}
		return steps;
	}, [c, selectedTarget]);

	useEffect(() => {
		let data: FunnelData;
		if (selectedTarget === 'Events') {
			if (eventFunnelData == null) {
				return;
			}
			data = eventFunnelData;
		} else {
			if (userFunnelData == null) {
				return;
			}
			data = userFunnelData;
		}
		const arrows: ReactNode[] = [];
		for (let [i, s] of steps.entries()) {
			const isFilterStep = s === 'FILTER';

			const identifierIndex = STEP_IDENTIFIERS.findIndex((identifier) => identifier === s);
			const passedData = data[identifierIndex].passed;
			const failedData = data[identifierIndex].failed;

			let forwardArrow = (
				<Arrow
					key={`forward-arrow-${i}`}
					start={`funnel-circle-passed-${i}`}
					end={i === steps.length - 1 ? 'funnel-circle-final' : `funnel-circle-passed-${i + 1}`}
					startAnchor='right'
					endAnchor='left'
					showHead={true}
					label={
						i === 5 ? null : (
							<div className='connection-overview__funnel-label connection-overview__funnel-label--passed'>
								{formatNumber(passedData)}
							</div>
						)
					}
				></Arrow>
			);
			let bottomArrow = (
				<Arrow
					key={`bottom-arrow-${i}`}
					start={`funnel-circle-passed-${i}`}
					end={`funnel-circle-failed-${i}`}
					startAnchor='bottom'
					endAnchor='top'
					showHead={true}
					path='grid'
					label={
						<div
							className={`connection-overview__funnel-label connection-overview__funnel-label--failed${isFilterStep ? ' connection-overview__funnel-label--discarded' : ''}`}
						>
							{formatNumber(failedData)}
						</div>
					}
				></Arrow>
			);
			arrows.push(
				<>
					{forwardArrow}
					{bottomArrow}
				</>,
			);
		}
		arrows.push(
			<Arrow
				start='funnel-circle-initial'
				end='funnel-circle-passed-0'
				startAnchor='right'
				endAnchor='left'
				showHead={true}
			></Arrow>,
		);
		setFunnelArrows(arrows);
	}, [isLoading, eventFunnelData, userFunnelData]);

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		const fetchData = async () => {
			let userActionsIds: number[] = [];
			let eventActionsIds: number[] = [];
			for (const action of c.actions) {
				if (action.target === 'Users') {
					userActionsIds.push(action.id);
				} else if (action.target === 'Events') {
					eventActionsIds.push(action.id);
				}
			}

			if (userActionsIds.length === 0 && eventActionsIds.length === 0) {
				stopLoading();
				return;
			}

			if (userActionsIds.length > 0) {
				supportedTargets.current.push('Users');
			}

			if (eventActionsIds.length > 0) {
				supportedTargets.current.push('Events');
			}

			let fetchMetrics: (...args) => Promise<ActionMetrics> = null;
			if (selectedMetricsRange === 'last15Minutes') {
				fetchMetrics = async (actionIds) =>
					await api.workspaces.actionMetricsPerMinute(MINUTES_COUNT, actionIds);
			} else if (selectedMetricsRange === 'last24Hours') {
				fetchMetrics = async (actionIds) => await api.workspaces.actionMetricsPerHour(HOURS_COUNT, actionIds);
			} else if (selectedMetricsRange === 'last7Days') {
				fetchMetrics = async (actionIds) => await api.workspaces.actionMetricsPerDay(DAYS_COUNT, actionIds);
			} else {
				// end date must be shifted by one day to retrieve the
				// metrics including the last selected day.
				const endDate = new Date(customMetricsRange[0].endDate);
				endDate.setDate(endDate.getDate() + 1);
				try {
					validateMetricsRangeDates(customMetricsRange[0].startDate, endDate);
				} catch (err) {
					// fallback to the default metrics range.
					setSelectedMetricsRange('last15Minutes');
					handleError(err);
					return;
				}
				fetchMetrics = async (actionIds) =>
					await api.workspaces.actionMetricsPerDate(customMetricsRange[0].startDate, endDate, actionIds);
			}

			let target = selectedTarget;
			if (userActionsIds.length === 0) {
				target = 'Events';
			}
			setSelectedTarget(target);

			let ids: number[] = [];
			if (target === 'Users') {
				ids = userActionsIds;
			} else if (target === 'Events') {
				ids = eventActionsIds;
			}

			let metrics: ActionMetrics;
			try {
				metrics = await fetchMetrics(ids);
			} catch (err) {
				handleError(err);
				stopLoading();
				return;
			}
			if (target === 'Users') {
				setUserActionsMetrics(metrics);
			} else {
				setEventActionsMetrics(metrics);
			}

			let errorRes: ActionErrorsResponse;
			try {
				errorRes = await api.workspaces.actionErrors(metrics.start, metrics.end, ids, 0, 50, null);
			} catch (err) {
				handleError(err);
				stopLoading();
				return;
			}
			if (target === 'Users') {
				setUserActionsErrors(errorRes.errors);
			} else {
				setEventActionsErrors(errorRes.errors);
			}

			if (isLoading) {
				stopLoading();
			}
		};

		if (currentMetricsIntervalID.current != null) {
			clearInterval(currentMetricsIntervalID.current);
		}

		currentMetricsIntervalID.current = setInterval(() => {
			fetchData();
		}, 5000);
		fetchData();

		return () => {
			clearInterval(currentMetricsIntervalID.current);
		};
	}, [c, selectedTarget, selectedMetricsRange, customMetricsRange]);

	useEffect(() => {
		const handleCustomRangePickerClick = (e) => {
			const isInRangePicker = e.target.closest('.connection-overview__tabs-date-range-picker') != null;
			if (!isInRangePicker) {
				const isInRangePickerSelector = e.target.closest('.connection-overview__tabs-date-range') != null;
				if (!isInRangePickerSelector) {
					setIsCustomMetricsRangePickerOpen(false);
				}
			}
		};
		window.addEventListener('click', handleCustomRangePickerClick);
		() => {
			window.removeEventListener('click', handleCustomRangePickerClick);
		};
	}, []);

	const onSelectUsers = () => {
		setSelectedTarget('Users');
	};

	const onSelectEvents = () => {
		setSelectedTarget('Events');
	};

	const onSelectLast15Minutes = () => {
		setSelectedMetricsRange('last15Minutes');
	};

	const onSelectLast24Hours = () => {
		setSelectedMetricsRange('last24Hours');
	};

	const onSelectLast7Days = () => {
		setSelectedMetricsRange('last7Days');
	};

	const onSelectCustom = () => {
		setIsCustomMetricsRangePickerOpen(!isCustomMetricsRangePickerOpen);
	};

	const onChangeCustomMetricsRange = (selection) => {
		// the dates must be considered UTC.
		selection[0].startDate = considerAsUTC(selection[0].startDate);
		selection[0].endDate = considerAsUTC(selection[0].endDate);
		setCustomMetricsRange(selection);
		setSelectedMetricsRange('Custom');
	};

	if (isLoading) {
		return (
			<div className='connection-overview--loading'>
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			</div>
		);
	}

	const hasBothTargets = supportedTargets.current.includes('Users') && supportedTargets.current.includes('Events');
	const isUsersSelected = selectedTarget === 'Users';
	let titleRange = '';
	if (selectedMetricsRange === 'last15Minutes') {
		titleRange = 'in the last 15 minutes';
	} else if (selectedMetricsRange === 'last24Hours') {
		titleRange = 'in the last 24 hours';
	} else if (selectedMetricsRange === 'last7Days') {
		titleRange = 'in the last 7 days';
	} else {
		titleRange = `between ${customMetricsRange[0].startDate.toLocaleDateString()} and ${customMetricsRange[0].endDate.toLocaleDateString()}`;
	}

	return (
		<div className='connection-overview'>
			<div className='connection-overview__title'>Metrics & Log</div>
			{supportedTargets.current.length > 0 ? (
				<>
					<div className='connection-overview__tabs'>
						<SlButtonGroup>
							<SlButton
								variant={selectedMetricsRange === 'last15Minutes' ? 'primary' : 'default'}
								onClick={onSelectLast15Minutes}
								size='small'
							>
								Last 15 minutes
							</SlButton>
							<SlButton
								variant={selectedMetricsRange === 'last24Hours' ? 'primary' : 'default'}
								onClick={onSelectLast24Hours}
								size='small'
							>
								Last 24 hours
							</SlButton>
							<SlButton
								variant={selectedMetricsRange === 'last7Days' ? 'primary' : 'default'}
								onClick={onSelectLast7Days}
								size='small'
							>
								Last 7 days
							</SlButton>
							<div className='connection-overview__tabs-date-range'>
								<SlButton
									variant={selectedMetricsRange === 'Custom' ? 'primary' : 'default'}
									onClick={onSelectCustom}
									size='small'
								>
									{selectedMetricsRange === 'Custom'
										? `${customMetricsRange[0].startDate.toLocaleDateString()} - ${customMetricsRange[0].endDate.toLocaleDateString()}`
										: 'Custom range'}
								</SlButton>
								<div
									className={`connection-overview__tabs-date-range-picker${isCustomMetricsRangePickerOpen ? ' connection-overview__tabs-date-range-picker--open' : ''}`}
								>
									<DateRange
										editableDateInputs={true}
										onChange={(item) => onChangeCustomMetricsRange([item.selection])}
										showSelectionPreview={true}
										moveRangeOnFirstSelection={false}
										months={2}
										ranges={customMetricsRange}
										direction='horizontal'
									/>
								</div>
							</div>
						</SlButtonGroup>
						{hasBothTargets && (
							<SlButtonGroup>
								<SlButton
									variant={isUsersSelected ? 'default' : 'primary'}
									onClick={onSelectEvents}
									size='small'
								>
									Events
								</SlButton>
								<SlButton
									variant={isUsersSelected ? 'primary' : 'default'}
									onClick={onSelectUsers}
									size='small'
								>
									Users
								</SlButton>
							</SlButtonGroup>
						)}
					</div>
					<div className='connection-overview__chart'>
						<div className='connection-overview__chart-heading'>
							{isUsersSelected ? 'Users' : 'Events'} {c.isSource ? 'ingestion' : 'exports'} {titleRange}
						</div>
						<ResponsiveContainer width='100%' height='100%'>
							<ComposedChart
								data={isUsersSelected ? userActionMetricsData : eventActionMetricsData}
								margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
							>
								<CartesianGrid strokeDasharray='3 3' />
								<XAxis dataKey='time' />
								<YAxis
									tickFormatter={(value) => {
										// abbreviate with letters (ex.
										// "K", "M") instead of showing
										// big numbers.
										return new Intl.NumberFormat('en-US', {
											notation: 'compact',
											compactDisplay: 'short',
										}).format(value);
									}}
								/>
								<Tooltip
									formatter={(value) => {
										return formatNumber(Number(value));
									}}
								/>
								<Legend />
								<Bar dataKey='passed' fill='#4f46e5' />
								<Bar dataKey='failed' fill='#cf3a3a' />
								<Line
									type='monotone'
									dataKey='total'
									stroke='#a1a1aa'
									strokeDasharray='7 7'
									dot={{ stroke: '#3f3f46', fill: '#3f3f46', strokeWidth: 0 }}
								/>
							</ComposedChart>
						</ResponsiveContainer>
					</div>
					<div className='connection-overview__funnel'>
						<div className='connection-overview__funnel-heading'>Pipeline</div>
						<div className='connection-overview__funnel-passed'>
							<div className='connection-overview__funnel-initial' id={`funnel-circle-initial`}>
								{isUsersSelected
									? formatNumber(userFunnelData[0].passed + userFunnelData[0].failed)
									: formatNumber(eventFunnelData[0].passed + eventFunnelData[0].failed)}
							</div>
							{Array.from(steps.entries()).map(([i, s]) => {
								return (
									<div className='connection-overview__funnel-step' key={`funnel-passed-${i}`}>
										<div className='connection-overview__funnel-title'>
											{c.isDestination && c.isApp && !isUsersSelected && s === 'FINALIZE'
												? 'Delivering'
												: STEP_NAME_BY_IDENTIFIER[s]}
										</div>
										<div
											className='connection-overview__funnel-circle'
											id={`funnel-circle-passed-${i}`}
										/>
									</div>
								);
							})}
							<div className='connection-overview__funnel-final' id={`funnel-circle-final`}>
								{isUsersSelected
									? formatNumber(userFunnelData[5].passed)
									: formatNumber(eventFunnelData[5].passed)}
							</div>
						</div>
						<div className='connection-overview__funnel-failed'>
							<div key='funnel-initial-empty' />
							{Array.from(steps.entries()).map(([i, _]) => {
								return (
									<div
										key={`funnel-failed-${i}`}
										className='connection-overview__funnel-circle'
										id={`funnel-circle-failed-${i}`}
									/>
								);
							})}
						</div>
						{funnelArrows}
					</div>
					<div className='connection-overview__errors'>
						<div className='connection-overview__errors-heading'>Error log {titleRange}</div>
						<Grid
							columns={ERRORS_COLUMNS}
							rows={isUsersSelected ? userActionErrorRows : eventActionErrorRows}
							noRowsMessage={`No errors have occurred ${titleRange}`}
						/>
					</div>
				</>
			) : (
				<div className='connection-overview__nothing-to-show'>
					Currently there is nothing to show for connection {c.name}
				</div>
			)}
		</div>
	);
};

const computeActionErrorRows = (connection: TransformedConnection, actionErrors: ActionError[]): GridRow[] => {
	if (actionErrors == null) {
		return null;
	}
	let actionErrorRows: GridRow[] = [];
	for (const error of actionErrors) {
		const row = {
			cells: [
				<Link path={`connections/${connection.id}/actions/edit/${error.action}`}>
					{connection.actions.find((a) => a.id == error.action)?.name}
				</Link>,
				STEP_NAMES[error.step],
				error.count,
				<RelativeTime date={error.lastOccurred} />,
				error.message,
			],
		};
		actionErrorRows.push(row);
	}
	return actionErrorRows;
};

const computeActionMetricsData = (actionMetrics: ActionMetrics, range: metricsRange): ActionMetricsPoint[] => {
	if (actionMetrics == null) {
		return null;
	}
	let points: ActionMetricsPoint[] = [];
	const timeLength = actionMetrics.passed.length;
	let counter = timeLength;
	for (let timeUnit = 0; timeUnit < timeLength; timeUnit++) {
		let failedTotal = 0;
		for (let i = 0; i < 6; i++) {
			if (i === 2) {
				// filtered must not be considered as failed.
				continue;
			}
			failedTotal += actionMetrics.failed[timeUnit][i];
		}
		let filteredTotal = actionMetrics.failed[timeUnit][2];
		let passedTotal = actionMetrics.passed[timeUnit][5];
		let total = failedTotal + filteredTotal + passedTotal;
		const d = new Date(actionMetrics.end.getTime());
		let time = '';
		if (range === 'last15Minutes') {
			d.setMinutes(d.getMinutes() - counter);
			time = `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`;
		} else if (range === 'last24Hours') {
			d.setHours(d.getHours() - counter);
			time = `${d.getHours()}:00`;
		} else {
			d.setDate(d.getDate() - counter);
			time = `${d.toLocaleDateString()}`;
		}
		points.push({
			time: `${time}`,
			passed: passedTotal,
			failed: failedTotal,
			total: total,
		});
		counter--;
	}
	return points;
};

const computeFunnelData = (actionMetrics: ActionMetrics): FunnelData => {
	if (actionMetrics == null) {
		return [
			{ passed: 0, failed: 0 },
			{ passed: 0, failed: 0 },
			{ passed: 0, failed: 0 },
			{ passed: 0, failed: 0 },
			{ passed: 0, failed: 0 },
			{ passed: 0, failed: 0 },
		];
	}
	const data = [];
	for (let i = 0; i < 6; i++) {
		let totalPassed = 0;
		let totalFailed = 0;
		for (const p of actionMetrics.passed) {
			totalPassed += p[i];
		}
		for (const f of actionMetrics.failed) {
			totalFailed += f[i];
		}
		data.push({
			passed: totalPassed,
			failed: totalFailed,
		});
	}
	return data as FunnelData;
};

const LOWER_DATE = new Date('1970-01-01');
const UPPER_DATE = new Date('2262-04-11');
const ONE_DAY = 24 * 60 * 60 * 1000;
const validateMetricsRangeDates = (start: Date, end: Date): void => {
	if (start < LOWER_DATE || start > UPPER_DATE || end < LOWER_DATE || end > UPPER_DATE) {
		throw new Error(
			`Dates must be in the range between ${LOWER_DATE.toLocaleDateString()} and ${UPPER_DATE.toLocaleDateString()}`,
		);
	}
	const difference = end.getTime() - start.getTime();
	if (difference < ONE_DAY) {
		throw new Error('Start date must be at least one day before the end date');
	}
};

export default ConnectionOverview;
