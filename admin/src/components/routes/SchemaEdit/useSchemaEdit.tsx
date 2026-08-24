import React, { useState, useMemo, useEffect, ReactNode, useRef, useContext } from 'react';
import Type, { ObjectType, Role, TypeKind } from '../../../lib/api/types/types';
import { SortableGridRef, SortableGridRow, GridColumn } from '../../base/Grid/Grid.types';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import {
	EditableProperty,
	EditableSchema,
	getParentPropertyKey,
	normalizeSchema,
	transformSchema,
} from './SchemaEdit.helpers';
import { usePropertyReordering } from './usePropertyReordering';
import { PreviewAlterProfileSchemaResponse, RePaths } from '../../../lib/api/types/responses';
import AppContext from '../../../context/AppContext';
import { isMetaProperty } from '../../../lib/core/schema';
import TransformedConnection from '../../../lib/core/connection';
import { PrimarySources } from '../../../lib/api/types/workspace';
import { SchemaContext } from '../../../context/SchemaContext';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { toKrenalisStringType } from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const SCHEMA_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Type' },
	{ name: 'Description' },
	{ name: 'Primary source' },
	{ name: '' },
];

interface PropertyToEdit {
	key?: string;
	parentKey?: string;
	indentation?: number;
	root?: string;
	name?: string;
	prefilled?: string;
	role?: Role;
	type?: Type | null;
	readOptional?: boolean;
	createRequired?: boolean;
	updateRequired?: boolean;
	nullable?: boolean;
	description?: string;
	isEditable?: boolean;
}

interface PropertyToRemove {
	key: string;
	name: string;
	type: TypeKind;
}

interface PropertyParent {
	key: string;
	label: string;
	indentation: number;
	root: string;
}

interface SelectPropertyOptions {
	animateActionsIfBlocked?: boolean;
}

interface PropertyFieldChanges {
	name: boolean;
	type: boolean;
	description: boolean;
	primarySource: boolean;
}

type SelectProperty = (propertyKey: string, property: EditableProperty, options?: SelectPropertyOptions) => void;

type PropertyChangeStatus = 'added' | 'modified' | 'reordered';

const propertyStatusLabels: Record<PropertyChangeStatus, string> = {
	added: 'Added',
	modified: 'Modified',
	reordered: 'Reordered',
};

const PropertyStatusBadge = ({ status }: { status: PropertyChangeStatus }) => (
	<SlBadge className={`schema-edit__property-status schema-edit__property-status--${status}`} pill variant='neutral'>
		{propertyStatusLabels[status]}
	</SlBadge>
);

