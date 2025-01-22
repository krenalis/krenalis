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
import { Variant } from '../App/App.types';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: '' }];

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
			if (a.running) {
				running.push(a.id);
			}
		}
		setRunningActions(running);
	}, [actions]);

	useEffect(() => {
		for (const a of actions) {
			runButtonRefs.current[a.id] = React.createRef();
		}
	}, [actions]);

	const onActionStatusSwitch = async (actionID: number) => {
		const index = connection.actions!.findIndex((a) => a.id === actionID);
		const enabledValue = connection.actions![index].enabled;
		try {
			await api.workspaces.connections.setActionStatus(actionID, !enabledValue);
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
			await api.workspaces.connections.deleteAction(actionToDelete);
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
		let executionID: number;
		try {
			executionID = await api.workspaces.connections.executeAction(actionID);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				runButtonRefs.current[actionID].current!.error(err.message);
				return;
			}
			runButtonRefs.current[actionID].current!.stop();
			handleError(err);
			return;
		}

		let execution: Execution | null = null;
		while (execution == null) {
			await sleep(500);
			try {
				execution = await api.workspaces.connections.execution(executionID);
			} catch (err) {
				handleError(err);
				return;
			}
			if (execution.endTime == null) {
				execution = null;
			}
		}

		if (execution.error !== '') {
			runButtonRefs.current[actionID].current!.error(
				<>
					{execution.error}
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
		const period = e.currentTarget.value === 'Off' ? null : e.currentTarget.value;
		try {
			await api.workspaces.connections.setActionSchedulePeriod(actionID, period);
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
		connection.connector.type !== 'Website' &&
		connection.connector.type !== 'Mobile' &&
		connection.connector.type !== 'Server';

	const rows: GridRow[] = [];
	for (const a of actions) {
		let linkedActionType: ActionType | null = null;
		for (const t of connection.actionTypes!) {
			if (a.target === 'Users' || a.target === 'Groups') {
				if (a.target === t.target) {
					linkedActionType = t;
				}
			} else {
				const eventActionTypes = connection.actionTypes.filter((actionType) => actionType.target === 'Events');
				linkedActionType = eventActionTypes.find((actionType) => actionType.eventType === a.eventType);
			}
		}
		if (linkedActionType == null) {
			throw new Error(`Event type '${a.eventType}' of action ${a.id} does not exist anymore`);
		}
		const nameCell = (
			<div className='connection-actions__action-name'>
				<div className='connection-actions__action-name-name'>{a.name}</div>
				<div className='connection-actions__action-name-description'>{linkedActionType.description}</div>
			</div>
		);

		let conditionsCell: ReactNode;
		if (a.filter != null) {
			const cells: ReactNode[] = [];
			for (const [i, c] of a.filter.conditions.entries()) {
				cells.push(
					<div key={i}>
						{c.property} {c.operator}{' '}
						{c.values != null
							? c.values.map((val, i) => {
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

		const enabledCell = <SlSwitch onSlChange={() => onActionStatusSwitch(a.id)} checked={a.enabled}></SlSwitch>;

		let scheduleDotVariant: Variant = 'neutral';
		if (a.enabled && a.schedulePeriod != null) {
			scheduleDotVariant = 'success';
		}
		const actionsCell = (
			<div className='connection-actions__buttons'>
				{(a.target === 'Users' || a.target === 'Groups') && isActionExecutionSupported && (
					<>
						<FeedbackButton
							ref={runButtonRefs.current[a.id]}
							className='connection-actions__run-button'
							size='small'
							onClick={() => executeAction(a.id)}
							loading={runningActions.includes(a.id)}
							disabled={!a.enabled}
							hoist={true}
						>
							<SlIcon slot='prefix' name='play' />
							Run now
						</FeedbackButton>
						<SlDropdown hoist={true}>
							<SlButton
								slot='trigger'
								variant='default'
								size='small'
								className='connection-actions__scheduler-button'
							>
								<SlIcon slot='prefix' name='clock' />
								Schedule: {a.schedulePeriod || 'Off'}
								<SlIcon
									slot='suffix'
									className={`connection-actions__scheduler-dot connection-actions__scheduler-dot--${scheduleDotVariant}`}
									name='circle-fill'
								/>
							</SlButton>
							<SlMenu className='connection-actions__scheduler-options'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, a.id)}
									value={a.schedulePeriod || 'Off'}
								>
									{SCHEDULE_PERIODS.map((period) => (
										<SlRadio key={period} value={period}>
											{period}
										</SlRadio>
									))}
								</SlRadioGroup>
							</SlMenu>
						</SlDropdown>
					</>
				)}
				<SlButton variant='default' size='small' onClick={() => onManageClick(a)}>
					Manage...
				</SlButton>
				<SlButton
					className='connection-actions__delete-action'
					variant='danger'
					size='small'
					onClick={() => onDeleteAction(a.id)}
				>
					Delete
				</SlButton>
			</div>
		);

		const row: GridRow = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: String(a.id) };
		if (a.id === newActionID.current && connection.actions!.length > 1) {
			row.animation = 'fade-in';
		}

		rows.push(row);
	}

	return (
		<>
			<Grid
				className='connection-actions__grid'
				rows={rows}
				columns={GRID_COLUMNS}
				noRowsMessage='No actions to show'
			></Grid>
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
