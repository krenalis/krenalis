import { useState } from 'react';
import './ConnectionDeletion.css';
import FlexContainer from '../FlexContainer/FlexContainer';
import call from '../../utils/call';
import { SlButton, SlDialog, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionDeletion = ({ connection: c, onDelete, onError }) => {
	let [askDeletionConfirmation, setAskDeletionConfirmation] = useState(false);

	const onDeletionConfirmation = async () => {
		let [, err] = await call('/admin/connections/delete', 'POST', [c.ID]);
		if (err !== null) {
			onError(err);
			return;
		}
		setAskDeletionConfirmation(false);
		onDelete();
	};

	return (
		<>
			<div className='panelTitle'>Deletion</div>
			<fieldset className='dangerZone'>
				<div className='heading'>Danger zone</div>
				<div className='label'>Delete the connection</div>
				<FlexContainer justifyContent='space-between' alignItems='baseline'>
					<div className='description'>Delete permanently the connection</div>
					<SlButton
						className='deleteButton'
						variant='danger'
						onClick={() => setAskDeletionConfirmation(true)}
					>
						<SlIcon slot='suffix' name='trash3' />
						Delete
					</SlButton>
				</FlexContainer>
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
