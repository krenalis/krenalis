import React, { useContext, useState, useEffect, useRef, ReactNode } from 'react';
import Grid from '../../shared/Grid/Grid';
import { SCHEDULE_PERIODS } from '../../../lib/helpers/transformedAction';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { UnprocessableError } from '../../../lib/api/errors';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlRadio from '@shoelace-style/shoelace/dist/react/radio/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import { Action, ActionType } from '../../../types/external/action';
import { ShoelaceEventTarget } from '../../../types/internal/app';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import FeedbackButton, { FeedbackButtonRef } from '../../shared/FeedbackButton/FeedbackButton';
import { Execution } from '../../../types/external/api';
import { sleep } from '../../../lib/utils/sleep';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: '' }];

const TIME_DELTA = 1000;

interface ActionsGridProps {
	newActionID: React.MutableRefObject<number>;
	actions: Action[];
	onSelectAction: (action: Action) => void;
}

const ActionsGrid = ({ newActionID, actions, onSelectAction }: ActionsGridProps) => {
	const [runningActions, setRunningActions] = useState<number[]>([]);
	const { api, showError, setAreConnectionsStale, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const runButtonRefs = useRef<{
		[key: number]: React.RefObject<FeedbackButtonRef>;
	}>({});

	useEffect(() => {
		const running: number[] = [];
		for (const a of actions) {
			if (a.Running) {
				running.push(a.ID);
			}
		}
		setRunningActions(running);
	}, [actions]);

	useEffect(() => {
		for (const a of actions) {
			runButtonRefs.current[a.ID] = React.createRef();
		}
	}, [actions]);

	const onActionStatusSwitch = async (actionID: number) => {
		const index = connection.actions!.findIndex((a) => a.ID === actionID);
		const enabledValue = connection.actions![index].Enabled;
		try {
			await api.workspaces.connections.setActionStatus(connection.id, actionID, !enabledValue);
		} catch (err) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const onRemoveAction = async (actionID: number) => {
		newActionID.current = 0; // avoid repainting with the animation on the new action's row
		try {
			await api.workspaces.connections.deleteAction(connection.id, actionID);
		} catch (err) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const executeAction = async (actionID: number) => {
		runButtonRefs.current[actionID].current!.load();
		const startTime = new Date().getTime();
		const errorButton = (
			<div className='linkToOverview'>
				Go to{' '}
				<span
					className='link'
					onClick={() =>
						redirect(`connections/${connection.id}/overview?failed-execution-action=${actionID}`)
					}
				>
					overview
				</span>{' '}
				for details
			</div>
		);
		try {
			await api.workspaces.connections.executeAction(connection.id, actionID, true); // TODO: handle the reimport bool.
		} catch (err) {
			if (err instanceof UnprocessableError) {
				runButtonRefs.current[actionID].current!.error(
					<>
						{err.message}
						{errorButton}
					</>,
				);
				return;
			}
			runButtonRefs.current[actionID].current!.stop();
			showError(err);
			return;
		}

		let execution: Execution;
		while (execution == null) {
			let executions: Execution[];
			try {
				executions = await api.workspaces.connections.executions(connection.id);
			} catch (err) {
				showError(err);
				return;
			}

			const exec = executions
				.filter((imp) => {
					return imp.Action === actionID;
				})
				.filter((imp) => {
					return new Date(imp.StartTime).getTime() >= startTime - TIME_DELTA;
				})[0];

			if (!exec || exec.EndTime == null) {
				// wait before making a new request.
				await sleep(500);
				continue;
			}

			execution = exec;
		}

		if (execution.Error !== '') {
			runButtonRefs.current[actionID].current!.error(
				<>
					{execution.Error}
					{errorButton}
				</>,
			);
			return;
		}

		runButtonRefs.current[actionID].current!.confirm();
	};

	const onSchedulerPeriodChange = async (e: Event, actionID: number) => {
		const target = e.currentTarget as ShoelaceEventTarget;
		const period = SCHEDULE_PERIODS[target.value];
		try {
			await api.workspaces.connections.setActionSchedulePeriod(connection.id, actionID, period);
		} catch (err) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const isActionExecutionSupported =
		connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server';

	const rows: GridRow[] = [];
	for (const a of actions) {
		let linkedActionType: ActionType | null = null;
		for (const t of connection.actionTypes!) {
			if (a.Target === 'Users' || a.Target === 'Groups') {
				if (a.Target === t.Target) linkedActionType = t;
				continue;
			}
			if (a.EventType === t.EventType) linkedActionType = t;
		}
		if (linkedActionType == null) {
			throw Error(`Event type '${a.EventType}' of action ${a.ID} does not exist anymore`);
		}
		const nameCell = (
			<div className='actionName'>
				<div className='name'>{a.Name}</div>
				<div className='description'>{linkedActionType.Description}</div>
			</div>
		);
		const conditionsCell: ReactNode[] = [];
		if (a.Filter != null) {
			for (const [i, c] of a.Filter.Conditions.entries()) {
				conditionsCell.push(
					<div key={i}>
						{c.Property} {c.Operator} {c.Value}
					</div>,
				);
			}
		} else {
			conditionsCell.push('-');
		}
		const enabledCell = <SlSwitch onSlChange={() => onActionStatusSwitch(a.ID)} checked={a.Enabled}></SlSwitch>;
		const actionsCell = (
			<div className='actionButtons'>
				{(a.Target === 'Users' || a.Target === 'Groups') && isActionExecutionSupported && (
					<>
						<SlDropdown>
							<SlButton slot='trigger' variant='default' size='small'>
								<SlIcon slot='prefix' name='clock' />
								Schedule: {a.SchedulePeriod}
							</SlButton>
							<SlMenu className='schedulerOptions'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, a.ID)}
									value={Object.keys(SCHEDULE_PERIODS).find(
										(k) => SCHEDULE_PERIODS[k] === a.SchedulePeriod,
									)}
								>
									{Object.entries(SCHEDULE_PERIODS).map(([value, time]) => (
										<SlRadio key={value} value={value}>
											{time}
										</SlRadio>
									))}
								</SlRadioGroup>
							</SlMenu>
						</SlDropdown>
						<FeedbackButton
							ref={runButtonRefs.current[a.ID]}
							className='runButton'
							size='small'
							onClick={() => executeAction(a.ID)}
							loading={runningActions.includes(a.ID)}
						>
							<SlIcon slot='prefix' name='play' />
							Run now
						</FeedbackButton>
					</>
				)}
				<SlButton variant='default' size='small' onClick={() => onSelectAction(a)}>
					Edit...
				</SlButton>
				<SlButton className='removeAction' variant='danger' size='small' onClick={() => onRemoveAction(a.ID)}>
					Remove
				</SlButton>
			</div>
		);
		const row: GridRow = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: String(a.ID) };
		if (a.ID === newActionID.current && connection.actions!.length > 1) {
			row.animation = 'fade-in';
		}
		rows.push(row);
	}

	return <Grid rows={rows} columns={GRID_COLUMNS} noRowsMessage='No actions to show'></Grid>;
};

export default ActionsGrid;
