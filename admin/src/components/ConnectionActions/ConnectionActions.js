import { useState, useEffect, useContext } from 'react';
import './ConnectionActions.css';
import LittleLogo from '../LittleLogo/LittleLogo';
import Flex from '../Flex/Flex';
import ListTile from '../ListTile/ListTile';
import IconWrapper from '../IconWrapper/IconWrapper';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import StyledGrid from '../StyledGrid/StyledGrid';
import Action from '../Action/Action';
import statuses from '../../constants/statuses';
import { UnprocessableError } from '../../api/errors';
import { schedulePeriods } from '../../utils/schedulePeriods';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import {
	SlButton,
	SlIcon,
	SlDialog,
	SlSwitch,
	SlSpinner,
	SlDropdown,
	SlMenu,
	SlRadio,
	SlRadioGroup,
} from '@shoelace-style/shoelace/dist/react/index.js';

let ConnectionActions = () => {
	let [actions, setActions] = useState([]);
	let [actionTypes, setActionTypes] = useState([]);
	let [isDialogOpen, setIsDialogOpen] = useState(false);
	let [selectedActionType, setSelectedActionType] = useState(null);
	let [selectedAction, setSelectedAction] = useState(null);
	let [description, setDescription] = useState(null);
	let [isLoading, setIsLoading] = useState(true);

	let { API, showError, showStatus, connectors } = useContext(AppContext);
	let { connection: c, setCurrentConnectionSection } = useContext(ConnectionContext);

	setCurrentConnectionSection('actions');

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 500);
		};
		const fetchData = async () => {
			let err, actionTypes, actions, connector;
			[actionTypes, err] = await API.connections.actionTypes(c.ID);
			if (err != null) {
				showError(err);
				stopLoading();
				return;
			}
			setActionTypes(actionTypes);
			[actions, err] = await API.connections.actions(c.ID);
			if (err != null) {
				showError(err);
				stopLoading();
				return;
			}
			setActions(actions);
			[connector, err] = await API.connectors.get(c.Connector);
			if (err != null) {
				showError(err);
				stopLoading();
				return;
			}
			let description;
			if (c.Role === 'Source') {
				description = connector.SourceDescription;
			} else {
				description = connector.DestinationDescription;
			}
			setDescription(description);
			stopLoading();
		};
		fetchData();
	}, []);

	const onActionStatusSwitch = async (actionID) => {
		let a = [...actions];
		let i = a.findIndex((a) => a.ID === actionID);
		let isEnabled = !a[i].Enabled;
		let [, err] = await API.connections.setActionStatus(c.ID, actionID, isEnabled);
		if (err != null) {
			showError(err);
			return;
		}
		a[i].Enabled = isEnabled;
		setActions(a);
	};

	const onRemoveAction = async (actionID) => {
		let err;
		[, err] = await API.connections.deleteAction(c.ID, actionID);
		if (err != null) {
			showError(err);
			return;
		}
		let a = [...actions];
		let filtered = a.filter((a) => a.ID !== actionID);
		setActions(filtered);
		let actionTypes;
		[actionTypes, err] = await API.connections.actionTypes(c.ID);
		if (err != null) {
			showError(err);
			return;
		}
		setActionTypes(actionTypes);
	};

	const executeAction = async (actionID) => {
		let err;
		[, err] = await API.connections.executeAction(c.ID, actionID, true); // TODO: handle the reimport bool.
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
		showStatus(statuses.importStarted);
	};

	const onSchedulerPeriodChange = async (e, actionID) => {
		let period = schedulePeriods[e.currentTarget.value];
		let [, err] = await API.connections.setActionSchedulePeriod(c.ID, actionID, period);
		if (err != null) {
			showError(err);
			return;
		}
	};

	const refresh = async () => {
		setSelectedActionType(null);
		setSelectedAction(null);
		let actionTypes, err;
		[actionTypes, err] = await API.connections.actionTypes(c.ID);
		if (err != null) {
			showError(err);
			return;
		}
		setActionTypes(actionTypes);
		let actions;
		[actions, err] = await API.connections.actions(c.ID);
		if (err != null) {
			showError(err);
			return;
		}
		setActions(actions);
	};

	if (isLoading) {
		return (
			<div className='ConnectionActions loading'>
				<SlSpinner
					style={{
						fontSize: '3rem',
						'--track-width': '6px',
					}}
				></SlSpinner>
			</div>
		);
	}

	let columns = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: null }];

	let rows = [];
	for (let a of actions) {
		let linkedActionType;
		for (let t of actionTypes) {
			if (a.Target === 'Users' || a.Target === 'Groups') {
				if (a.Target === t.Target) linkedActionType = t;
				continue;
			}
			if (a.EventType === t.EventType) linkedActionType = t;
		}
		let nameCell = (
			<div className='actionName'>
				<div className='name'>{a.Name}</div>
				<div className='description'>{linkedActionType.Description}</div>
			</div>
		);
		let conditionsCell = [];
		if (a.Filter != null) {
			for (let c of a.Filter.Conditions) {
				conditionsCell.push(
					<div>
						{c.Property} {c.Operator} {c.Value}
					</div>
				);
			}
		} else {
			conditionsCell.push('-');
		}
		let enabledCell = <SlSwitch onSlChange={() => onActionStatusSwitch(a.ID)} checked={a.Enabled}></SlSwitch>;
		let actionsCell = (
			<div className='actionButtons'>
				{(a.Target === 'Users' || a.Target === 'Groups') && (
					<>
						<SlDropdown>
							<SlButton slot='trigger' variant='default' size='small'>
								<SlIcon slot='prefix' name='clock' />
								Scheduler
							</SlButton>
							<SlMenu className='schedulerOptions'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, a.ID)}
									value={Object.keys(schedulePeriods).find(
										(k) => schedulePeriods[k] === a.SchedulePeriod
									)}
								>
									{Object.entries(schedulePeriods).map(([value, time]) => (
										<SlRadio value={value}>{time}</SlRadio>
									))}
								</SlRadioGroup>
							</SlMenu>
						</SlDropdown>
						<SlButton variant='default' size='small' onClick={() => executeAction(a.ID)}>
							Execute
						</SlButton>
					</>
				)}
				<SlButton variant='default' size='small' onClick={() => setSelectedAction(a)}>
					Edit...
				</SlButton>
				<SlButton className='removeAction' variant='danger' size='small' onClick={() => onRemoveAction(a.ID)}>
					Remove
				</SlButton>
			</div>
		);
		rows.push([nameCell, conditionsCell, enabledCell, actionsCell]);
	}

	if (selectedActionType !== null || selectedAction !== null) {
		return <Action actionType={selectedActionType} action={selectedAction} onClose={refresh}></Action>;
	}

	let connector = connectors.find((connector) => connector.ID === c.Connector);
	let logo;
	if (connector.Icon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connector.Icon} />;
	}

	let standardActionTypes = [];
	let eventActionTypes = [];
	for (let t of actionTypes) {
		let tile = (
			<ListTile
				icon={logo}
				name={t.Name}
				description={t.Description}
				missingSchema={t.MissingSchema}
				onClick={() => {
					setIsDialogOpen(false);
					setSelectedActionType(t);
				}}
			/>
		);
		if (t.Target === 'Users' || t.Target === 'Groups') {
			standardActionTypes.push(tile);
		} else {
			eventActionTypes.push(tile);
		}
	}

	return (
		<>
			<div className='ConnectionActions'>
				{actions.length === 0 ? (
					<>
						<div className='noAction'>
							<IconWrapper name='send-exclamation' size={40} />
							<div className='description'>Add an action to {description}</div>
							<SlButton
								variant='primary'
								onClick={() => {
									setIsDialogOpen(true);
								}}
							>
								Add action...
							</SlButton>
						</div>
					</>
				) : (
					<>
						<Flex justifyContent={'end'} alignItems={'center'}>
							<SlButton
								variant='text'
								onClick={() => {
									setIsDialogOpen(true);
								}}
							>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new action
							</SlButton>
						</Flex>
						<StyledGrid rows={rows} columns={columns} noRowsMessage='No actions to show'></StyledGrid>
					</>
				)}
			</div>
			<SlDialog
				label='Add action'
				className='actionDialog'
				onSlAfterHide={() => setIsDialogOpen(false)}
				open={isDialogOpen}
				style={{ '--width': '600px' }}
			>
				<div className='actionTypes'>
					{standardActionTypes}
					{eventActionTypes.length > 0 && (
						<>
							<div className='eventActionTypesTitle'>Events</div>
							{eventActionTypes}
						</>
					)}
				</div>
			</SlDialog>
		</>
	);
};

export default ConnectionActions;
