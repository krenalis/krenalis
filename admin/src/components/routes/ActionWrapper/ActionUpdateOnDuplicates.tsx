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
					If a single profile in Meergo matches multiple app users, update them anyway
				</SlCheckbox>
			</div>
		)
	);
};

export default ActionUpdateOnDuplicates;
