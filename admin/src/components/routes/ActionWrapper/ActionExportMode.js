import { useContext } from 'react';
import Section from '../../shared/Section/Section';
import { EXPORT_MODE_OPTIONS } from '../../../lib/helpers/action';
import { ActionContext } from '../../../context/ActionContext';
import { SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';

const ActionExportMode = () => {
	const { action, setAction } = useContext(ActionContext);

	const onChangeExportMode = (e) => {
		const a = { ...action };
		a.ExportMode = e.currentTarget.value;
		setAction(a);
	};

	return (
		<Section title='Export Mode' description='The mode used to export the data' padded={true}>
			<SlSelect size='medium' value={action.ExportMode} onSlChange={onChangeExportMode}>
				{Object.keys(EXPORT_MODE_OPTIONS).map((k) => (
					<SlOption value={k}>{EXPORT_MODE_OPTIONS[k]}</SlOption>
				))}
			</SlSelect>
		</Section>
	);
};

export default ActionExportMode;
