import React, { useContext } from 'react';
import ActionContext from '../../../context/ActionContext';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';

const ActionExportOnDuplicates = () => {
	const { action, setAction } = useContext(ActionContext);

	const onChangeExportOnDuplicates = (e) => {
		const a = { ...action };
		a.exportOnDuplicates = e.currentTarget.checked;
		setAction(a);
	};

	return (
		action.exportMode.includes('Update') && (
			<div className='action__export-on-duplicates'>
				<SlCheckbox checked={action.exportOnDuplicates!} onSlChange={onChangeExportOnDuplicates}>
					If multiple app users match a single user, update them anyway
				</SlCheckbox>
			</div>
		)
	);
};

export default ActionExportOnDuplicates;
