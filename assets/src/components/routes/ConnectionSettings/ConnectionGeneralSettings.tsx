import React, { useState, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import TransformedConnection from '../../../lib/core/connection';
import { NotFoundError } from '../../../lib/api/errors';
import DangerZone from '../../base/DangerZone/DangerZone';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import Flex from '../../base/Flex/Flex';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import { ConnectionToSet } from '../../../lib/api/types/connection';

interface GeneralProps {
	connection: TransformedConnection;
	onDelete: () => void;
}

const ConnectionGeneralSettings = ({ connection, onDelete }: GeneralProps) => {
	const [connectionToSet, setConnectionToSet] = useState<ConnectionToSet>({
		name: connection.name,
		strategy: connection.strategy,
		sendingMode: connection.sendingMode,
	});
	const [askDeletionConfirmation, setAskDeletionConfirmation] = useState<boolean>(false);
	const [isDeleting, setIsDeleting] = useState<boolean>(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError, redirect, setIsLoadingConnections } = useContext(AppContext);

	const onNameInput = (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.name = value;
		setConnectionToSet(c);
	};

	const onStrategyChange = (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.strategy = value;
		setConnectionToSet(c);
	};

	const onModeChange = (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.sendingMode = value;
		setConnectionToSet(c);
	};

	const onDeletionConfirmation = async () => {
		setIsDeleting(true);
		try {
			await api.workspaces.connections.delete(connection.id);
		} catch (err) {
			setIsDeleting(false);
			if (err instanceof NotFoundError) {
				redirect('connections');
				handleError('The connection does not exist anymore');
				return;
			}
			handleError(err);
			return;
		}
		setTimeout(() => {
			setAskDeletionConfirmation(false);
			setIsDeleting(false);
			setIsLoadingConnections(true);
			onDelete();
		}, 300);
	};

	const onSave = async () => {
		setIsSaving(true);
		try {
			await api.workspaces.connections.update(connection.id, connectionToSet);
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsSaving(false);
			}, 500);
			return;
		}
		setTimeout(() => {
			setIsSaving(false);
			setIsLoadingConnections(true);
		}, 500);
	};

	const showStrategy = connection.role === 'Source' && connection.connector.strategies;

	return (
		<div className='connection-settings__general-settings'>
			<SlInput
				label='Name'
				className='connection-settings__name-field'
				value={connectionToSet.name}
				onSlInput={onNameInput}
				maxlength={100}
			/>

			{showStrategy && (
				<SlSelect
					value={connectionToSet.strategy}
					label='Strategy'
					className='connection-settings__strategy-field'
					onSlChange={onStrategyChange}
				>
					<SlOption value='Conversion'>Conversion strategy</SlOption>
					<SlOption value='Fusion'>Fusion strategy</SlOption>
					<SlOption value='Isolation'>Isolation strategy</SlOption>
					<SlOption value='Preservation'>Preservation strategy</SlOption>
				</SlSelect>
			)}

			{connection.isDestination && connection.connector.supportedSendingModes.length > 0 && (
				<SlSelect
					value={connectionToSet.sendingMode}
					label='Sending mode'
					className='connection-settings__mode-field'
					onSlChange={onModeChange}
				>
					<div className='connection-settings__mode-value-icon' slot='prefix'>
						<SlIcon
							name={
								connectionToSet.sendingMode === 'Cloud'
									? 'cloud'
									: connectionToSet.sendingMode === 'Device'
										? 'phone'
										: 'send'
							}
						/>
					</div>
					{connection.connector.supportedSendingModes.map((m) => (
						<SlOption key={m} value={m}>
							<div slot='prefix'>
								<SlIcon
									className='connection-settings__mode-icon'
									name={m === 'Cloud' ? 'cloud' : m === 'Device' ? 'phone' : 'send'}
								/>
							</div>
							{m}
						</SlOption>
					))}
				</SlSelect>
			)}

			<SlButton
				className='connection-settings__update-button'
				variant='primary'
				loading={isSaving}
				onClick={onSave}
			>
				Save
			</SlButton>

			<SlDivider />

			<DangerZone className='connection-settings__danger-zone-field'>
				<div className='connection-settings__danger-zone-label'>Delete the connection</div>
				<Flex justifyContent='space-between' alignItems='baseline'>
					<div className='connection-settings__danger-zone-description'>
						Delete permanently the connection
					</div>
					<SlButton
						className='connection-settings__danger-zone-delete-button'
						variant='danger'
						onClick={() => setAskDeletionConfirmation(true)}
					>
						Delete
					</SlButton>
				</Flex>
			</DangerZone>

			<AlertDialog
				variant='danger'
				isOpen={askDeletionConfirmation}
				onClose={() => setAskDeletionConfirmation(false)}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={() => setAskDeletionConfirmation(false)}>Cancel</SlButton>
						<SlButton variant='danger' onClick={onDeletionConfirmation} loading={isDeleting}>
							Delete
						</SlButton>
					</>
				}
			>
				<p>If you continue, you will permanently lose all the connection data</p>
			</AlertDialog>
		</div>
	);
};

export default ConnectionGeneralSettings;
