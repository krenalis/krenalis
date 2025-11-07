import React, { ReactNode } from 'react';
import ListTile from '../../base/ListTile/ListTile';
import { ActionType } from '../../../lib/api/types/action';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import TransformedConnection from '../../../lib/core/connection';

interface ActionTypesDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	actionTypes: ActionType[];
	connection: TransformedConnection;
	connectionLogo: ReactNode;
	onSelectActionType: (actionType: ActionType) => void;
}

const ActionTypesDialog = ({
	isOpen,
	setIsOpen,
	actionTypes,
	connection,
	connectionLogo,
	onSelectActionType,
}: ActionTypesDialogProps) => {
	const standardActionTypes: ReactNode[] = [];
	const eventActionTypes: ReactNode[] = [];
	for (const type of actionTypes) {
		let disablingReason = null;
		if (connection.actions != null && type.target === 'Event' && connection.isSource) {
			let importEventAction = connection.actions.findIndex((a) => a.target === 'Event');
			if (importEventAction > -1) {
				disablingReason = 'You can add only one action that imports events';
			}
		}

		const tile = (
			<ListTile
				key={type.name}
				icon={connectionLogo}
				name={type.name}
				description={type.description}
				disablingReason={disablingReason}
				disabled={disablingReason != null}
				onClick={() => {
					onSelectActionType(type);
				}}
				action={<SlIcon name='chevron-right' />}
			/>
		);
		if (type.target === 'User' || type.target === 'Group') {
			standardActionTypes.push(tile);
		} else {
			eventActionTypes.push(tile);
		}
	}

	return (
		<SlDialog
			label='Add action'
			className='connection-actions__dialog'
			onSlAfterHide={() => setIsOpen(false)}
			open={isOpen}
			style={{ '--width': '600px' } as React.CSSProperties}
		>
			<div className='connection-actions__dialog-action-types'>
				{standardActionTypes}
				{eventActionTypes.length > 0 && (
					<>
						<div className='connection-actions__dialog-event-action-types-title'>Events</div>
						{eventActionTypes}
					</>
				)}
			</div>
		</SlDialog>
	);
};

export default ActionTypesDialog;
