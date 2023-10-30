import React, { useState, useEffect, useContext, useRef, useLayoutEffect } from 'react';
import './ConnectionActions.css';
import Flex from '../../shared/Flex/Flex';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import ActionsGrid from './ActionsGrid';
import ListTile from '../../shared/ListTile/ListTile';
import ActionTypesDialog from './ActionTypesDialog';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { Outlet } from 'react-router-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { Action, ActionType } from '../../../types/external/action';
import getConnectorLogo from '../../helpers/getConnectorLogo';

const ConnectionActions = () => {
	const [isActionTypesDialogOpen, setIsActionTypesDialogOpen] = useState<boolean>(false);
	const [isActionOpen, setIsActionOpen] = useState<boolean>(false);
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { setAreConnectionsStale, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const refreshConnectionIntervalID = useRef<number>(0);
	const newActionID = useRef<number>(0);

	useLayoutEffect(() => {
		const isNew = window.location.search.indexOf('new=true') !== -1;
		if (isNew) {
			setIsLoading(true);
			setTimeout(() => {
				setIsLoading(false);
			}, 1000);
		}
	}, []);

	useEffect(() => {
		if (!isActionOpen) {
			refreshConnectionIntervalID.current = window.setInterval(async () => {
				setAreConnectionsStale(true);
			}, 1000);

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

	const onSelectActionType = (actionType: ActionType) => {
		let name: string;
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
		redirect(newLocation);
	};

	const onSelectAction = (action: Action) => {
		const newLocation = `connections/${connection.id}/actions/edit/${action.ID}`;
		redirect(newLocation);
	};

	if (isLoading) {
		return (
			<SlSpinner
				style={
					{
						display: 'block',
						position: 'relative',
						top: '50px',
						margin: 'auto',
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></SlSpinner>
		);
	}

	return (
		<>
			<div className='connectionActions'>
				{connection.actions!.length === 0 ? (
					<div className='noAction'>
						<IconWrapper name='send-exclamation' size={40} />
						<div className='description'>Add an action to {connection.description}</div>
						<div className='actionTypes'>
							{connection.actionTypes.map((actionType) => (
								<ListTile
									key={actionType.Name}
									icon={getConnectorLogo(connection.connector.icon)}
									name={actionType.Name}
									description={actionType.Description}
									disabled={actionType.MissingSchema}
									className='actionType'
									disablingReason={actionType.MissingSchema ? 'Missing schema' : ''}
									action={
										<SlButton
											size='small'
											variant='primary'
											onClick={() => {
												onSelectActionType(actionType);
											}}
										>
											Add
										</SlButton>
									}
								/>
							))}
						</div>
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
							actions={connection.actions!}
							onSelectAction={onSelectAction}
						/>
					</>
				)}
			</div>
			<ActionTypesDialog
				isOpen={isActionTypesDialogOpen}
				setIsOpen={setIsActionTypesDialogOpen}
				actionTypes={connection.actionTypes!}
				connectionLogo={getConnectorLogo(connection.connector.icon)}
				onSelectActionType={onSelectActionType}
			/>
			<Outlet context={{ setIsActionOpen }} />
		</>
	);
};

export default ConnectionActions;
