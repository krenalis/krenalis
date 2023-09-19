import React, { useState, useContext } from 'react';
import Flex from '../../shared/Flex/Flex';
import statuses from '../../../constants/statuses';
import { AppContext } from '../../../context/providers/AppProvider';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import { NotFoundError } from '../../../lib/api/errors';
import TransformedConnection from '../../../lib/helpers/transformedConnection';

interface DeletionProps {
	connection: TransformedConnection;
	onDelete: () => void;
}

const Deletion = ({ connection: c, onDelete }: DeletionProps) => {
	const [askDeletionConfirmation, setAskDeletionConfirmation] = useState<boolean>(false);

	const { api, showError, showStatus, redirect, setAreConnectionsStale } = useContext(AppContext);

	const onDeletionConfirmation = async () => {
		try {
			await api.connections.delete(c.id);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		setAskDeletionConfirmation(false);
		setAreConnectionsStale(true);
		onDelete();
	};

	return (
		<>
			<fieldset className='dangerZone'>
				<div className='heading'>Danger zone</div>
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
			</fieldset>
			<SlDialog
				className='deletionDialog'
				open={askDeletionConfirmation}
				style={{ '--width': '600px' } as React.CSSProperties}
				onSlAfterHide={() => setAskDeletionConfirmation(false)}
				label={`Are you sure you want to remove ${c.name}?`}
			>
				<div className='buttons'>
					<SlButton variant='neutral' onClick={() => setAskDeletionConfirmation(false)}>
						Cancel
					</SlButton>
					<SlButton variant='danger' onClick={onDeletionConfirmation}>
						Remove
					</SlButton>
				</div>
			</SlDialog>
		</>
	);
};

export default Deletion;
