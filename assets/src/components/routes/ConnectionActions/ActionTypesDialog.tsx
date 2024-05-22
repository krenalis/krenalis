import React, { ReactNode } from 'react';
import ListTile from '../../base/ListTile/ListTile';
import { ActionType } from '../../../lib/api/types/action';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface ActionTypesDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	actionTypes: ActionType[];
	connectionLogo: ReactNode;
	onSelectActionType: (actionType: ActionType) => void;
}

const ActionTypesDialog = ({
	isOpen,
	setIsOpen,
	actionTypes,
	connectionLogo,
	onSelectActionType,
}: ActionTypesDialogProps) => {
	const standardActionTypes: ReactNode[] = [];
	const eventActionTypes: ReactNode[] = [];
	for (const type of actionTypes) {
		const tile = (
			<ListTile
				key={type.Name}
				icon={connectionLogo}
				name={type.Name}
				description={type.Description}
				onClick={() => {
					onSelectActionType(type);
				}}
				action={<SlIcon name='chevron-right' />}
			/>
		);
		if (type.Target === 'Users' || type.Target === 'Groups') {
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
