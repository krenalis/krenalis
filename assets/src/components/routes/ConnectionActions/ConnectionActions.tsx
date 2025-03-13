import React, { useState, useContext, useRef, useLayoutEffect } from 'react';
import './ConnectionActions.css';
import Flex from '../../base/Flex/Flex';
import ActionsGrid from './ActionsGrid';
import ListTile from '../../base/ListTile/ListTile';
import ActionTypesDialog from './ActionTypesDialog';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { Outlet } from 'react-router-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { Action, ActionType } from '../../../lib/api/types/action';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { LinkedConnections } from '../ConnectionSettings/LinkedConnections';
import { isEventConnection } from '../../../lib/core/connection';
import Section from '../../base/Section/Section';
import { Snippet } from '../../base/Snippet/Snippet';

const ConnectionActions = () => {
	const [isActionTypesDialogOpen, setIsActionTypesDialogOpen] = useState<boolean>(false);
	const [isActionOpen, setIsActionOpen] = useState<boolean>(false);
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const newActionID = useRef<number>(0);

	useLayoutEffect(() => {
		const isNew = window.location.search.indexOf('new=true') !== -1;
		if (isNew) {
			setIsLoading(true);
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		}
	}, []);

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
		if (actionType.target === 'Events') {
			if (actionType.eventType) {
				name = `event/${actionType.eventType}`;
			} else {
				name = 'event';
			}
		} else {
			name = actionType.target.toLowerCase();
		}
		const newLocation = `connections/${connection.id}/actions/add/${name}`;
		setIsActionTypesDialogOpen(false);
		redirect(newLocation);
	};

	const onSelectAction = (action: Action) => {
		const newLocation = `connections/${connection.id}/actions/edit/${action.id}`;
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

	let linkedConnections = (
		<Section
			title={connection.isSource ? 'Event destinations' : 'Event sources'}
			description={
				<>
					{connection.isSource
						? 'Select which destinations should receive events from this source.'
						: 'Select which sources should send events to this destination.'}
					<br />
					{connection.isSource
						? 'When you link a destination connection here, events from this source will automatically be forwarded to that destination and processed by its actions'
						: 'When you link a source connection here, events from that source will automatically be forwarded to this destination and processed by its actions'}
				</>
			}
			annotated={true}
			className={
				connection.isSource ? 'connection-actions__linked-destinations' : 'connection-actions__linked-sources'
			}
		>
			<LinkedConnections connection={connection} />
		</Section>
	);

	return (
		<div
			className={`connection-actions${connection.actions!.length === 0 ? ' connection-actions--no-action' : ''}`}
		>
			{connection.isWebsite && (
				<Section
					title='Add Meergo to your website'
					className='connection-actions__instructions'
					description={
						<div className='connection-actions__instructions-text'>
							Copy this snippet and paste it into your website to receive events
							<a target='_blank' href='http://localhost:8080/developers/javascript-sdk'>
								See documentation
							</a>
						</div>
					}
					annotated={true}
				>
					<Snippet connectionID={connection.id} />
				</Section>
			)}

			{/* Linked connections are shown: before the actions, in the case of destination actions; after the actions,
			in the case of source actions. This is to better suggest the usability flow. */}
			{connection.isDestination &&
				isEventConnection(
					'Destination',
					connection.connector.type,
					connection.connector.asDestination.targets,
				) &&
				linkedConnections}

			<Section
				className='connection-actions__list'
				title='Actions'
				description={`Actions import events, users, and groups from a website into the workspace's data warehouse using ${connection.name}`}
				annotated={true}
			>
				{connection.actions!.length === 0 ? (
					<div className='connection-actions__no-action'>
						<div className='connection-actions__no-action-action-types'>
							{connection.actionTypes.map((actionType) => (
								<ListTile
									key={actionType.name}
									icon={getConnectorLogo(connection.connector.icon)}
									name={actionType.name}
									description={actionType.description}
									className={`connection-actions__action-type connection-actions__action-type--${actionType.target.toLowerCase()}`}
									action={
										<SlButton
											size='small'
											variant='primary'
											onClick={() => {
												onSelectActionType(actionType);
											}}
										>
											Add...
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
								className='connection-actions__add'
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
			</Section>

			{/* Linked connections are shown: before the actions, in the case of destination actions; after the actions,
			in the case of source actions. This is to better suggest the usability flow. */}
			{connection.isSource &&
				isEventConnection('Source', connection.connector.type, connection.connector.asSource.targets) &&
				linkedConnections}

			<ActionTypesDialog
				isOpen={isActionTypesDialogOpen}
				setIsOpen={setIsActionTypesDialogOpen}
				actionTypes={connection.actionTypes!}
				connection={connection}
				connectionLogo={getConnectorLogo(connection.connector.icon)}
				onSelectActionType={onSelectActionType}
			/>
			<Outlet context={{ setIsActionOpen }} />
		</div>
	);
};

export default ConnectionActions;
