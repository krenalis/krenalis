import { useState, useEffect, useContext, useRef, useLayoutEffect } from 'react';
import './ConnectionActions.css';
import Flex from '../../shared/Flex/Flex';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import ActionsGrid from './ActionsGrid';
import ActionTypesDialog from './ActionTypesDialog';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { Outlet } from 'react-router-dom';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionActions = () => {
	const [isActionTypesDialogOpen, setIsActionTypesDialogOpen] = useState(false);
	const [isActionOpen, setIsActionOpen] = useState(false);

	const { setAreConnectionsStale, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const refreshConnectionIntervalID = useRef(0);
	const newActionID = useRef(0);

	useEffect(() => {
		if (!isActionOpen) {
			refreshConnectionIntervalID.current = setInterval(async () => {
				setAreConnectionsStale(true);
			}, 1500);

			return () => {
				clearInterval(refreshConnectionIntervalID.current);
			};
		} else {
			clearInterval(refreshConnectionIntervalID.current);
		}
	}, [isActionOpen]);

	useLayoutEffect(() => {
		if (!isActionOpen) {
			const id = sessionStorage.getItem('newActionID');
			if (id && id !== '') {
				newActionID.current = Number(id);
				sessionStorage.removeItem('newActionID');
			}
		}
	}, [isActionOpen]);

	const onSelectActionType = (actionType) => {
		let name;
		if (actionType.Target === 'Events') {
			if (actionType.EventType) {
				name = `event/${actionType.EventType}`;
			} else {
				name = 'event';
			}
		} else {
			name = actionType.Target.toLowerCase();
		}
		const newLocation = `connections/${connection.id}/actions/add/${name}`;
		setIsActionTypesDialogOpen(false);
		setIsActionOpen(true);
		redirect(newLocation);
	};

	const onSelectAction = (action) => {
		const newLocation = `connections/${connection.id}/actions/edit/${action.ID}`;
		setIsActionOpen(true);
		redirect(newLocation);
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
						<Flex alignItems={'center'}>
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
						<ActionsGrid
							newActionID={newActionID}
							actions={connection.actions}
							onSelectAction={onSelectAction}
						/>
					</>
				)}
			</div>
			<ActionTypesDialog
				isOpen={isActionTypesDialogOpen}
				setIsOpen={setIsActionTypesDialogOpen}
				actionTypes={connection.actionTypes}
				connectionLogo={connection.logo}
				onSelectActionType={onSelectActionType}
			/>
			<Outlet context={{ setIsActionOpen }} />
		</>
	);
};

export default ConnectionActions;
