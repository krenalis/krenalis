import { flattenSchema } from '../../lib/helpers/action';

const getSchemaComboboxItems = (schema, nonSelectableProperties) => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const items = [];
	for (const propertyName in flatSchema) {
		if (nonSelectableProperties && nonSelectableProperties.includes(propertyName)) continue;
		items.push({
			content: (
				<>
					<div className='propertiesItemName'>
						{flatSchema[propertyName].label != null && flatSchema[propertyName].label !== '' ? (
							<>
								<div className='label'>{flatSchema[propertyName].label}</div>
								<div className='name'>{propertyName}</div>
							</>
						) : (
							propertyName
						)}
					</div>
					<div className='propertiesItemType'>{flatSchema[propertyName].type}</div>
				</>
			),
			searchableTerm: propertyName,
		});
	}
	return items;
};

export { getSchemaComboboxItems };
