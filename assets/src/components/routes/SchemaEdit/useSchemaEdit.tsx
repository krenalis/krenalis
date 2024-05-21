import React, { useState, useMemo, useEffect, ReactNode, useRef, useContext } from 'react';
import Type, { ObjectType, Role, TypeName } from '../../../types/external/types';
import { SortableGridRow, GridColumn } from '../../shared/Grid/Grid.types';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { EditableProperty, EditableSchema, transformSchema, normalizeSchema } from './SchemaEdit.helpers';
import { ChangeUsersSchemaQueriesResponse, RePaths } from '../../../types/external/api';
import AppContext from '../../../context/AppContext';
import { enrichPropertyType } from '../../helpers/enrichPropertyType';
import { SortableGridRef } from '../../shared/Grid/SortableGrid';

const SCHEMA_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Type' },
	{ name: 'Nullable', alignment: 'center' },
	{ name: 'Label' },
	{ name: '' }, // buttons
];

interface PropertyToEdit {
	key?: string;
	parentKey?: string;
	indentation?: number;
	root?: string;
	name?: string;
	label?: string;
	description?: string;
	placeholder?: string;
	role?: Role;
	type?: Type | null;
	required?: boolean;
	nullable?: boolean;
	isEditable?: boolean;
}

interface PropertyToRemove {
	key: string;
	name: string;
	type: TypeName;
}

