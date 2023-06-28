import { useMemo, useRef, useContext } from 'react';
import Action from '../../../lib/connections/action';
import Grid from '../../common/Grid/Grid';
import { AppContext } from '../../../providers/AppProvider';
import { UnprocessableError } from '../../../lib/api/errors';
import * as statuses from '../../../constants/statuses';
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

const ActionsGrid = ({ connection, onSelectAction }) => {
	const newActionID = useRef(0);

	const { api, showError, showStatus, setAreConnectionsStale } = useContext(AppContext);

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
		const [, err] = await api.connections.deleteAction(connection.id, actionID);
		if (err != null) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	const executeAction = async (actionID) => {
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
		setAreConnectionsStale(true);
	};

	const onSchedulerPeriodChange = async (e, actionID) => {
		const period = Action.SCHEDULE_PERIODS[e.currentTarget.value];
		const [, err] = await api.connections.setActionSchedulePeriod(connection.id, actionID, period);
		if (err != null) {
			showError(err);
			return;
		}
		setAreConnectionsStale(true);
	};

	return useMemo(() => {
		const rows = [];
		for (const a of connection.actions) {
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
					{(a.Target === 'Users' || a.Target === 'Groups') && (
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
										value={Object.keys(Action.SCHEDULE_PERIODS).find(
											(k) => Action.SCHEDULE_PERIODS[k] === a.SchedulePeriod
										)}
									>
										{Object.entries(Action.SCHEDULE_PERIODS).map(([value, time]) => (
											<SlRadio value={value}>{time}</SlRadio>
										))}
									</SlRadioGroup>
								</SlMenu>
							</SlDropdown>
							<SlButton
								disabled={a.Running}
								variant='default'
								className='runButton'
								size='small'
								onClick={() => executeAction(a.ID)}
							>
								{a.Running ? <SlSpinner slot='prefix' /> : <SlIcon slot='prefix' name='play' />}
								Run now
							</SlButton>
						</>
					)}
					<SlButton variant='default' size='small' onClick={() => onSelectAction(a)}>
						Edit...
					</SlButton>
					<SlButton
						className='removeAction'
						variant='danger'
						size='small'
						onClick={() => onRemoveAction(a.ID)}
					>
						Remove
					</SlButton>
				</div>
			);
			const row = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: a.ID };
			if (a.ID === newActionID.current && connection.actions.length > 1) {
				row.animation = 'fade-in';
				newActionID.current = 0;
			}
			rows.push(row);
		}
		return <Grid rows={rows} columns={GRID_COLUMNS} noRowsMessage='No actions to show'></Grid>;
	}, [connection]);
};

export default ActionsGrid;
