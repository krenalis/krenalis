import { flattenSchema } from '../../lib/helpers/action';

const getSchemaComboboxItems = (schema, disabledKeys) => {
	if (schema == null) {
		return [];
	}
	const properties = flattenSchema(schema);
	const propertiesList = [];
	for (const k in properties) {
		if (disabledKeys && disabledKeys.includes(k)) continue;
		let name;
		if (properties[k].label != null && properties[k].label !== '') {
			name = (
				<div className='propertiesItemName'>
					<div className='label'>{properties[k].label}</div>
					<div className='name'>{k}</div>
				</div>
			);
		} else {
			name = <div className='propertiesItemName'>{k}</div>;
		}
		const content = (
			<>
				{name}
				<div className='propertiesItemType'>{properties[k].type}</div>
			</>
		);
		propertiesList.push({
			content: content,
			searchableTerm: k,
		});
	}
	return propertiesList;
};

export { getSchemaComboboxItems };
