import { useState, useEffect, useContext, useRef } from 'react';
import './ConnectionActions.css';
import Flex from '../../common/Flex/Flex';
import IconWrapper from '../../common/IconWrapper/IconWrapper';
import Fullscreen from '../../common/Fullscreen/Fullscreen';
import Action from './Action/Action';
import ActionsGrid from './ActionsGrid';
import ActionTypesDialog from './ActionTypesDialog';
import { AppContext } from '../../../providers/AppProvider';
import { ConnectionContext } from '../../../providers/ConnectionProvider';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionActions = () => {
	const [selectedActionType, setSelectedActionType] = useState(null);
	const [selectedAction, setSelectedAction] = useState(null);
	const [isActionTypesDialogOpen, setIsActionTypesDialogOpen] = useState(false);

	const { setAreConnectionsStale } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const refreshConnectionIntervalID = useRef(0);

	useEffect(() => {
		const isEditing = selectedAction != null || selectedActionType != null;
		if (isEditing) {
			clearInterval(refreshConnectionIntervalID.current);
			return;
		} else {
			refreshConnectionIntervalID.current = setInterval(async () => {
				setAreConnectionsStale(true);
			}, 1500);
		}
		return () => {
			clearInterval(refreshConnectionIntervalID.current);
		};
	}, [selectedAction, selectedActionType]);

	const onActionTypesDialogClose = async () => {
		setAreConnectionsStale(true);
		setSelectedActionType(null);
		setSelectedAction(null);
	};

	return (
		<>
			<div className='connectionActions'>
				{connection.actions.length === 0 ? (
					<div className='noAction'>
						<IconWrapper name='send-exclamation' size={40} />
						<div className='description'>Add an action to {connection.description}</div>
						<SlButton
							variant='primary'
							onClick={() => {
								setIsActionTypesDialogOpen(true);
							}}
						>
							Add action...
						</SlButton>
					</div>
				) : (
					<>
						<Flex justifyContent={'end'} alignItems={'center'}>
							<SlButton
								variant='text'
								onClick={() => {
									setIsActionTypesDialogOpen(true);
								}}
							>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new action
							</SlButton>
						</Flex>
						<ActionsGrid connection={connection} onSelectAction={setSelectedAction} />
					</>
				)}
			</div>
			<ActionTypesDialog
				isOpen={isActionTypesDialogOpen}
				setIsOpen={setIsActionTypesDialogOpen}
				actionTypes={connection.actionTypes}
				connectionLogo={connection.logo}
				onSelectActionType={(type) => {
					setIsActionTypesDialogOpen(false);
					setSelectedActionType(type);
				}}
			/>
			<Fullscreen isOpen={selectedActionType != null || selectedAction != null}>
				<Action actionType={selectedActionType} action={selectedAction} onClose={onActionTypesDialogClose} />
			</Fullscreen>
		</>
	);
};

export default ConnectionActions;
