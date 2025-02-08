import React from 'react';
import { TransformedMapping, flattenSchema } from '../../lib/core/action';
import { DecimalType, ObjectType, typeKindToIconName } from '../../lib/api/types/types';
import { ComboboxItem } from '../base/Combobox/Combobox.types';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

const getSchemaComboboxItems = (schema: ObjectType | TransformedMapping): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const isFlat = !Array.isArray(schema.properties);
	if (!isFlat) {
		schema = flattenSchema(schema as ObjectType);
	}
	return computeItems(schema as TransformedMapping);
};

const getUIPreferencesComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const typ = flatSchema[k].type;
		if (typ === 'Int' || typ === 'Uint' || typ === 'UUID' || typ === 'Decimal' || typ === 'Text') {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const getFilterPropertyComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const property = flatSchema[k];
		if (property.type === 'Object' || property.type === 'Array') {
			const isNullable = property.full.nullable === true;
			if (!isNullable) {
				continue;
			}
		}
		filteredSchema[k] = v;
	}
	return computeItems(filteredSchema);
};

const getIdentityColumnComboboxItems = (schema: ObjectType): ComboboxItem[] => {
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

const filterOrderingPropertySchema = (schema: ObjectType): TransformedMapping | null => {
	if (schema == null) {
		return null;
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
	return filteredSchema;
};

const getOrderingPropertyPathComboboxItems = (schema: ObjectType): ComboboxItem[] => {
	const filteredSchema = filterOrderingPropertySchema(schema);
	if (filteredSchema == null) {
		return [];
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
	for (const name in flatSchema) {
		let typ = flatSchema[name].type;
		items.push({
			content: (
				<div className='schema-combobox-item' key={name}>
					<div className='schema-combobox-item__type'>
						<SlIcon name={typeKindToIconName[typ]} />
					</div>
					<div className='schema-combobox-item__text'>
						<div className='schema-combobox-item__name'>{name}</div>
					</div>
				</div>
			),
			term: name,
		});
	}
	return items;
};

export {
	getSchemaComboboxItems,
	getUIPreferencesComboboxItems,
	getIdentityColumnComboboxItems,
	getLastChangeTimeComboboxItems,
	filterOrderingPropertySchema,
	getOrderingPropertyPathComboboxItems,
	getTableKeyComboboxItems,
	getFilterPropertyComboboxItems,
};
