import React, { useContext } from 'react';
import Section from '../../shared/Section/Section';
import { EXPORT_MODE_OPTIONS } from '../../../lib/helpers/transformedAction';
import ActionContext from '../../../context/ActionContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';

const ActionExportMode = () => {
	const { action, setAction } = useContext(ActionContext);

	const onChangeExportMode = (e) => {
		const a = { ...action };
		a.ExportMode = e.currentTarget.value;
		setAction(a);
	};

	return (
		<Section title='Export Mode' description='The mode used to export the data' padded={true}>
			<SlSelect size='medium' value={action.ExportMode!} onSlChange={onChangeExportMode}>
				{Object.keys(EXPORT_MODE_OPTIONS).map((k) => (
					<SlOption key={k} value={k}>
						{EXPORT_MODE_OPTIONS[k]}
					</SlOption>
				))}
			</SlSelect>
		</Section>
	);
};

export default ActionExportMode;
