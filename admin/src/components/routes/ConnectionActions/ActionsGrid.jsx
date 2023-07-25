import { useContext, useState, useEffect } from 'react';
import Grid from '../../shared/Grid/Grid';
import { SCHEDULE_PERIODS } from '../../../lib/helpers/action';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import {
	SlButton,
	SlIcon,
	SlSwitch,
	SlSpinner,
	SlDropdown,
	SlMenu,
	SlRadio,
	SlRadioGroup,
} from '@shoelace-style/shoelace/dist/react/index.js';

const GRID_COLUMNS = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: null }];

const ActionsGrid = ({ newActionID, actions, onSelectAction }) => {
	const [runningActions, setRunningActions] = useState([]);

	const { api, showError, showStatus, setAreConnectionsStale } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useEffect(() => {
		const running = [];
		for (const a of actions) {
			if (a.Running) {
				running.push(a.ID);
			}
		}
		setRunningActions(running);
	}, [actions]);

	const onActionStatusSwitch = async (actionID) => {
		const index = connection.actions.findIndex((a) => a.ID === actionID);
		const enabledValue = connection.actions[index].Enabled;
		const [, err] = await api.connections.setActionStatus(connection.id, actionID, !enabledValue);
		if (err != null) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const onRemoveAction = async (actionID) => {
		newActionID.current = 0; // avoid repainting with the animation on the new action's row
		const [, err] = await api.connections.deleteAction(connection.id, actionID);
		if (err != null) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const executeAction = async (actionID) => {
		setRunningActions([...runningActions, actionID]);
		const [, err] = await api.connections.executeAction(connection.id, actionID, true); // TODO: handle the reimport bool.
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ExecutionInProgress':
						showStatus(statuses.actionExecutionInProgress);
						break;
					case 'NoStorage':
						showStatus(statuses.noStorage);
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			return;
		}
	};

	const onSchedulerPeriodChange = async (e, actionID) => {
		const period = SCHEDULE_PERIODS[e.currentTarget.value];
		const [, err] = await api.connections.setActionSchedulePeriod(connection.id, actionID, period);
		if (err != null) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const isActionExecutionSupported =
		connection.type !== 'Website' && connection.type !== 'Mobile' && connection.type !== 'Server';

	const rows = [];
	for (const a of actions) {
		let linkedActionType;
		for (const t of connection.actionTypes) {
			if (a.Target === 'Users' || a.Target === 'Groups') {
				if (a.Target === t.Target) linkedActionType = t;
				continue;
			}
			if (a.EventType === t.EventType) linkedActionType = t;
		}
		if (linkedActionType === undefined) {
			throw Error(`Event type '${a.EventType}' of action ${a.ID} does not exist anymore`);
		}
		const nameCell = (
			<div className='actionName'>
				<div className='name'>{a.Name}</div>
				<div className='description'>{linkedActionType.Description}</div>
			</div>
		);
		const conditionsCell = [];
		if (a.Filter != null) {
			for (const c of a.Filter.Conditions) {
				conditionsCell.push(
					<div>
						{c.Property} {c.Operator} {c.Value}
					</div>
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
										(k) => SCHEDULE_PERIODS[k] === a.SchedulePeriod
									)}
								>
									{Object.entries(SCHEDULE_PERIODS).map(([value, time]) => (
										<SlRadio value={value}>{time}</SlRadio>
									))}
								</SlRadioGroup>
							</SlMenu>
						</SlDropdown>
						<SlButton
							disabled={runningActions.includes(a.ID)}
							variant='default'
							className='runButton'
							size='small'
							onClick={() => executeAction(a.ID)}
						>
							{runningActions.includes(a.ID) ? (
								<SlSpinner slot='prefix' />
							) : (
								<SlIcon slot='prefix' name='play' />
							)}
							Run now
						</SlButton>
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
		const row = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: a.ID };
		if (a.ID === newActionID.current && connection.actions.length > 1) {
			row.animation = 'fade-in';
		}
		rows.push(row);
	}

	return <Grid rows={rows} columns={GRID_COLUMNS} noRowsMessage='No actions to show'></Grid>;
};

export default ActionsGrid;
