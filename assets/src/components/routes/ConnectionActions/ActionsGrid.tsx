import React, { useContext, useState, useEffect, useRef, ReactNode } from 'react';
import Grid from '../../base/Grid/Grid';
import { SCHEDULE_PERIODS } from '../../../lib/core/action';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { UnprocessableError } from '../../../lib/api/errors';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlRadio from '@shoelace-style/shoelace/dist/react/radio/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import { Action, ActionType } from '../../../lib/api/types/action';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import FeedbackButton, { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import { Execution } from '../../../lib/api/types/responses';
import { sleep } from '../../../utils/sleep';
import { Link } from '../../base/Link/Link';
import AlertDialog from '../../base/AlertDialog/AlertDialog';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: '' }];

const TIME_DELTA = 1000;

interface ActionsGridProps {
	newActionID: React.MutableRefObject<number>;
	actions: Action[];
	onSelectAction: (action: Action) => void;
}

const ActionsGrid = ({ newActionID, actions, onSelectAction }: ActionsGridProps) => {
	const [runningActions, setRunningActions] = useState<number[]>([]);
	const [actionToDelete, setActionToDelete] = useState<number>();

	const { api, handleError, setIsLoadingConnections } = useContext(AppContext);
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
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onDeleteAction = (actionID: number) => {
		setActionToDelete(actionID);
	};

	const onConfirmDeleteAction = async () => {
		newActionID.current = 0; // do not re-trigger the animation of the new action's row during the repainting.
		try {
			await api.workspaces.connections.deleteAction(connection.id, actionToDelete);
		} catch (err) {
			handleError(err);
			setActionToDelete(null);
			return;
		}
		setActionToDelete(null);
		setIsLoadingConnections(true);
	};

	const executeAction = async (actionID: number) => {
		runButtonRefs.current[actionID].current!.load();
		const startTime = new Date().getTime();
		try {
			await api.workspaces.connections.executeAction(connection.id, actionID);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				runButtonRefs.current[actionID].current!.error(err.message);
				return;
			}
			runButtonRefs.current[actionID].current!.stop();
			handleError(err);
			return;
		}

		let execution: Execution;
		while (execution == null) {
			let executions: Execution[];
			try {
				executions = await api.workspaces.connections.executions(connection.id);
			} catch (err) {
				handleError(err);
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
					<div className='connection-actions__link-to-overview'>
						Go to{' '}
						<Link path={`connections/${connection.id}/overview?failed-execution-action=${actionID}`}>
							<span className='connection-actions__link'>overview</span>
						</Link>{' '}
						for details
					</div>
				</>,
			);
			return;
		}

		runButtonRefs.current[actionID].current!.confirm();
	};

	const onSchedulerPeriodChange = async (e: any, actionID: number) => {
		const target = e.currentTarget;
		const period = SCHEDULE_PERIODS[target.value];
		try {
			await api.workspaces.connections.setActionSchedulePeriod(connection.id, actionID, period);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onManageClick = (action: Action) => {
		for (const key in runButtonRefs.current) {
			const button = runButtonRefs.current[key].current;
			if (button != null) {
				button.hideError();
			}
		}
		onSelectAction(action);
	};

	const isActionExecutionSupported =
		connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server';

	const rows: GridRow[] = [];
	for (const a of actions) {
		let linkedActionType: ActionType | null = null;
		for (const t of connection.actionTypes!) {
			if (a.Target === 'Users' || a.Target === 'Groups') {
				if (a.Target === t.Target) {
					linkedActionType = t;
				}
			} else {
				const eventActionTypes = connection.actionTypes.filter((actionType) => actionType.Target === 'Events');
				linkedActionType = eventActionTypes.find((actionType) => actionType.EventType === a.EventType);
			}
		}
		if (linkedActionType == null) {
			throw new Error(`Event type '${a.EventType}' of action ${a.ID} does not exist anymore`);
		}
		const nameCell = (
			<div className='connection-actions__action-name'>
				<div className='connection-actions__action-name-name'>{a.Name}</div>
				<div className='connection-actions__action-name-description'>{linkedActionType.Description}</div>
			</div>
		);
		let conditionsCell: ReactNode;
		if (a.Filter != null) {
			const cells: ReactNode[] = [];
			for (const [i, c] of a.Filter.Conditions.entries()) {
				cells.push(
					<div key={i}>
						{c.Property} {c.Operator}{' '}
						{c.Values != null
							? c.Values.map((val, i) => {
									let v = '';
									if (i > 0) {
										v += '-';
									}
									v += val;
									return v;
								})
							: ''}
					</div>,
				);
			}
			conditionsCell = <div className='connection-actions__action-filter'>{cells}</div>;
		} else {
			conditionsCell = '-';
		}
		const enabledCell = <SlSwitch onSlChange={() => onActionStatusSwitch(a.ID)} checked={a.Enabled}></SlSwitch>;
		const actionsCell = (
			<div className='connection-actions__buttons'>
				{(a.Target === 'Users' || a.Target === 'Groups') && isActionExecutionSupported && (
					<>
						<SlDropdown>
							<SlButton
								slot='trigger'
								variant='default'
								size='small'
								className='connection-actions__scheduler-button'
							>
								<SlIcon slot='prefix' name='clock' />
								Schedule: {a.SchedulePeriod}
							</SlButton>
							<SlMenu className='connection-actions__scheduler-options'>
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
							className='connection-actions__run-button'
							size='small'
							onClick={() => executeAction(a.ID)}
							loading={runningActions.includes(a.ID)}
							disabled={!connection.enabled}
						>
							<SlIcon slot='prefix' name='play' />
							Run now
						</FeedbackButton>
					</>
				)}
				<SlButton variant='default' size='small' onClick={() => onManageClick(a)}>
					Manage...
				</SlButton>
				<SlButton
					className='connection-actions__delete-action'
					variant='danger'
					size='small'
					onClick={() => onDeleteAction(a.ID)}
				>
					Delete
				</SlButton>
			</div>
		);
		const row: GridRow = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: String(a.ID) };
		if (a.ID === newActionID.current && connection.actions!.length > 1) {
			row.animation = 'fade-in';
		}
		rows.push(row);
	}

	return (
		<>
			<Grid rows={rows} columns={GRID_COLUMNS} noRowsMessage='No actions to show'></Grid>
			<AlertDialog
				variant='danger'
				isOpen={actionToDelete != null}
				onClose={() => setActionToDelete(null)}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={() => setActionToDelete(null)}>Cancel</SlButton>
						<SlButton variant='danger' onClick={onConfirmDeleteAction}>
							Delete
						</SlButton>
					</>
				}
			>
				<p>If you continue, you will permanently lose the action</p>
			</AlertDialog>
		</>
	);
};

export default ActionsGrid;
