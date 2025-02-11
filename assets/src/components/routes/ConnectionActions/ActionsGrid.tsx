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
import { Action } from '../../../lib/api/types/action';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import FeedbackButton, { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import { Execution } from '../../../lib/api/types/responses';
import { sleep } from '../../../utils/sleep';
import { Link } from '../../base/Link/Link';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { Variant } from '../App/App.types';
import getConnectorLogo from '../../helpers/getConnectorLogo';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: '' }];

interface ActionsGridProps {
	newActionID: React.MutableRefObject<number>;
	actions: Action[];
	onSelectAction: (action: Action) => void;
}

const ActionsGrid = ({ newActionID, actions, onSelectAction }: ActionsGridProps) => {
	const [runningActions, setRunningActions] = useState<number[]>([]);
	const [actionToDelete, setActionToDelete] = useState<number>();

	const { api, handleError, setIsLoadingConnections, connectors } = useContext(AppContext);
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

		let link = `connections/${connection.id}/overview`;
		if (execution.error) {
			link += `?failed-execution-action=${actionID}`;
		}
		const overviewLink = (
			<div className='connection-actions__link-to-overview'>
				Go to{' '}
				<Link path={link}>
					<span className='connection-actions__link'>overview</span>
				</Link>{' '}
				for details
			</div>
		);

		if (execution.error !== '') {
			runButtonRefs.current[actionID].current!.error(
				<>
					{execution.error}
					{overviewLink}
				</>,
			);
			return;
		}

		const passed = execution.passed;
		const failed = execution.failed;
		const infoMessage = (
			<div className='connection-actions__execution-info'>
				<div className='connection-actions__execution-info-title'>
					{connection.isSource ? 'Import' : 'Export'} completed
				</div>
				<ul>
					<li>
						{passed} {passed === 1 ? 'user' : 'users'} {connection.isSource ? 'imported' : 'exported'}
					</li>
					<li>
						{failed === 0
							? 'No errors occurred'
							: `${failed} not ${connection.isSource ? 'imported' : 'exported'} due to errors`}
					</li>
				</ul>
				{overviewLink}
			</div>
		);
		runButtonRefs.current[actionID].current!.info(infoMessage);
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
				button.hideTooltip();
			}
		}
		onSelectAction(action);
	};

	const isActionExecutionSupported =
		connection.connector.type !== 'Website' &&
		connection.connector.type !== 'Mobile' &&
		connection.connector.type !== 'Server';

	const rows: GridRow[] = [];
	for (const action of actions) {
		const actionType = connection.actionTypes.find(
			(t) => action.target === t.target && (!('eventType' in action) || action.eventType === t.eventType),
		);
		if (actionType == null) {
			throw new Error(
				`Connection ${connection.id} no longer has target '${action.target}' and event type '${action.eventType}' for action ${action.id}`,
			);
		}

		let description = actionType.description;
		if (connection.isFileStorage) {
			description = `${connection.isSource ? 'Import' : 'Export'} the ${action.target.toLowerCase()} ${connection.isSource ? 'from' : 'to'} ${action.path}`;
		}

		let logo: ReactNode;
		if (connection.isFileStorage) {
			const formatConnector = connectors.find((c) => c.name === action.format);
			logo = (
				<div className='connection-actions__action-logo'>
					<span style={{ position: 'relative', top: '3px' }}>{getConnectorLogo(formatConnector.icon)}</span>
				</div>
			);
		}

		const nameCell = (
			<div className='connection-actions__action-name'>
				{logo}
				<div className='connection-actions__action-text'>
					<div className='connection-actions__action-name-name'>{action.name}</div>
					<div className='connection-actions__action-name-description'>{description}</div>
				</div>
			</div>
		);

		let conditionsCell: ReactNode;
		if (action.filter != null) {
			const cells: ReactNode[] = [];
			for (const [i, c] of action.filter.conditions.entries()) {
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

		const enabledCell = (
			<SlSwitch onSlChange={() => onActionStatusSwitch(action.id)} checked={action.enabled}></SlSwitch>
		);

		let scheduleDotVariant: Variant = 'neutral';
		if (action.enabled && action.schedulePeriod != null) {
			scheduleDotVariant = 'success';
		}
		const actionsCell = (
			<div className='connection-actions__buttons'>
				{(action.target === 'Users' || action.target === 'Groups') && isActionExecutionSupported && (
					<>
						<FeedbackButton
							ref={runButtonRefs.current[action.id]}
							className='connection-actions__run-button'
							size='small'
							onClick={() => executeAction(action.id)}
							loading={runningActions.includes(action.id)}
							disabled={!action.enabled}
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
								Schedule: {action.schedulePeriod || 'Off'}
								<SlIcon
									slot='suffix'
									className={`connection-actions__scheduler-dot connection-actions__scheduler-dot--${scheduleDotVariant}`}
									name='circle-fill'
								/>
							</SlButton>
							<SlMenu className='connection-actions__scheduler-options'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, action.id)}
									value={action.schedulePeriod || 'Off'}
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
				<SlButton variant='default' size='small' onClick={() => onManageClick(action)}>
					Manage...
				</SlButton>
				<SlButton
					className='connection-actions__delete-action'
					variant='danger'
					size='small'
					onClick={() => onDeleteAction(action.id)}
				>
					Delete
				</SlButton>
			</div>
		);

		const row: GridRow = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: String(action.id) };
		if (action.id === newActionID.current && connection.actions!.length > 1) {
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