const useSchemaEdit = (
	schema: ObjectType,
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeName: TypeName) => void,
	onClose: () => void,
) => {
	const [editableSchema, setEditableSchema] = useState<EditableSchema>();
	const [queries, setQueries] = useState<string[]>();
	const [isQueriesLoading, setIsQueriesLoading] = useState<boolean>(false);
	const [isConfirmChangesLoading, setIsConfirmChangesLoading] = useState<boolean>(false);

	const sortableGridRef = useRef<SortableGridRef>();

	const { api, handleError } = useContext(AppContext);

	const rows = useMemo(() => {
		return getRows(editableSchema, onAddClick, onEditClick, onRemoveClick);
	}, [editableSchema]);

	useEffect(() => {
		setEditableSchema(transformSchema(schema));
	}, [schema]);

	const rePaths = useRef<RePaths>({});
	const deletedAppliedKeys = useRef<string[]>([]);

	const onAddProperty = (property: PropertyToEdit) => {
		let key = property.name;
		if (property.parentKey !== '') {
			key = `${property.parentKey}.${property.name}`;
		}

		// Check if a property with the same name already exists.
		for (let k in editableSchema) {
			if (!editableSchema.hasOwnProperty(k)) {
				continue;
			}
			let p = editableSchema[k];
			if (p.name === property.name) {
				if (p.indentation === property.indentation) {
					if (p.indentation > 0) {
						if (p.root === property.root) {
							throw new Error(`Property "${property.name}" already exists`);
						}
					} else {
						throw new Error(`Property "${property.name}" already exists`);
					}
				}
			}
		}

		// Update the RePaths.
		if (deletedAppliedKeys.current.includes(key)) {
			rePaths.current[key] = null;
		}

		const s = { ...editableSchema };

		// Check if the key already exists (a renamed property, changes
		// the name but maintains the same key), and in that case add a
		// numeric index to it.
		let k = key;
		let counter = 2;
		while (s[k] != null) {
			k = `${key}-${counter}`;
		}

		s[k] = {
			indentation: property.indentation,
			root: property.root === '' ? property.name : property.root,
			name: property.name,
			type: property.type,
			nullable: property.nullable,
			label: property.label,
			description: property.description,
			placeholder: '',
			role: 'Both',
			required: false,
			isEditable: true,
		};
		setEditableSchema(s);
		if (property.indentation > 0) {
			setTimeout(() => {
				if (sortableGridRef.current != null) {
					sortableGridRef.current.expandRow(property.parentKey);
				}
			}, 100);
		}
	};

	const onEditProperty = (property: PropertyToEdit) => {
		const key = property.key;
		const s = { ...editableSchema };
		const current = s[key];

		// Check if the property has been renamed.
		if (property.name !== current.name) {
			// Check if a property with the same name already exists.
			for (let k in s) {
				if (!s.hasOwnProperty(k)) {
					continue;
				}
				let p = s[k];
				if (p.name === property.name) {
					if (p.indentation === property.indentation) {
						if (p.indentation > 0) {
							if (p.root === property.root) {
								throw new Error(`Property "${property.name}" already exists`);
							}
						} else {
							throw new Error(`Property "${property.name}" already exists`);
						}
					}
				}
			}

			// Update the 'root' field of the children properties.
			if (property.type.name === 'Object') {
				for (let k in s) {
					if (!s.hasOwnProperty(k)) {
						continue;
					}
					let p = { ...s[k] };
					if (p.root === current.name) {
						p.root = property.name;
						s[k] = p;
					}
				}
			}

			// Update the RePaths.
			for (const k in rePaths.current) {
				if (!rePaths.current.hasOwnProperty(k)) {
					continue;
				}
				if (rePaths.current[k] === key) {
					delete rePaths.current[k];
				}
			}
			if (key in rePaths.current && rePaths.current[key] == null) {
				delete rePaths.current[key];
			}

			let newKey = property.name;
			if (key.includes('.')) {
				newKey = key.split('.').slice(0, -1).join('.') + `.${property.name}`;
			}

			if (deletedAppliedKeys.current.includes(newKey)) {
				rePaths.current[newKey] = null;
			}
			if (!current.isEditable) {
				rePaths.current[newKey] = key;
			}
		}

		const editedProperty = {
			indentation: current.indentation,
			root: property.name,
			name: property.name,
			type: property.type,
			nullable: property.nullable,
			label: property.label,
			description: property.description,
			placeholder: current.placeholder,
			role: current.role,
			required: current.required,
			isEditable: current.isEditable ? current.isEditable : false,
		};
		s[key] = editedProperty;
		setEditableSchema(s);
	};

	const onRemoveProperty = (propertyKey: string) => {
		const schema = { ...editableSchema };
		if (schema[propertyKey].type.name === 'Object') {
			for (const key of Object.keys(schema)) {
				const isNested = key.startsWith(`${propertyKey}.`);
				if (isNested) {
					delete schema[key];
					// Check if nested property is in the deleted keys.
					if (deletedAppliedKeys.current.includes(key)) {
						delete deletedAppliedKeys.current[key];
					}
					// Check if nested property is in the RePaths.
					for (const k in rePaths.current) {
						if (!rePaths.current.hasOwnProperty(k)) {
							continue;
						}
						if (rePaths.current[k] === key) {
							delete rePaths.current[k];
						}
					}
				}
			}
		}
		const isAlreadyApplied = !schema[propertyKey].isEditable;
		if (isAlreadyApplied) {
			deletedAppliedKeys.current.push(propertyKey);
		}
		delete schema[propertyKey];
		setEditableSchema(schema);
	};

	const onSortRow = (overRowID: string, movedRowID: string) => {
		const s = { ...editableSchema };
		const keys = Object.keys(s);
		const overPropertyIndex = keys.findIndex((k) => k === overRowID);
		const movedPropertyIndex = keys.findIndex((k) => k === movedRowID);
		const isAfter = overPropertyIndex > movedPropertyIndex;
		const keysToMove = keys.filter((k) => k === movedRowID || k.startsWith(`${movedRowID}.`));
		const sk = keys.filter((k) => !keysToMove.includes(k));
		let insertIndex = sk.findIndex((k) => k === overRowID);
		if (isAfter) {
			insertIndex++;
		}
		sk.splice(insertIndex, 0, ...keysToMove);
		const newSchema: EditableSchema = {};
		for (let key of sk) {
			newSchema[key] = s[key];
		}
		setEditableSchema(newSchema);
	};

	const onApplyChanges = async () => {
		setIsQueriesLoading(true);
		const s = normalizeSchema(editableSchema);
		let res: ChangeUsersSchemaQueriesResponse;
		try {
			res = await api.workspaces.changeUsersSchemaQueries(s, rePaths.current);
		} catch (err) {
			setTimeout(() => {
				setQueries(null);
				setIsQueriesLoading(false);
				handleError(err);
			}, 300);
			return;
		}
		setTimeout(() => {
			setQueries(res.Queries);
			setIsQueriesLoading(false);
		}, 300);
	};

	const onConfirmChanges = async () => {
		setIsConfirmChangesLoading(true);
		const s = normalizeSchema(editableSchema);
		try {
			await api.workspaces.changeUsersSchema(s, rePaths.current);
		} catch (err) {
			setTimeout(() => {
				setQueries(null);
				setIsConfirmChangesLoading(false);
				handleError(err);
			}, 300);
			return;
		}
		setTimeout(() => {
			setQueries(null);
			setIsConfirmChangesLoading(false);
			onClose();
		}, 300);
	};

	const onCancelChanges = () => {
		setQueries(null);
	};

	return {
		rows: rows,
		columns: SCHEMA_COLUMNS,
		queries,
		isQueriesLoading,
		isConfirmChangesLoading,
		onAddProperty,
		onEditProperty,
		onRemoveProperty,
		onSortRow,
		onApplyChanges,
		onConfirmChanges,
		onCancelChanges,
		sortableGridRef,
	};
};