const useSchemaEdit = (
	schema: ObjectType,
	onSelectProperty: SelectProperty,
	onClose: () => void,
	selectedPropertyKey?: string,
	search?: string,
	showOnlyChanged?: boolean,
	initialPropertyKey?: string | null,
) => {
	const [editableSchema, setEditableSchema] = useState<EditableSchema>();
	const [queries, setQueries] = useState<string[]>();
	const [isQueriesLoading, setIsQueriesLoading] = useState<boolean>(false);
	const [isConfirmChangesLoading, setIsConfirmChangesLoading] = useState<boolean>(false);

	const sortableGridRef = useRef<SortableGridRef>(null);

	const { api, handleError, workspaces, selectedWorkspace, connections, setIsLoadingWorkspaces } =
		useContext(AppContext);

	const { setIsAltering } = useContext(SchemaContext);

	const primarySources = useRef<PrimarySources>(
		structuredClone(workspaces.find((w) => w.id === selectedWorkspace).primarySources),
	);
	const rePaths = useRef<RePaths>({});
	const deletedAppliedKeys = useRef<string[]>([]);
	const initialEditableSchema = useRef<EditableSchema>();
	const initialPrimarySources = useRef<PrimarySources>();
	const initialSchemaEditStateKey = useRef<string>();
	const { onSortRow, reorderedPropertyKeys, resetMoveHistory } = usePropertyReordering({
		editableSchema,
		initialEditableSchema: initialEditableSchema.current,
		setEditableSchema,
	});
	const schemaEditStateKey = useMemo(() => {
		if (editableSchema == null) {
			return null;
		}
		try {
			return buildSchemaEditStateKey(editableSchema, primarySources.current, rePaths.current);
		} catch {
			return null;
		}
	}, [editableSchema]);

	const hasSchemaChanges =
		initialSchemaEditStateKey.current != null && schemaEditStateKey !== initialSchemaEditStateKey.current;

	const propertyStatuses = useMemo(() => {
		return getPropertyChangeStatuses(
			editableSchema,
			initialEditableSchema.current,
			primarySources.current,
			initialPrimarySources.current,
			reorderedPropertyKeys,
		);
	}, [editableSchema, reorderedPropertyKeys]);
	const selectedPropertyFieldChanges = useMemo(() => {
		if (selectedPropertyKey == null) {
			return undefined;
		}
		const property = editableSchema?.[selectedPropertyKey];
		const initialProperty = initialEditableSchema.current?.[selectedPropertyKey];
		if (property == null || initialProperty == null) {
			return undefined;
		}
		return getPropertyFieldChanges(
			property,
			initialProperty,
			primarySources.current[selectedPropertyKey],
			initialPrimarySources.current?.[selectedPropertyKey],
		);
	}, [editableSchema, selectedPropertyKey]);
	const { objectCount, propertyCount } = useMemo(() => {
		let objectCount = 0;
		const properties = Object.values(editableSchema || {});
		for (const property of properties) {
			if (property.type.kind === 'object') {
				objectCount++;
			}
		}

		return { objectCount, propertyCount: properties.length };
	}, [editableSchema]);
	const propertyParents = useMemo(() => getPropertyParents(editableSchema), [editableSchema]);
	const rows = useMemo(() => {
		return getRows(
			editableSchema,
			primarySources.current,
			connections,
			propertyStatuses,
			selectedPropertyKey,
			search,
			showOnlyChanged,
			onSelectProperty,
		);
	}, [editableSchema, onSelectProperty, propertyStatuses, search, selectedPropertyKey, showOnlyChanged]);

	useEffect(() => {
		if (schema == null) {
			return;
		}
		// Remove meta properties from the schema.
		const s: ObjectType = { kind: 'object', properties: [] };
		for (const p of schema.properties) {
			if (!isMetaProperty(p.name)) {
				s.properties.push(p);
			}
		}
		rePaths.current = {};
		deletedAppliedKeys.current = [];
		resetMoveHistory();
		const editable = transformSchema(s);
		initialEditableSchema.current = structuredClone(editable);
		initialPrimarySources.current = structuredClone(primarySources.current);
		initialSchemaEditStateKey.current = buildSchemaEditStateKey(editable, primarySources.current, rePaths.current);
		setEditableSchema(editable);
		const propertyKey =
			initialPropertyKey != null && editable[initialPropertyKey] != null
				? initialPropertyKey
				: Object.keys(editable)[0];
		if (propertyKey != null) {
			onSelectProperty(propertyKey, editable[propertyKey]);
		}
	}, [resetMoveHistory, schema]);

	const onAddProperty = (property: PropertyToEdit, primarySource: string | null) => {
		if (isMetaProperty(property.name)) {
			throw new Error(`Profile schema property names cannot start with an underscore`);
		}

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
			// If the property now added takes the name of a previously
			// deleted property, add the “null” repath.
			rePaths.current[key] = null;
		}

		// If the property now added takes the name of a previously
		// renamed property, add the "null" repath.
		if (Object.values(rePaths.current).includes(key)) {
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

		// Update the primary sources.
		if (primarySource) {
			primarySources.current[k] = primarySource;
		}

		s[k] = {
			indentation: property.indentation,
			root: property.root === '' ? property.name : property.root,
			name: property.name,
			type: property.type,
			nullable: property.nullable,
			prefilled: '',
			role: 'Both',
			readOptional: true,
			createRequired: false,
			updateRequired: false,
			description: property.description,
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
		return { key: k, ...s[k] };
	};

	const onEditProperty = (property: PropertyToEdit, primarySource: string | null) => {
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
			if (property.type.kind === 'object') {
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
				if (rePaths.current[k] === key) {
					// If it was already renamed previously, delete the
					// old repath.
					delete rePaths.current[k];
				}
			}
			if (key in rePaths.current && rePaths.current[key] == null) {
				// If the property was created with a name identical to
				// that of another previously deleted or renamed
				// property, since we are now renaming it, delete the
				// corresponding “null” repath.
				delete rePaths.current[key];
			}

			let newKey = property.name;
			if (key.includes('.')) {
				newKey = key.split('.').slice(0, -1).join('.') + `.${property.name}`;
			}

			if (deletedAppliedKeys.current.includes(newKey)) {
				// If the property now renamed takes the name of a
				// previously deleted property, add the “null” repath.
				rePaths.current[newKey] = null;
			} else if (!current.isEditable) {
				// If the property was already applied to the schema,
				// add the repath.
				rePaths.current[newKey] = key;
			}
		}

		// Update the primary sources.
		if (primarySource) {
			primarySources.current[key] = primarySource;
		} else {
			if (primarySources.current[key]) {
				delete primarySources.current[key];
			}
		}

		const editedProperty = {
			indentation: current.indentation,
			root: property.name,
			name: property.name,
			type: property.type,
			nullable: property.nullable,
			prefilled: current.prefilled,
			role: current.role,
			readOptional: current.readOptional,
			createRequired: current.createRequired,
			updateRequired: current.updateRequired,
			description: property.description,
			isEditable: current.isEditable ? current.isEditable : false,
		};
		s[key] = editedProperty;
		setEditableSchema(s);
	};

	const onRemoveProperty = (propertyKey: string) => {
		const schema = { ...editableSchema };
		const nextProperty = getPropertySelectionAfterRemoval(schema, propertyKey);
		if (schema[propertyKey].type.kind === 'object') {
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
						if (rePaths.current[k] === key) {
							delete rePaths.current[k];
						}
					}
					// Check if nested property is in primary sources.
					if (primarySources.current[key]) {
						delete primarySources.current[key];
					}
				}
			}
		}
		const isAlreadyApplied = !schema[propertyKey].isEditable;
		if (isAlreadyApplied) {
			deletedAppliedKeys.current.push(propertyKey);
		}
		if (primarySources.current[propertyKey]) {
			delete primarySources.current[propertyKey];
		}
		delete schema[propertyKey];
		setEditableSchema(schema);
		return nextProperty;
	};

	const onApplyChanges = async () => {
		setIsQueriesLoading(true);
		try {
			validateEditableSchema(editableSchema);
		} catch (err) {
			handleError(err);
			setIsQueriesLoading(false);
			return;
		}
		const s = normalizeSchema(editableSchema);
		let res: PreviewAlterProfileSchemaResponse;
		try {
			res = await api.workspaces.previewAlterProfileSchema(s, rePaths.current);
		} catch (err) {
			setTimeout(() => {
				setQueries(null);
				setIsQueriesLoading(false);
				handleError(err);
			}, 300);
			return;
		}
		setTimeout(() => {
			setQueries(res.queries);
			setIsQueriesLoading(false);
		}, 300);
	};

	const onConfirmChanges = async () => {
		// compute the real paths of the primary sources (currently they
		// are based on the editable schema keys).
		const sources: PrimarySources = {};
		for (const k in primarySources.current) {
			let path: string = '';
			let fragments = k.split('.');
			for (let i = 0; i < fragments.length; i++) {
				if (i !== 0) {
					path += '.';
				}
				const key = fragments.slice(0, i + 1).join('.');
				path += editableSchema[key].name;
			}
			sources[path] = primarySources.current[k];
		}
		setIsConfirmChangesLoading(true);
		const s = normalizeSchema(editableSchema);
		try {
			await api.workspaces.alterProfileSchema(s, sources, rePaths.current);
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
			setIsLoadingWorkspaces(true);
			setIsAltering(true);
			onClose();
		}, 300);
	};

	const onCancelChanges = () => {
		setQueries(null);
	};
	const changeCount = hasSchemaChanges
		? Math.max(1, Object.keys(propertyStatuses).length + deletedAppliedKeys.current.length)
		: 0;

	return {
		rows: rows,
		columns: SCHEMA_COLUMNS,
		changeCount,
		objectCount,
		propertyCount,
		propertyParents,
		selectedPropertyFieldChanges,
		propertyStatuses,
		primarySources: primarySources.current,
		queries,
		hasSchemaChanges,
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

const buildSchemaEditStateKey = (
	editableSchema: EditableSchema,
	primarySources: PrimarySources,
	rePaths: RePaths,
): string => {
	const schema = normalizeSchema(structuredClone(editableSchema));
	const sortedPrimarySources = Object.entries(primarySources).sort(([a], [b]) => a.localeCompare(b));
	const sortedRePaths = Object.entries(rePaths).sort(([a], [b]) => a.localeCompare(b));
	return JSON.stringify({ schema, primarySources: sortedPrimarySources, rePaths: sortedRePaths });
};

const getPropertyChangeStatuses = (
	schema: EditableSchema,
	initialSchema: EditableSchema,
	primarySources: PrimarySources,
	initialPrimarySources: PrimarySources,
	reorderedPropertyKeys: ReadonlySet<string>,
): Record<string, PropertyChangeStatus> => {
	const statuses: Record<string, PropertyChangeStatus> = {};
	if (schema == null || initialSchema == null || initialPrimarySources == null) {
		return statuses;
	}
	for (const [key, property] of Object.entries(schema)) {
		const initialProperty = initialSchema[key];
		if (initialProperty == null || property.isEditable === true) {
			statuses[key] = 'added';
			continue;
		}
		const fieldChanges = getPropertyFieldChanges(
			property,
			initialProperty,
			primarySources[key],
			initialPrimarySources[key],
		);
		const hasPropertyChanges = Object.values(fieldChanges).some((changed) => changed);
		if (hasPropertyChanges) {
			statuses[key] = 'modified';
		} else if (reorderedPropertyKeys.has(key)) {
			statuses[key] = 'reordered';
		}
	}
	return statuses;
};

const getPropertyFieldChanges = (
	property: EditableProperty,
	initialProperty: EditableProperty,
	primarySource: string | null | undefined,
	initialPrimarySource: string | null | undefined,
): PropertyFieldChanges => {
	return {
		name: property.name !== initialProperty.name,
		type: JSON.stringify(property.type) !== JSON.stringify(initialProperty.type),
		description: (property.description || '') !== (initialProperty.description || ''),
		primarySource: primarySource !== initialPrimarySource,
	};
};

const getPropertySelectionAfterRemoval = (schema: EditableSchema, propertyKey: string): PropertyToEdit | null => {
	const keys = Object.keys(schema);
	const propertyIndex = keys.indexOf(propertyKey);
	const isRemovedProperty = (key: string) => key === propertyKey || key.startsWith(`${propertyKey}.`);
	let nextPropertyKey = keys.slice(propertyIndex + 1).find((key) => !isRemovedProperty(key));
	if (nextPropertyKey == null) {
		nextPropertyKey = keys
			.slice(0, propertyIndex)
			.reverse()
			.find((key) => !isRemovedProperty(key));
	}
	return nextPropertyKey == null ? null : { key: nextPropertyKey, ...schema[nextPropertyKey] };
};

const getPropertyParents = (schema: EditableSchema): PropertyParent[] => {
	const parents: PropertyParent[] = [
		{
			key: '',
			label: 'Profile (top level)',
			indentation: 0,
			root: '',
		},
	];
	if (schema == null) {
		return parents;
	}
	const parentsByParentKey = new Map<string, PropertyParent[]>();
	for (const [key, property] of Object.entries(schema)) {
		if (property.type.kind !== 'object') {
			continue;
		}
		const path = key
			.split('.')
			.map((_, index, fragments) => schema[fragments.slice(0, index + 1).join('.')]?.name)
			.filter((fragment) => fragment != null)
			.join(' › ');
		const parentKey = getParentPropertyKey(key);
		const siblings = parentsByParentKey.get(parentKey) || [];
		siblings.push({
			key,
			label: path,
			indentation: property.indentation + 1,
			root: property.root,
		});
		parentsByParentKey.set(parentKey, siblings);
	}
	const topLevelParents = parentsByParentKey.get('') || [];
	const pending = [...topLevelParents].reverse();
	while (pending.length > 0) {
		const parent = pending.pop();
		if (parent == null) {
			continue;
		}
		parents.push(parent);
		const children = parentsByParentKey.get(parent.key) || [];
		for (let index = children.length - 1; index >= 0; index--) {
			pending.push(children[index]);
		}
	}
	return parents;
};

const getVisiblePropertyKeys = (
	schema: EditableSchema,
	statuses: Record<string, PropertyChangeStatus>,
	search: string | undefined,
	showOnlyChanged: boolean | undefined,
): Set<string> => {
	const visibleKeys = new Set<string>();
	if (schema == null) {
		return visibleKeys;
	}
	const term = search?.trim().toLocaleLowerCase() || '';
	for (const [key, property] of Object.entries(schema)) {
		const matchesSearch =
			term === '' ||
			`${property.name} ${property.description || ''} ${toKrenalisStringType(property.type)}`
				.toLocaleLowerCase()
				.includes(term);
		const matchesStatus = !showOnlyChanged || statuses[key] != null;
		if (!matchesSearch || !matchesStatus) {
			continue;
		}
		visibleKeys.add(key);
		const fragments = key.split('.');
		for (let index = 1; index < fragments.length; index++) {
			visibleKeys.add(fragments.slice(0, index).join('.'));
		}
		if (term !== '' && property.type.kind === 'object') {
			for (const candidateKey of Object.keys(schema)) {
				if (candidateKey.startsWith(`${key}.`)) {
					visibleKeys.add(candidateKey);
				}
			}
		}
	}
	return visibleKeys;
};
const getRows = (
	schema: EditableSchema,
	primarySources: PrimarySources,
	connections: TransformedConnection[],
	propertyStatuses: Record<string, PropertyChangeStatus>,
	selectedPropertyKey: string | undefined,
	search: string | undefined,
	showOnlyChanged: boolean | undefined,
	onSelectProperty: SelectProperty,
): SortableGridRow[] => {
	const mappedRows = {};
	const visibleKeys = getVisiblePropertyKeys(schema, propertyStatuses, search, showOnlyChanged);
	const isFiltered = (search?.trim() || '') !== '' || showOnlyChanged === true;
	for (const propertyKey in schema) {
		if (!schema.hasOwnProperty(propertyKey) || !visibleKeys.has(propertyKey)) {
			continue;
		}
		let primarySourceConnection: TransformedConnection | null = null;
		if (primarySources[propertyKey]) {
			primarySourceConnection = connections.find((c) => c.id === primarySources[propertyKey]);
		}
		const property = schema[propertyKey];
		const expanded = isFiltered || selectedPropertyKey?.startsWith(`${propertyKey}.`);
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
			if (property.type.kind === 'object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					propertyStatuses[propertyKey],
					selectedPropertyKey === propertyKey,
					expanded,
					isFiltered,
					onSelectProperty,
				);
				m[propertyKey] = subMap;
			} else {
				m[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					propertyStatuses[propertyKey],
					selectedPropertyKey === propertyKey,
					expanded,
					isFiltered,
					onSelectProperty,
				);
			}
		} else {
			if (property.type.kind === 'object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					propertyStatuses[propertyKey],
					selectedPropertyKey === propertyKey,
					expanded,
					isFiltered,
					onSelectProperty,
				);
				mappedRows[propertyKey] = subMap;
			} else {
				mappedRows[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					propertyStatuses[propertyKey],
					selectedPropertyKey === propertyKey,
					expanded,
					isFiltered,
					onSelectProperty,
				);
			}
		}
	}

	return convertToRows(mappedRows);
};

