import ListTile from '../../common/ListTile/ListTile';
import { SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ActionTypesDialog = ({ isOpen, setIsOpen, actionTypes, connectionLogo, onSelectActionType }) => {
	const standardActionTypes = [];
	const eventActionTypes = [];
	for (const type of actionTypes) {
		const tile = (
			<ListTile
				icon={connectionLogo}
				name={type.Name}
				description={type.Description}
				missingSchema={type.MissingSchema}
				onClick={() => {
					onSelectActionType(type);
				}}
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
			className='actionDialog'
			onSlAfterHide={() => setIsOpen(false)}
			open={isOpen}
			style={{ '--width': '600px' }}
		>
			<div className='actionTypes'>
				{standardActionTypes}
				{eventActionTypes.length > 0 && (
					<>
						<div className='eventActionTypesTitle'>Events</div>
						{eventActionTypes}
					</>
				)}
			</div>
		</SlDialog>
	);
};

export default ActionTypesDialog;
