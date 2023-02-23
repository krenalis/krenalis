import { useState, useContext } from 'react';
import './ConnectionDeletion.css';
import Flex from '../Flex/Flex';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { SlButton, SlDialog, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NotFoundError } from '../../api/errors';

const ConnectionDeletion = ({ connection: c, onDelete }) => {
	let [askDeletionConfirmation, setAskDeletionConfirmation] = useState(false);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	const onDeletionConfirmation = async () => {
		let [, err] = await API.connections.delete(c.ID);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		setAskDeletionConfirmation(false);
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
						<SlIcon slot='suffix' name='trash3' />
						Delete
					</SlButton>
				</Flex>
			</fieldset>
			<SlDialog
				className='deletionDialog'
				open={askDeletionConfirmation}
				style={{ '--width': '600px' }}
				onSlAfterHide={() => setAskDeletionConfirmation(false)}
				label={`Are you sure you want to remove ${c.Name}?`}
			>
				<div className='buttons'>
					<SlButton variant='neutral' onClick={() => setAskDeletionConfirmation(false)}>
						<SlIcon slot='suffix' name='x-lg' />
						Cancel
					</SlButton>
					<SlButton variant='danger' onClick={onDeletionConfirmation}>
						<SlIcon slot='suffix' name='trash3' />
						Remove
					</SlButton>
				</div>
			</SlDialog>
		</>
	);
};

export default ConnectionDeletion;
