import React from 'react';
import { TransformedMapping, flattenSchema, getCompatibleFilterOperators } from '../../lib/core/action';
import { DecimalType, ObjectType, Role } from '../../lib/api/types/types';
import { ComboboxItem } from '../base/Combobox/Combobox.types';
import { TypeIcon } from '../base/TypeIcon/TypeIcon';
import { ActionTarget } from '../../lib/api/types/action';

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
		if (typ === 'int' || typ === 'uint' || typ === 'uuid' || typ === 'decimal' || typ === 'text') {
			filteredSchema[k] = v;
		}
	}
	return computeItems(filteredSchema);
};

const getFilterPropertyComboboxItems = (schema: ObjectType, role: Role, target: ActionTarget): ComboboxItem[] => {
	if (schema == null) {
		return [];
	}
	const flatSchema = flattenSchema(schema);
	const filteredSchema: TransformedMapping = {};
	for (const [k, v] of Object.entries(flatSchema)) {
		const property = flatSchema[k];
		if (property.type === 'object' || property.type === 'array') {
			const compatibleOperators = getCompatibleFilterOperators(property, false, role, target);
			if (compatibleOperators.length === 0) {
				continue;
			}
		} else if (property.type === 'json' && role === 'Destination' && target === 'User') {
			continue;
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
		if (v.readOptional) {
			continue;
		}
		const typ = flatSchema[k].type;
		if (typ === 'int' || typ === 'uint' || typ === 'uuid' || typ === 'json' || typ === 'text') {
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
		if (typ === 'datetime' || typ === 'date' || typ == 'json' || typ === 'text') {
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
		if (typ === 'decimal') {
			const t = flatSchema[k].full.type as DecimalType;
			if (t.scale === 0) {
				filteredSchema[k] = v;
			}
			continue;
		}
		if (typ === 'int' || typ === 'uint' || typ === 'uuid' || typ === 'inet' || typ === 'text') {
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
		if (typ === 'int' || typ === 'uint' || typ === 'uuid' || typ === 'text') {
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
						<TypeIcon kind={typ} />
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
