import React, { Fragment } from 'react';
import { flattenSchema } from '../../lib/helpers/transformedAction';
import { ObjectType } from '../../types/external/types';
import { ComboboxItem } from '../../types/internal/app';

const getSchemaComboboxItems = (schema: ObjectType, nonSelectableProperties?: string[]): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const items: ComboboxItem[] = [];
	for (const propertyName in flatSchema) {
		if (nonSelectableProperties && nonSelectableProperties.includes(propertyName)) continue;
		items.push({
			content: (
				<Fragment key={propertyName}>
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
				</Fragment>
			),
			term: propertyName,
		});
	}
	return items;
};

export { getSchemaComboboxItems };
