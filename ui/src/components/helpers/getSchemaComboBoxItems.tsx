import React, { Fragment } from 'react';
import { TransformedMapping, flattenSchema } from '../../lib/helpers/transformedAction';
import { ObjectType } from '../../types/external/types';
import { ComboboxItem } from '../../types/internal/app';

const getSchemaComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	return computeItems(flatSchema);
};

const getIdentityPropertyComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (typ === 'Int' || typ === 'Uint' || typ === 'UUID' || typ === 'JSON' || typ === 'Text') {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const getDisplayedIDComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (
			typ === 'Int' ||
			typ === 'Uint' ||
			typ === 'Float' ||
			typ === 'Decimal' ||
			typ === 'JSON' ||
			typ === 'Text'
		) {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const getUpdatedAtComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (typ === 'DateTime' || typ === 'Date' || typ == 'JSON' || typ === 'Text') {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const computeItems = (flatSchema: TransformedMapping) => {
	const items: ComboboxItem[] = [];
	for (const propertyName in flatSchema) {
		let typ = flatSchema[propertyName].type;
		if (typ === 'Int' || typ === 'Uint' || typ === 'Float') {
			typ += `(${flatSchema[propertyName].size})`;
		}
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
					<div className='propertiesItemType'>{typ}</div>
				</Fragment>
			),
			term: propertyName,
		});
	}
	return items;
};

export {
	getSchemaComboboxItems,
	getIdentityPropertyComboboxItems,
	getDisplayedIDComboboxItems,
	getUpdatedAtComboboxItems,
};
