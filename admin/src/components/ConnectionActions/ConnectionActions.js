import { useState, useEffect, useContext } from 'react';
import { createPortal } from 'react-dom';
import './ConnectionActions.css';
import LittleLogo from '../LittleLogo/LittleLogo';
import Flex from '../Flex/Flex';
import StyledGrid from '../StyledGrid/StyledGrid';
import EditPage from '../EditPage/EditPage';
import Action from '../Action/Action';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { SlButton, SlIcon, SlDialog, SlSwitch } from '@shoelace-style/shoelace/dist/react/index.js';

let ConnectionActions = () => {
	let [actions, setActions] = useState([]);
	let [actionTypes, setActionTypes] = useState([]);
	let [isDialogOpen, setIsDialogOpen] = useState(false);
	let [selectedActionType, setSelectedActionType] = useState(null);
	let [actionToEdit, setActionToEdit] = useState(null);

	let { API, showError } = useContext(AppContext);
	let { connection: c, setCurrentConnectionSection } = useContext(ConnectionContext);

	setCurrentConnectionSection('actions');

	useEffect(() => {
		const fetchData = async () => {
			let err, actionTypes, actions;
			[actionTypes, err] = await API.connections.actionTypes(c.ID);
			if (err != null) {
				showError(err);
				return;
			}
			setActionTypes(actionTypes);
			[actions, err] = await API.connections.actions(c.ID);
			if (err != null) {
				showError(err);
				return;
			}
			setActions(actions);
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
		let [, err] = await API.connections.deleteAction(c.ID, actionID);
		if (err != null) {
			showError(err);
			return;
		}
		let a = [...actions];
		let filtered = a.filter((a) => a.ID !== actionID);
		setActions(filtered);
	};

	const refreshActions = async () => {
		setSelectedActionType(null);
		setActionToEdit(null);
		let [actions, err] = await API.connections.actions(c.ID);
		if (err != null) {
			showError(err);
			return;
		}
		setActions(actions);
	};

	let columns = [{ Name: 'Name' }, { Name: 'Filter' }, { Name: 'Action type' }, { Name: 'Enabled' }, { Name: '' }];

	// replace boolean with switches components.
	let rows = [];
	for (let a of actions) {
		let name = a.Name;
		let conditions = [];
		for (let c of a.Filter.Conditions) {
			conditions.push(
				<div>
					{c.Property} {c.Operator} {c.Value}
				</div>
			);
		}
		let actionType = actionTypes.find((at) => at.ID === a.ActionType).Name;
		let enabled = <SlSwitch onSlChange={() => onActionStatusSwitch(a.ID)} checked={a.Enabled}></SlSwitch>;
		let actions = (
			<div className='actionButtons'>
				<SlButton className='removeAction' variant='danger' size='small' onClick={() => onRemoveAction(a.ID)}>
					Remove
				</SlButton>
				<SlButton variant='default' size='small' onClick={() => setActionToEdit(a)}>
					Edit
				</SlButton>
			</div>
		);
		rows.push([name, conditions, actionType, enabled, actions]);
	}

	if (selectedActionType !== null) {
		return createPortal(
			<EditPage title={`Add "${selectedActionType.Name}" action`} onCancel={() => setSelectedActionType(null)}>
				<Action actionTypeProp={selectedActionType} onClose={refreshActions}></Action>
			</EditPage>,
			document.body
		);
	}

	if (actionToEdit !== null) {
		return createPortal(
			<EditPage title={`Edit "${actionToEdit.Name}" action`} onCancel={() => setActionToEdit(null)}>
				<Action onClose={refreshActions} actionProp={actionToEdit}></Action>
			</EditPage>,
			document.body
		);
	}

	return (
		<>
			<div className='ConnectionActions'>
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
			</div>
			<SlDialog
				label='Add action'
				className='actionDialog'
				onSlAfterHide={() => setIsDialogOpen(false)}
				open={isDialogOpen}
				style={{ '--width': '700px' }}
			>
				<div className='actionTypes'>
					{actionTypes.map((t) => {
						return (
							<div
								className='actionType'
								onClick={() => {
									setIsDialogOpen(false);
									setSelectedActionType(t);
								}}
							>
								<LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`}></LittleLogo>
								<div className='name'>{t.Name}</div>
								<div className='description'>{t.Description}</div>
							</div>
						);
					})}
				</div>
			</SlDialog>
		</>
	);
};

export default ConnectionActions;
