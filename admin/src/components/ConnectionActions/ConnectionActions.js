import { useState, useEffect, useContext } from 'react';
import { createPortal } from 'react-dom';
import './ConnectionActions.css';
import LittleLogo from '../LittleLogo/LittleLogo';
import Flex from '../Flex/Flex';
import ListTile from '../ListTile/ListTile';
import IconWrapper from '../IconWrapper/IconWrapper';
import StyledGrid from '../StyledGrid/StyledGrid';
import EditPage from '../EditPage/EditPage';
import Action from '../Action/Action';
import statuses from '../../constants/statuses';
import { UnprocessableError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { SlButton, SlIcon, SlDialog, SlSwitch, SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

let ConnectionActions = () => {
	let [actions, setActions] = useState([]);
	let [actionTypes, setActionTypes] = useState([]);
	let [isDialogOpen, setIsDialogOpen] = useState(false);
	let [selectedActionType, setSelectedActionType] = useState(null);
	let [actionToEdit, setActionToEdit] = useState(null);
	let [description, setDescription] = useState(null);
	let [isLoading, setIsLoading] = useState(true);

	let { API, showError, showStatus } = useContext(AppContext);
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

	const refresh = async () => {
		setSelectedActionType(null);
		setActionToEdit(null);
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

	let columns = [{ Name: 'Name' }, { Name: 'Filter' }, { Name: 'Type' }, { Name: 'Enabled' }, { Name: '' }];

	let rows = [];
	for (let a of actions) {
		let name = a.Name;
		let conditions = [];
		if (a.Filter != null) {
			for (let c of a.Filter.Conditions) {
				conditions.push(
					<div>
						{c.Property} {c.Operator} {c.Value}
					</div>
				);
			}
		}
		let actionType;
		for (let t of actionTypes) {
			if (a.Target === 'Users' || a.Target === 'Groups') {
				if (a.Target === t.Target) actionType = t.Name;
				continue;
			}
			if (a.EventType === t.EventType) actionType = t.Name;
		}
		let enabled = <SlSwitch onSlChange={() => onActionStatusSwitch(a.ID)} checked={a.Enabled}></SlSwitch>;
		let actions = (
			<div className='actionButtons'>
				{(a.Target === 'Users' || a.Target === 'Groups') && (
					<SlButton variant='default' size='small' onClick={() => executeAction(a.ID)}>
						<SlIcon slot='prefix' name='play' />
						Execute
					</SlButton>
				)}
				<SlButton variant='default' size='small' onClick={() => setActionToEdit(a)}>
					<SlIcon slot='prefix' name='pencil' />
					Edit
				</SlButton>
				<SlButton className='removeAction' variant='danger' size='small' onClick={() => onRemoveAction(a.ID)}>
					<SlIcon slot='prefix' name='trash' />
					Remove
				</SlButton>
			</div>
		);
		rows.push([name, conditions, actionType, enabled, actions]);
	}

	if (selectedActionType !== null) {
		return (
			<EditPage title={`Add "${selectedActionType.Name}" action`} onCancel={() => setSelectedActionType(null)}>
				<Action actionType={selectedActionType} onClose={refresh}></Action>
			</EditPage>
		);
	}

	if (actionToEdit !== null) {
		return (
			<EditPage title={`Edit "${actionToEdit.Name}" action`} onCancel={() => setActionToEdit(null)}>
				<Action action={actionToEdit} onClose={refresh}></Action>
			</EditPage>
		);
	}

	let standardActionTypes = [];
	let eventActionTypes = [];
	for (let t of actionTypes) {
		let icon;
		if (t.Target === 'Users') {
			icon = <SlIcon name='person' />;
		}
		if (t.Target === 'Groups') {
			icon = <SlIcon name='people' />;
		}
		if (t.Target === 'Events') {
			icon = <LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`}></LittleLogo>;
		}
		let tile = (
			<ListTile
				icon={icon}
				name={t.Name}
				description={t.Description}
				disabled={t.Disabled}
				disablingReason={t.DisablingReason}
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
								variant='neutral'
								onClick={() => {
									setIsDialogOpen(true);
								}}
							>
								<SlIcon name='plus-lg' slot='prefix'></SlIcon>
								Add action
							</SlButton>
						</div>
					</>
				) : (
					<>
						<Flex justifyContent={'end'} alignItems={'center'}>
							<SlButton
								variant='default'
								onClick={() => {
									setIsDialogOpen(true);
								}}
							>
								<SlIcon name='plus-lg' slot='prefix'></SlIcon>
								Add action
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