const buildRow = (
	propertyKey: string,
	property: EditableProperty,
	primarySourceConnection: TransformedConnection,
	status: PropertyChangeStatus | undefined,
	selected: boolean,
	expanded: boolean,
	isFiltered: boolean,
	onSelectProperty: SelectProperty,
): SortableGridRow => {
	const actions = (
		<div className='schema-edit__property-actions'>{status != null && <PropertyStatusBadge status={status} />}</div>
	);
	const typeCell: ReactNode = (
		<span className='schema-edit__property-technical-type'>{toKrenalisStringType(property.type)}</span>
	);
	let primarySourceCell: ReactNode;
	if (property.type.kind !== 'object' && property.type.kind !== 'array') {
		if (primarySourceConnection) {
			primarySourceCell = (
				<div className='schema-edit__primary-source'>
					<LittleLogo code={primarySourceConnection.connector.code} path={CONNECTORS_ASSETS_PATH} />
					{primarySourceConnection.name}
				</div>
			);
		} else {
			primarySourceCell = <span className='schema-edit__empty-cell'>—</span>;
		}
	}
	return {
		cells: [
			property.name,
			typeCell,
			property.description || <span className='schema-edit__empty-cell'>—</span>,
			primarySourceCell,
			actions,
		],
		dragKey: isFiltered ? '' : propertyKey,
		expanded,
		id: propertyKey,
		onClick: () => onSelectProperty(propertyKey, property),
		onToggleExpansion: () => onSelectProperty(propertyKey, property, { animateActionsIfBlocked: false }),
		selected,
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

const validateEditableSchema = (editableSchema: EditableSchema) => {
	const keys = Object.keys(editableSchema);
	for (const key of keys) {
		if (!editableSchema.hasOwnProperty(key)) {
			continue;
		}
		const p = editableSchema[key];
		const typ = p.type;
		if (typ.kind === 'object') {
			// Check that it has at least one sub-property.
			const hasSubProperties = keys.some((candidate) => candidate.startsWith(`${key}.`));
			if (!hasSubProperties) {
				throw new Error(`Object property "${p.name}" must contain at least one property`);
			}
		}
	}
};

export {
	useSchemaEdit,
	PropertyStatusBadge,
	PropertyChangeStatus,
	PropertyFieldChanges,
	PropertyParent,
	PropertyToEdit,
	PropertyToRemove,
	SelectPropertyOptions,
};
