import { useState, useLayoutEffect, useContext } from 'react';
import Action from './Action';
import Fullscreen from '../../shared/Fullscreen/Fullscreen';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { useParams, useLocation, useOutletContext } from 'react-router-dom';

const ActionWrapper = () => {
	const [selectedActionType, setSelectedActionType] = useState(null);
	const [selectedAction, setSelectedAction] = useState(null);

	const params = useParams();
	const location = useLocation();

	const { setIsActionOpen } = useOutletContext();
	const { setAreConnectionsStale, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useLayoutEffect(() => {
		const splitted = location.pathname.split('/');
		const instructionsStartIndex = splitted.findIndex((fragment) => fragment === 'actions') + 1;
		const instructions = splitted.slice(instructionsStartIndex);
		const isEditing = instructions[0] === 'edit';
		if (isEditing) {
			const action = connection.actions.find((a) => String(a.ID) === params.action);
			setSelectedAction(action);
			return;
		} else {
			let actionType;
			const isEvent = instructions.includes('event');
			if (isEvent) {
				if (instructions.length === 3) {
					const eventType = instructions[instructions.length - 1];
					actionType = connection.actionTypes.find((a) => a.EventType === eventType);
				} else {
					actionType = connection.actionTypes.find((a) => a.Target === 'Events' && a.EventType === null);
				}
			} else {
				const target = instructions[instructions.length - 1];
				const capitalized = target.charAt(0).toUpperCase() + target.slice(1);
				actionType = connection.actionTypes.find((a) => a.Target === capitalized);
			}
			setSelectedActionType(actionType);
			return;
		}
	}, [params, location]);

	const onClose = () => {
		setAreConnectionsStale(true);
		redirect(`connections/${connection.id}/actions`);
		setIsActionOpen(false);
	};

	return (
		<Fullscreen onClose={onClose}>
			<Action actionType={selectedActionType} action={selectedAction} />
		</Fullscreen>
	);
};

export default ActionWrapper;
