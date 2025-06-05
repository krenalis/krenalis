import React, { useState, useEffect, useContext } from 'react';
import Action from './Action';
import Fullscreen from '../../base/Fullscreen/Fullscreen';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { useParams, useLocation, useOutletContext } from 'react-router-dom';
import { Action as ActionInterface, ActionType } from '../../../lib/api/types/action';

const ActionWrapper = () => {
	const [selectedActionType, setSelectedActionType] = useState<ActionType>();
	const [selectedAction, setSelectedAction] = useState<ActionInterface>();

	const params = useParams();
	const location = useLocation();

	const { setIsActionOpen } = useOutletContext<any>();
	const { setIsLoadingConnections, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useEffect(() => {
		setIsActionOpen(true);
	}, []);

	useEffect(() => {
		const splitted = location.pathname.split('/');
		const instructionsStartIndex = splitted.findIndex((fragment) => fragment === 'actions') + 1;
		const instructions = splitted.slice(instructionsStartIndex);
		const isEditing = instructions[0] === 'edit';
		if (isEditing) {
			const action = connection.actions!.find((a) => String(a.id) === params.action);
			setSelectedAction(action);
			return;
		} else {
			let actionType: ActionType | undefined;
			const isEvent = instructions.includes('event');
			if (isEvent) {
				if (instructions.length === 3) {
					const eventType = instructions[instructions.length - 1];
					actionType = connection.actionTypes!.find((a) => a.eventType === eventType);
				} else {
					actionType = connection.actionTypes!.find((a) => a.target === 'Event' && a.eventType === null);
				}
			} else {
				const target = instructions[instructions.length - 1];
				const capitalized = target.charAt(0).toUpperCase() + target.slice(1);
				actionType = connection.actionTypes!.find((a) => a.target === capitalized);
			}
			if (actionType == null) {
				console.error(`Action type for instructions ${instructions} does not exist anymore`);
				return;
			}
			setSelectedActionType(actionType);
			return;
		}
	}, [params, location]);

	const onClose = () => {
		setIsLoadingConnections(true);
		redirect(`connections/${connection.id}/actions`);
		setIsActionOpen(false);
	};

	const isLoading = selectedActionType == null && selectedAction == null;
	return (
		<Fullscreen onClose={onClose} isLoading={isLoading}>
			<Action actionType={selectedActionType} action={selectedAction} />
		</Fullscreen>
	);
};

export default ActionWrapper;
