import React, { useState, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import statuses from '../../../constants/statuses';
import { NotFoundError } from '../../../lib/api/errors';
import DangerZone from '../../shared/DangerZone/DangerZone';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import Flex from '../../shared/Flex/Flex';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import { ConnectionToSet } from '../../../types/external/connection';

interface GeneralProps {
	connection: TransformedConnection;
	onDelete: () => void;
}

const ConnectionGeneralSettings = ({ connection, onDelete }: GeneralProps) => {
	const [connectionToSet, setConnectionToSet] = useState<ConnectionToSet>({
		name: connection.name,
		enabled: connection.enabled,
		strategy: connection.strategy,
		websiteHost: connection.websiteHost,
		SendingMode: connection.SendingMode,
	});
	const [askDeletionConfirmation, setAskDeletionConfirmation] = useState<boolean>(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError, showStatus, redirect, setIsLoadingConnections } = useContext(AppContext);

	const onNameChange = (e) => {
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

	const onWebsitehostChange = (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.websiteHost = value;
		setConnectionToSet(c);
	};

	const onModeChange = (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.SendingMode = value;
		setConnectionToSet(c);
	};

	const onSwitchChange = () => {
		const c = { ...connectionToSet };
		c.enabled = !c.enabled;
		setConnectionToSet(c);
	};

	const onDeletionConfirmation = async () => {
		try {
			await api.workspaces.connections.delete(connection.id);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			handleError(err);
			return;
		}
		setAskDeletionConfirmation(false);
		setIsLoadingConnections(true);
		onDelete();
	};

	const onSave = async () => {
		setIsSaving(true);
		try {
			await api.workspaces.connections.set(connection.id, connectionToSet);
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

	const showStrategy =
		connection.role === 'Source' && (connection.type === 'Mobile' || connection.type === 'Website');

	return (
		<div className='generalSettings'>
			<SlInput
				label='Name'
				className='nameField'
				value={connectionToSet.name}
				onSlChange={onNameChange}
				maxlength={100}
			/>

			{showStrategy && (
				<SlSelect
					value={connectionToSet.strategy}
					label='Strategy'
					className='strategyField'
					onSlChange={onStrategyChange}
				>
					<SlOption value='AB-C'>AB-C</SlOption>
					<SlOption value='ABC'>ABC</SlOption>
					<SlOption value='A-B-C'>A-B-C</SlOption>
					<SlOption value='AC-B'>AC-B</SlOption>
				</SlSelect>
			)}

			{connection.isDestination && connection.connector.supportedSendingModes.length > 0 && (
				<SlSelect
					value={connectionToSet.SendingMode}
					label='Sending mode'
					className='modeField'
					onSlChange={onModeChange}
				>
					<div className='modeValueIcon' slot='prefix'>
						<SlIcon
							name={
								connectionToSet.SendingMode === 'Cloud'
									? 'cloud'
									: connectionToSet.SendingMode === 'Device'
										? 'phone'
										: 'send'
							}
						/>
					</div>
					{connection.connector.supportedSendingModes.map((m) => (
						<SlOption key={m} value={m}>
							<div slot='prefix'>
								<SlIcon
									className='modeIcon'
									name={m === 'Cloud' ? 'cloud' : m === 'Device' ? 'phone' : 'send'}
								/>
							</div>
							{m}
						</SlOption>
					))}
				</SlSelect>
			)}

			{connection.type === 'Website' && (
				<SlInput
					label='Website host'
					className='websiteHostField'
					value={connectionToSet.websiteHost}
					onSlChange={onWebsitehostChange}
				/>
			)}

			<SlSwitch className='enablingField' onSlChange={onSwitchChange} checked={connectionToSet.enabled}>
				Enable connection
			</SlSwitch>

			<SlButton className='updateButton' variant='primary' loading={isSaving} onClick={onSave}>
				Save
			</SlButton>

			<SlDivider />

			<DangerZone className='dangerZone'>
				<div className='label'>Delete the connection</div>
				<Flex justifyContent='space-between' alignItems='baseline'>
					<div className='description'>Delete permanently the connection</div>
					<SlButton
						className='deleteButton'
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
						<SlButton variant='danger' onClick={onDeletionConfirmation}>
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