const getRows = (
	schema: EditableSchema,
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeName: TypeName) => void,
): SortableGridRow[] => {
	const mappedRows = {};
	for (const propertyKey in schema) {
		if (!schema.hasOwnProperty(propertyKey)) {
			continue;
		}
		const property = schema[propertyKey];
		const isSubProperty = property.indentation > 0;
		if (isSubProperty) {
			let fragments = propertyKey.split('.');
			let prefixes: string[] = [];
			for (let i = 1; i < fragments.length; i++) {
				prefixes.push(fragments.slice(0, i).join('.'));
			}
			let m = mappedRows;
			for (const prefix of prefixes) {
				m = m[prefix];
			}
			if (property.type.name === 'Object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(propertyKey, property, onAddClick, onEditClick, onRemoveClick);
				m[propertyKey] = subMap;
			} else {
				m[propertyKey] = buildRow(propertyKey, property, null, onEditClick, onRemoveClick);
			}
		} else {
			if (property.type.name === 'Object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(propertyKey, property, onAddClick, onEditClick, onRemoveClick);
				mappedRows[propertyKey] = subMap;
			} else {
				mappedRows[propertyKey] = buildRow(propertyKey, property, null, onEditClick, onRemoveClick);
			}
		}
	}

	return convertToRows(mappedRows);
};

const buildRow = (
	propertyKey: string,
	property: EditableProperty,
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeName: TypeName) => void,
): SortableGridRow => {
	const buttons = (
		<div className='schema-edit__property-buttons'>
			<SlButton size='small' onClick={() => onEditClick(propertyKey, property)}>
				Edit
			</SlButton>
			<SlButton
				size='small'
				variant='danger'
				outline={true}
				onClick={() => onRemoveClick(propertyKey, property.name, property.type.name)}
			>
				Remove
			</SlButton>
		</div>
	);
	let typeCell: ReactNode;
	if (property.type.name === 'Object') {
		typeCell = (
			<div className='schema-edit__editable-object-cell'>
				{property.type.name}
				<SlButton
					size='small'
					variant='primary'
					onClick={() => onAddClick(propertyKey, property.indentation + 1, property.name)}
					outline={true}
				>
					<SlIcon name='plus-circle' slot='suffix' />
					Add property
				</SlButton>
			</div>
		);
	} else {
		typeCell = enrichPropertyType(property.type);
	}
	let nullableCell: ReactNode;
	if (property.nullable) {
		nullableCell = 'Yes';
	} else {
		nullableCell = 'No';
	}
	return {
		cells: [property.name, typeCell, nullableCell, property.label, buttons],
		dragKey: propertyKey,
		id: propertyKey,
	};
};

const convertToRows = (mappedRows: object): SortableGridRow[] => {
	const rows: SortableGridRow[] = [];
	for (const key in mappedRows) {
		if (!mappedRows.hasOwnProperty(key)) {
			continue;
		}
		const isObjectRow = mappedRows[key].cells == null;
		if (isObjectRow) {
			rows.push(convertToRows(mappedRows[key]) as unknown as SortableGridRow);
		} else {
			rows.push(mappedRows[key]);
		}
	}
	return rows;
};

export { useSchemaEdit, PropertyToEdit, PropertyToRemove };
