import React, { useContext } from 'react';
import ActionContext from '../../../context/ActionContext';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';

const ActionUpdateOnDuplicates = () => {
	const { action, setAction } = useContext(ActionContext);

	const onChangeUpdateOnDuplicates = (e) => {
		const a = { ...action };
		a.updateOnDuplicates = e.currentTarget.checked;
		setAction(a);
	};

	return (
		action.exportMode.includes('Update') && (
			<div className='action__update-on-duplicates'>
				<SlCheckbox checked={action.updateOnDuplicates!} onSlChange={onChangeUpdateOnDuplicates}>
					If multiple app users match a single user in Meergo, update them anyway
				</SlCheckbox>
			</div>
		)
	);
};

export default ActionUpdateOnDuplicates;
