import React, { Fragment } from 'react';
import { TransformedMapping, flattenSchema } from '../../lib/core/action';
import { DecimalType, ObjectType } from '../../lib/api/types/types';
import { ComboboxItem } from '../base/ComboBox/ComboBox.types';

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

const getLastChangeTimeComboboxItems = (schema: ObjectType): ComboboxItem[] => {
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

const getOrderingPropertyPathComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (typ === 'Decimal') {
			const t = flatSchema[k].full.type as DecimalType;
			if (t.scale === 0) {
				filteredSchema[k] = v;
			}
			continue;
		}
		if (typ === 'Int' || typ === 'Uint' || typ === 'UUID' || typ === 'Inet' || typ === 'Text') {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const getTableKeyComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (typ === 'Int' || typ === 'Uint' || typ === 'UUID' || typ === 'Text') {
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
					<div className='schema-combobox-item'>
						{flatSchema[propertyName].label != null && flatSchema[propertyName].label !== '' ? (
							<>
								<div className='schema-combobox-item__label'>{flatSchema[propertyName].label}</div>
								<div className='schema-combobox-item__name'>{propertyName}</div>
							</>
						) : (
							propertyName
						)}
					</div>
					<div className='schema-combobox-item__type'>{typ}</div>
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
	getLastChangeTimeComboboxItems,
	getOrderingPropertyPathComboboxItems,
	getTableKeyComboboxItems,
};
