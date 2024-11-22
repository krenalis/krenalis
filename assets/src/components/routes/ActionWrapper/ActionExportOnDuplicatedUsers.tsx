import React, { useContext } from 'react';
import Section from '../../base/Section/Section';
import ActionContext from '../../../context/ActionContext';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';

const ActionExportOnDuplicatedUsers = () => {
	const { action, setAction } = useContext(ActionContext);
	const onChangeExportOnDuplicatedUsers = (e) => {
		const a = { ...action };
		a.ExportOnDuplicatedUsers = e.currentTarget.checked;
		setAction(a);
	};
	return (
		<Section
			title='Export on duplicated users'
			className='action__export-on-duplicated'
			description='Determine the behavior in case of duplicated users on the app, which are users which have the same value for the specified property'
			padded={true}
			annotated={true}
		>
			<SlCheckbox checked={action.ExportOnDuplicatedUsers!} onSlChange={onChangeExportOnDuplicatedUsers}>
				Run the export even in case of duplicated users on the app, instead of not starting the export
			</SlCheckbox>
		</Section>
	);
};

export default ActionExportOnDuplicatedUsers;
