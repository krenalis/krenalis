import React, { useState, useEffect, useContext, useMemo, useRef, ReactNode } from 'react';
import './ConnectionMetrics.css';
import 'react-date-range/dist/styles.css';
import 'react-date-range/dist/theme/default.css';
import Grid from '../../base/Grid/Grid';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { ComposedChart, Line, Bar, Legend, XAxis, Tooltip, YAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import Arrow from '../../base/Arrow/Arrow';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { DateRange } from 'react-date-range';
import { ActionError, ActionErrorsResponse } from '../../../lib/api/types/responses';
import { ActionMetrics, ActionTarget } from '../../../lib/api/types/action';
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
	{ name: 'Error', type: 'html' },
];

const STEP_IDENTIFIERS: StepIdentifier[] = [
	'RECEIVE',
	'INPUT_VALIDATION',
	'FILTER',
	'TRANSFORMATION',
	'OUTPUT_VALIDATION',
	'FINALIZE',
];

const ConnectionMetrics = () => {
	const { connection: c } = useContext(ConnectionContext);

	const [userActionsMetrics, setUserActionsMetrics] = useState<ActionMetrics>();
	const [eventActionsMetrics, setEventActionsMetrics] = useState<ActionMetrics>();
	const [userActionsErrors, setUserActionsErrors] = useState<ActionError[]>([]);
	const [eventActionsErrors, setEventActionsErrors] = useState<ActionError[]>([]);
	const [funnelArrows, setFunnelArrows] = useState<ReactNode[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [selectedTarget, setSelectedTarget] = useState<ActionTarget>(
		c.actions.findIndex((a) => a.target === 'Event') !== -1 ? 'Event' : 'User',
	);
	const [selectedMetricsRange, setSelectedMetricsRange] = useState<metricsRange>('last15Minutes');
	const [isCustomMetricsRangePickerOpen, setIsCustomMetricsRangePickerOpen] = useState<boolean>(false);
	const [customMetricsRange, setCustomMetricsRange] = useState([
		{
			startDate: new Date(),
			endDate: new Date(),
			key: 'selection',
		},
	]);
	const [selectedAction, setSelectedAction] = useState<number | null>(null);

	const { api, handleError } = useContext(AppContext);

	const supportedTargets = useRef([]);
	const currentMetricsIntervalID = useRef<any>();
	const previouslySelectedAction = useRef<number | null>(null);

	const isUsersSelected = selectedTarget === 'User';

	let receiveStepTerm = '';
	let finalizeStepTerm = '';
	if (c.isSource) {
		if (isUsersSelected) {
			receiveStepTerm = `Extract from ${c.name}`;
		} else {
			receiveStepTerm = `Receive from ${c.name}`;
		}
		finalizeStepTerm = 'Load into warehouse';
	} else {
		if (isUsersSelected) {
			receiveStepTerm = 'Extract from warehouse';
			finalizeStepTerm = `Load into ${c.name}`;
		} else {
			receiveStepTerm = 'Receive from sources';
			finalizeStepTerm = `Send to ${c.name}`;
		}
	}

	const stepTermByIdentifier: Record<StepIdentifier, string> = {
		RECEIVE: receiveStepTerm,
		INPUT_VALIDATION: 'Check user data',
		FILTER: 'Apply filter',
		TRANSFORMATION: 'Transform',
		OUTPUT_VALIDATION: 'Validate',
		FINALIZE: finalizeStepTerm,
	};

	const { userActionErrorRows, eventActionErrorRows } = useMemo(() => {
		const stepTerms = Object.values(stepTermByIdentifier);
		return {
			userActionErrorRows: computeActionErrorRows(c, userActionsErrors, stepTerms),
			eventActionErrorRows: computeActionErrorRows(c, eventActionsErrors, stepTerms),
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
					if (selectedTarget == 'Event') {
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
			case 'SDK':
				if (selectedTarget == 'User') {
					steps = steps.filter((v) => v !== 'INPUT_VALIDATION'); // No Input Validation.
				} else {
					steps = ['RECEIVE', 'FILTER', 'FINALIZE'];
				}
		}
		return steps;
	}, [c, selectedTarget]);

	useEffect(() => {
		let data: FunnelData;
		if (selectedTarget === 'Event') {
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
						i === steps.length - 1 ? null : (
							<div className='connection-metrics__funnel-label connection-metrics__funnel-label--passed'>
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
							className={`connection-metrics__funnel-label connection-metrics__funnel-label--failed${isFilterStep ? ' connection-metrics__funnel-label--discarded' : ''}`}
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
			if (selectedAction == null) {
				for (const action of c.actions) {
					if (action.target === 'User') {
						userActionsIds.push(action.id);
					} else if (action.target === 'Event') {
						eventActionsIds.push(action.id);
					}
				}
			} else {
				const a = c.actions.find((action) => action.id === selectedAction);
				if (a.target === 'User') {
					userActionsIds.push(a.id);
				} else if (a.target === 'Event') {
					eventActionsIds.push(a.id);
				}
			}

			if (userActionsIds.length === 0 && eventActionsIds.length === 0) {
				stopLoading();
				return;
			}

			if (userActionsIds.length > 0) {
				supportedTargets.current.push('User');
			}

			if (eventActionsIds.length > 0) {
				supportedTargets.current.push('Event');
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
			let ids: number[] = [];
			if (target === 'User') {
				ids = userActionsIds;
			} else if (target === 'Event') {
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
			if (target === 'User') {
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
			if (target === 'User') {
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
	}, [c, selectedTarget, selectedMetricsRange, customMetricsRange, selectedAction]);

	useEffect(() => {
		const handleCustomRangePickerClick = (e) => {
			const isInRangePicker = e.target.closest('.connection-metrics__tabs-date-range-picker') != null;
			if (!isInRangePicker) {
				const isInRangePickerSelector = e.target.closest('.connection-metrics__tabs-date-range') != null;
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

	const onChangeSelectedTarget = (target: ActionTarget) => {
		const isAlreadySelected = selectedTarget === target;
		if (isAlreadySelected) {
			return;
		}
		const toRestore = previouslySelectedAction.current;
		previouslySelectedAction.current = null;
		setSelectedTarget(target);
		if (selectedAction != null) {
			previouslySelectedAction.current = selectedAction;
		}
		setSelectedAction(toRestore);
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

	const onChangeSelectedAction = (e: any) => {
		const v = e.target.value;
		if (v === '') {
			setSelectedAction(null);
		} else {
			setSelectedAction(Number(v));
		}
	};

	if (isLoading) {
		return (
			<div className='connection-metrics--loading'>
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

	let titleRange = '';
	if (selectedMetricsRange === 'last15Minutes') {
		titleRange = 'Last 15 minutes';
	} else if (selectedMetricsRange === 'last24Hours') {
		titleRange = 'Last 24 hours';
	} else if (selectedMetricsRange === 'last7Days') {
		titleRange = 'Last 7 days';
	} else {
		titleRange = `Between ${customMetricsRange[0].startDate.toLocaleDateString()} and ${customMetricsRange[0].endDate.toLocaleDateString()}`;
	}

	let chartTitle = '';
	if (c.isSource) {
		chartTitle = 'Imported';
	} else {
		if (isUsersSelected) {
			chartTitle = 'Exported';
		} else {
			chartTitle = 'Sent';
		}
	}

	return (
		<div className='connection-metrics'>
			<div className='connection-metrics__title'>Metrics & Log</div>
			{supportedTargets.current.length > 0 ? (
				<>
					<div className='connection-metrics__tabs'>
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
							<div className='connection-metrics__tabs-date-range'>
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
									className={`connection-metrics__tabs-date-range-picker${isCustomMetricsRangePickerOpen ? ' connection-metrics__tabs-date-range-picker--open' : ''}`}
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
						<SlButtonGroup>
							<SlButton
								variant={isUsersSelected ? 'default' : 'primary'}
								onClick={
									supportedTargets.current.includes('Event')
										? () => onChangeSelectedTarget('Event')
										: null
								}
								size='small'
								disabled={!supportedTargets.current.includes('Event')}
							>
								Events
							</SlButton>
							<SlButton
								variant={isUsersSelected ? 'primary' : 'default'}
								onClick={
									supportedTargets.current.includes('User')
										? () => onChangeSelectedTarget('User')
										: null
								}
								size='small'
								disabled={!supportedTargets.current.includes('User')}
							>
								Users
							</SlButton>
						</SlButtonGroup>
						{c.actions?.length > 1 && !(c.isSDK && c.isSource && selectedTarget === 'Event') && (
							<SlSelect
								size='small'
								label='Action'
								onSlChange={onChangeSelectedAction}
								value={selectedAction == null ? '' : String(selectedAction)}
								className={`connection-metrics__actions${selectedAction != null ? ' connection-metrics__actions--filtered' : ''}`}
								clearable
							>
								{c.actions?.map((a) => {
									if (a.target == selectedTarget) {
										return <SlOption value={String(a.id)}>{a.name}</SlOption>;
									}
									return null;
								})}
							</SlSelect>
						)}
					</div>
					<div className='connection-metrics__chart'>
						<div className='connection-metrics__chart-heading'>
							{chartTitle} {isUsersSelected ? 'users' : 'events'} <span>{titleRange}</span>
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
									allowDecimals={false}
								/>
								<Tooltip
									formatter={(value) => {
										return formatNumber(Number(value));
									}}
								/>
								<Legend />
								<Bar dataKey='passed' name={chartTitle} fill='#4f46e5' />
								<Bar dataKey='failed' name='Failed' fill='#cf3a3a' />
								<Line
									type='monotone'
									dataKey='total'
									name='Total'
									stroke='#a1a1aa'
									strokeDasharray='7 7'
									dot={{ stroke: '#3f3f46', fill: '#3f3f46', strokeWidth: 0 }}
								/>
							</ComposedChart>
						</ResponsiveContainer>
					</div>
					<div className='connection-metrics__funnel'>
						<div className='connection-metrics__funnel-heading'>Pipeline</div>
						<div className='connection-metrics__funnel-passed'>
							<div className='connection-metrics__funnel-initial' id={`funnel-circle-initial`}>
								{isUsersSelected
									? formatNumber(userFunnelData[0].passed + userFunnelData[0].failed)
									: formatNumber(eventFunnelData[0].passed + eventFunnelData[0].failed)}
							</div>
							{Array.from(steps.entries()).map(([i, s]) => {
								return (
									<div className='connection-metrics__funnel-step' key={`funnel-passed-${i}`}>
										<div className='connection-metrics__funnel-title'>
											{stepTermByIdentifier[s]}
										</div>
										<div
											className='connection-metrics__funnel-circle'
											id={`funnel-circle-passed-${i}`}
										/>
									</div>
								);
							})}
							<div className='connection-metrics__funnel-final' id={`funnel-circle-final`}>
								{isUsersSelected
									? formatNumber(userFunnelData[5].passed)
									: formatNumber(eventFunnelData[5].passed)}
							</div>
						</div>
						<div className='connection-metrics__funnel-failed'>
							<div key='funnel-initial-empty' />
							{Array.from(steps.entries()).map(([i, _]) => {
								return (
									<div
										key={`funnel-failed-${i}`}
										className='connection-metrics__funnel-circle'
										id={`funnel-circle-failed-${i}`}
									/>
								);
							})}
						</div>
						{funnelArrows}
					</div>
					<div className='connection-metrics__errors'>
						<div className='connection-metrics__errors-heading'>
							Error log <span>{titleRange}</span>
						</div>
						<Grid
							columns={ERRORS_COLUMNS}
							rows={isUsersSelected ? userActionErrorRows : eventActionErrorRows}
							noRowsMessage={'No errors have occurred'}
						/>
					</div>
				</>
			) : (
				<div className='connection-metrics__nothing-to-show'>
					Currently there is nothing to show for connection {c.name}
				</div>
			)}
		</div>
	);
};

const computeActionErrorRows = (
	connection: TransformedConnection,
	actionErrors: ActionError[],
	stepTerms: string[],
): GridRow[] => {
	const quotedTextToCode = (input: string): string => {
		const style = 'background:#eee; padding: 2px 8px; border-radius: 6px; font-size:12px;';
		return input.replace(/«(.*?)»/g, (_, content) => {
			return `<code style="${style}">${content}</code>`;
		});
	};
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
				stepTerms[error.step],
				formatNumber(error.count),
				<RelativeTime date={error.lastOccurred} />,
				quotedTextToCode(error.message),
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

export default ConnectionMetrics;
