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
		storage: connection.storage,
		compression: connection.compression,
		websiteHost: connection.websiteHost,
		businessID: connection.businessID,
	});
	const [askDeletionConfirmation, setAskDeletionConfirmation] = useState<boolean>(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError, showStatus, redirect, setIsLoadingConnections, connections } = useContext(AppContext);

	const onNameChange = async (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.name = value;
		setConnectionToSet(c);
	};

	const onCompressionChange = async (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.compression = value;
		setConnectionToSet(c);
	};

	const onWebsitehostChange = async (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.websiteHost = value;
		setConnectionToSet(c);
	};

	const onBusinessIDNameChange = async (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.businessID.Name = value;
		setConnectionToSet(c);
	};

	const onBusinessIDLabelChange = async (e) => {
		const value = e.target.value;
		const c = { ...connectionToSet };
		c.businessID.Label = value;
		setConnectionToSet(c);
	};

	const onSwitchChange = async () => {
		const c = { ...connectionToSet };
		c.enabled = !c.enabled;
		setConnectionToSet(c);
	};

	const onStorageChange = async (e) => {
		const v = Number(e.target.value);
		const c = { ...connectionToSet };
		c.storage = v;
		if (v === 0) {
			c.compression = '';
		}
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

	const storages: TransformedConnection[] = [];
	for (const cn of connections) {
		if (cn.type === 'Storage' && cn.role === connection.role) {
			storages.push(cn);
		}
	}

	const showBusinessID =
		connection.role === 'Source' && connection.type !== 'Storage' && connection.type !== 'Stream';
	const businessIDKind = ['File', 'Database'].includes(connection.type) ? 'column' : 'property';

	return (
		<div className='generalSettings'>
			<SlInput
				label='Name'
				className='nameField'
				value={connectionToSet.name}
				onSlChange={onNameChange}
				maxlength={100}
			/>

			{connection.type === 'File' && (
				<SlSelect
					label='Storage'
					className='storageField'
					value={String(connectionToSet.storage)}
					onSlChange={onStorageChange}
				>
					<SlOption value='0'>No storage</SlOption>
					{storages.map((s) => (
						<SlOption key={s.id} value={String(s.id)}>
							{s.name}
						</SlOption>
					))}
				</SlSelect>
			)}

			{connection.type === 'File' && (
				<SlSelect
					value={connectionToSet.compression}
					label='Compression'
					className='compressionField'
					disabled={connectionToSet.storage === 0}
					onSlChange={onCompressionChange}
				>
					<SlOption value=''>None</SlOption>
					<SlOption value='Zip'>Zip</SlOption>
					<SlOption value='Gzip'>Gzip</SlOption>
					<SlOption value='Snappy'>Snappy</SlOption>
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

			{showBusinessID && (
				<>
					<SlInput
						label='Business ID Name'
						className='nameField'
						helpText={`The name of the ${businessIDKind} from which the Business ID is read when importing. Can be left empty to indicate to not import it.`}
						placeholder='Something like "email", "customer_id", etc...'
						value={connectionToSet.businessID.Name}
						onSlChange={onBusinessIDNameChange}
						maxlength={1024}
					/>
					<SlInput
						label='Business ID Label'
						className='nameField'
						placeholder='Something like "Email", "Customer ID", etc...'
						helpText='A human-readable label for the Business ID. Mandatory when a Business ID name is specified.'
						value={connectionToSet.businessID.Label}
						onSlChange={onBusinessIDLabelChange}
						maxlength={16}
					/>
				</>
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
