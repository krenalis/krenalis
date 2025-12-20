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
import { PipelineError, PipelineErrorsResponse } from '../../../lib/api/types/responses';
import { PipelineMetrics, PipelineTarget } from '../../../lib/api/types/pipeline';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';
import considerAsUTC from '../../../utils/considerUTC';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { formatNumber } from '../../../utils/formatNumber';

interface PipelineMetricsPoint {
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
	{ name: 'Pipeline' },
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

	const [userPipelinesMetrics, setUserPipelinesMetrics] = useState<PipelineMetrics>();
	const [eventPipelinesMetrics, setEventPipelinesMetrics] = useState<PipelineMetrics>();
	const [userPipelinesErrors, setUserPipelinesErrors] = useState<PipelineError[]>([]);
	const [eventPipelinesErrors, setEventPipelinesErrors] = useState<PipelineError[]>([]);
	const [funnelArrows, setFunnelArrows] = useState<ReactNode[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [selectedTarget, setSelectedTarget] = useState<PipelineTarget>(
		new URLSearchParams(window.location.search).get('target') === 'event'
			? 'Event'
			: new URLSearchParams(window.location.search).get('target') === 'user'
				? 'User'
				: c.pipelines.findIndex((p) => p.target === 'Event') !== -1
					? 'Event'
					: 'User',
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
	const [selectedPipeline, setSelectedPipeline] = useState<number | null>(null);

	const { api, handleError } = useContext(AppContext);

	const supportedTargets = useRef([]);
	const currentMetricsIntervalID = useRef<any>();
	const previouslySelectedPipeline = useRef<number | null>(null);

	const isUsersSelected = selectedTarget === 'User';

	let receiveStepTerm = '';
	let finalizeStepTerm = '';
	if (c.isSource) {
		if (isUsersSelected) {
			receiveStepTerm = `Extract from ${c.connector.label}`;
		} else {
			receiveStepTerm = `Receive from ${c.connector.label}`;
		}
		finalizeStepTerm = 'Load into warehouse';
	} else {
		if (isUsersSelected) {
			receiveStepTerm = 'Extract from warehouse';
			finalizeStepTerm = `Load into ${c.connector.label}`;
		} else {
			receiveStepTerm = 'Receive from sources';
			finalizeStepTerm = `Send to ${c.connector.label}`;
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

	const { userPipelineErrorRows, eventPipelineErrorRows } = useMemo(() => {
		const stepTerms = Object.values(stepTermByIdentifier);
		return {
			userPipelineErrorRows: computePipelineErrorRows(c, userPipelinesErrors, stepTerms),
			eventPipelineErrorRows: computePipelineErrorRows(c, eventPipelinesErrors, stepTerms),
		};
	}, [userPipelinesErrors, eventPipelinesErrors]);

	const { userPipelineMetricsData, eventPipelineMetricsData } = useMemo(() => {
		return {
			userPipelineMetricsData: computePipelineMetricsData(userPipelinesMetrics, selectedMetricsRange),
			eventPipelineMetricsData: computePipelineMetricsData(eventPipelinesMetrics, selectedMetricsRange),
		};
	}, [userPipelinesMetrics, eventPipelinesMetrics]);

	const { userFunnelData, eventFunnelData } = useMemo(() => {
		return {
			userFunnelData: computeFunnelData(userPipelinesMetrics),
			eventFunnelData: computeFunnelData(eventPipelinesMetrics),
		};
	}, [userPipelinesMetrics, eventPipelinesMetrics]);

	const steps = useMemo(() => {
		let steps: StepIdentifier[] = [...STEP_IDENTIFIERS];
		switch (c.connector.type) {
			case 'Application':
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
			case 'Webhook':
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
					strokeWidth={1.5}
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
					strokeWidth={1.5}
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
				strokeWidth={1.5}
			></Arrow>,
		);
		setTimeout(() => {
			setFunnelArrows(arrows);
		});
	}, [isLoading, eventFunnelData, userFunnelData]);

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		const fetchData = async () => {
			let userPipelinesIds: number[] = [];
			let eventPipelinesIds: number[] = [];
			if (selectedPipeline == null) {
				for (const pipeline of c.pipelines) {
					if (pipeline.target === 'User') {
						userPipelinesIds.push(pipeline.id);
					} else if (pipeline.target === 'Event') {
						eventPipelinesIds.push(pipeline.id);
					}
				}
			} else {
				const p = c.pipelines.find((pipeline) => pipeline.id === selectedPipeline);
				if (p.target === 'User') {
					userPipelinesIds.push(p.id);
				} else if (p.target === 'Event') {
					eventPipelinesIds.push(p.id);
				}
			}

			if (userPipelinesIds.length === 0 && eventPipelinesIds.length === 0) {
				stopLoading();
				return;
			}

			if (userPipelinesIds.length > 0) {
				supportedTargets.current.push('User');
			}

			if (eventPipelinesIds.length > 0) {
				supportedTargets.current.push('Event');
			}

			let fetchMetrics: (...args) => Promise<PipelineMetrics> = null;
			if (selectedMetricsRange === 'last15Minutes') {
				fetchMetrics = async (pipelineIds) =>
					await api.workspaces.pipelineMetricsPerMinute(MINUTES_COUNT, pipelineIds);
			} else if (selectedMetricsRange === 'last24Hours') {
				fetchMetrics = async (pipelineIds) =>
					await api.workspaces.pipelineMetricsPerHour(HOURS_COUNT, pipelineIds);
			} else if (selectedMetricsRange === 'last7Days') {
				fetchMetrics = async (pipelineIds) =>
					await api.workspaces.pipelineMetricsPerDay(DAYS_COUNT, pipelineIds);
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
				fetchMetrics = async (pipelineIds) =>
					await api.workspaces.pipelineMetricsPerDate(customMetricsRange[0].startDate, endDate, pipelineIds);
			}

			let target = selectedTarget;
			let ids: number[] = [];
			if (target === 'User') {
				ids = userPipelinesIds;
			} else if (target === 'Event') {
				ids = eventPipelinesIds;
			}

			let metrics: PipelineMetrics;
			try {
				metrics = await fetchMetrics(ids);
			} catch (err) {
				handleError(err);
				stopLoading();
				return;
			}
			if (target === 'User') {
				setUserPipelinesMetrics(metrics);
			} else {
				setEventPipelinesMetrics(metrics);
			}

			let errorRes: PipelineErrorsResponse;
			try {
				errorRes = await api.workspaces.pipelineErrors(metrics.start, metrics.end, ids, 0, 50, null);
			} catch (err) {
				handleError(err);
				stopLoading();
				return;
			}
			if (target === 'User') {
				setUserPipelinesErrors(errorRes.errors);
			} else {
				setEventPipelinesErrors(errorRes.errors);
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
	}, [c, selectedTarget, selectedMetricsRange, customMetricsRange, selectedPipeline]);

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

	const onChangeSelectedTarget = (target: PipelineTarget) => {
		const isAlreadySelected = selectedTarget === target;
		if (isAlreadySelected) {
			return;
		}
		const toRestore = previouslySelectedPipeline.current;
		previouslySelectedPipeline.current = null;
		setSelectedTarget(target);
		if (selectedPipeline != null) {
			previouslySelectedPipeline.current = selectedPipeline;
		}
		setSelectedPipeline(toRestore);
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

	const onChangesetSelectedPipeline = (e: any) => {
		const v = e.target.value;
		if (v === '') {
			setSelectedPipeline(null);
		} else {
			setSelectedPipeline(Number(v));
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
								Profiles
							</SlButton>
						</SlButtonGroup>
						{c.pipelines?.length > 1 &&
							!((c.isSDK || c.isWebhook) && c.isSource && selectedTarget === 'Event') && (
								<SlSelect
									size='small'
									label='Pipeline'
									onSlChange={onChangesetSelectedPipeline}
									value={selectedPipeline == null ? '' : String(selectedPipeline)}
									className={`connection-metrics__pipelines${selectedPipeline != null ? ' connection-metrics__pipelines--filtered' : ''}`}
									clearable
								>
									{c.pipelines?.map((p) => {
										if (p.target == selectedTarget) {
											return <SlOption value={String(p.id)}>{p.name}</SlOption>;
										}
										return null;
									})}
								</SlSelect>
							)}
					</div>
					<div className='connection-metrics__chart'>
						<div className='connection-metrics__chart-heading'>
							{chartTitle} {isUsersSelected ? 'profiles' : 'events'} <span>{titleRange}</span>
						</div>
						<ResponsiveContainer width='100%' height='100%'>
							<ComposedChart
								data={isUsersSelected ? userPipelineMetricsData : eventPipelineMetricsData}
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
						<div className='connection-metrics__funnel-content'>
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
					</div>
					<div className='connection-metrics__errors'>
						<div className='connection-metrics__errors-heading'>
							Error log <span>{titleRange}</span>
						</div>
						<Grid
							columns={ERRORS_COLUMNS}
							rows={isUsersSelected ? userPipelineErrorRows : eventPipelineErrorRows}
							noRowsMessage={'No errors have occurred'}
						/>
					</div>
				</>
			) : (
				<div className='connection-metrics__nothing-to-show'>
					Currently there is nothing to show for this connection
				</div>
			)}
		</div>
	);
};

const computePipelineErrorRows = (
	connection: TransformedConnection,
	pipelineErrors: PipelineError[],
	stepTerms: string[],
): GridRow[] => {
	const quotedTextToCode = (input: string): string => {
		const style = 'background:#eee; padding: 2px 8px; border-radius: 6px; font-size:12px;';
		return input.replace(/«(.*?)»/g, (_, content) => {
			return `<code style="${style}">${content}</code>`;
		});
	};
	if (pipelineErrors == null) {
		return null;
	}
	let pipelineErrorRows: GridRow[] = [];
	for (const error of pipelineErrors) {
		const row = {
			cells: [
				<Link path={`connections/${connection.id}/pipelines/edit/${error.pipeline}`}>
					{connection.pipelines.find((p) => p.id == error.pipeline)?.name}
				</Link>,
				stepTerms[error.step],
				formatNumber(error.count),
				<RelativeTime date={error.lastOccurred} />,
				quotedTextToCode(error.message),
			],
		};
		pipelineErrorRows.push(row);
	}
	return pipelineErrorRows;
};

const computePipelineMetricsData = (pipelineMetrics: PipelineMetrics, range: metricsRange): PipelineMetricsPoint[] => {
	if (pipelineMetrics == null) {
		return [];
	}
	let points: PipelineMetricsPoint[] = [];
	const timeLength = pipelineMetrics.passed.length;
	let counter = timeLength;
	for (let timeUnit = 0; timeUnit < timeLength; timeUnit++) {
		let failedTotal = 0;
		for (let i = 0; i < 6; i++) {
			if (i === 2) {
				// filtered must not be considered as failed.
				continue;
			}
			failedTotal += pipelineMetrics.failed[timeUnit][i];
		}
		let filteredTotal = pipelineMetrics.failed[timeUnit][2];
		let passedTotal = pipelineMetrics.passed[timeUnit][5];
		let total = failedTotal + filteredTotal + passedTotal;
		const d = new Date(pipelineMetrics.end.getTime());
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

const computeFunnelData = (pipelineMetrics: PipelineMetrics): FunnelData => {
	if (pipelineMetrics == null) {
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
		for (const p of pipelineMetrics.passed) {
			totalPassed += p[i];
		}
		for (const f of pipelineMetrics.failed) {
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
