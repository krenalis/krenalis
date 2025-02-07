import React, { useContext, useMemo } from 'react';
import { EXPORT_MODE_OPTIONS, flattenSchema } from '../../../lib/core/action';
import ActionContext from '../../../context/ActionContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';

const ActionExportMode = () => {
	const { action, setAction, actionType, connection } = useContext(ActionContext);

	const error = useMemo<string>(() => {
		if (action.matching.out === '' || action.exportMode === 'UpdateOnly') {
			return '';
		}
		// If the export mode is "CreateOnly" or "CreateOrUpdate", the
		// out matching property must be present in the destination
		// schema.
		const flatDestinationSchema = flattenSchema(actionType.outputSchema);
		const p = flatDestinationSchema[action.matching.out]?.full;
		if (p == null) {
			return `${actionType.target} cannot be created but can be updated, as the "${action.matching.out}" property of ${connection.name} is read-only`;
		}
	}, [action]);

	const onChangeExportMode = (e) => {
		const a = { ...action };
		a.exportMode = e.currentTarget.value;
		setAction(a);
	};

	return (
		<div className='action__export-mode'>
			<SlSelect
				className='action__export-mode-select'
				size='medium'
				label='What can be done with users?'
				value={action.exportMode!}
				onSlChange={onChangeExportMode}
			>
				{Object.keys(EXPORT_MODE_OPTIONS).map((k) => (
					<SlOption key={k} value={k}>
						{EXPORT_MODE_OPTIONS[k]}
					</SlOption>
				))}
			</SlSelect>
			{error != '' && <div className='action__export-mode-error'>{error}</div>}
		</div>
	);
};

export default ActionExportMode;
