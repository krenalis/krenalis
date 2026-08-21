import React, { useState, useMemo, useEffect, ReactNode, useRef, useContext, useCallback } from 'react';
import Type, { ObjectType, Role, TypeKind } from '../../../lib/api/types/types';
import { SortableGridRow, GridColumn } from '../../base/Grid/Grid.types';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { EditableProperty, EditableSchema, transformSchema, normalizeSchema } from './SchemaEdit.helpers';
import { PreviewAlterProfileSchemaResponse, RePaths } from '../../../lib/api/types/responses';
import AppContext from '../../../context/AppContext';
import { SortableGridRef } from '../../base/Grid/SortableGrid';
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
	{ name: 'Primary source' },
	{ name: '' }, // buttons
];

interface PropertyToEdit {
	key?: string;
	parentKey?: string;
	indentation?: number;
	root?: string;
	name?: string;
	label?: string;
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

interface SchemaPreviewRequest {
	key: string;
	schema: ObjectType;
	rePaths: RePaths;
}

interface SchemaPreview {
	key: string;
	queries: string[];
}

const useSchemaEdit = (
	schema: ObjectType,
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeKind: TypeKind) => void,
	onClose: () => void,
) => {
	const [editableSchema, setEditableSchema] = useState<EditableSchema>();
	const [queries, setQueries] = useState<string[]>();
	const [schemaPreview, setSchemaPreview] = useState<SchemaPreview>();
	const [isSchemaPreviewLoading, setIsSchemaPreviewLoading] = useState<boolean>(false);
	const [isQueriesLoading, setIsQueriesLoading] = useState<boolean>(false);
	const [isConfirmChangesLoading, setIsConfirmChangesLoading] = useState<boolean>(false);

	const sortableGridRef = useRef<SortableGridRef>();

	const { api, handleError, workspaces, selectedWorkspace, connections, setIsLoadingWorkspaces } =
		useContext(AppContext);

	const { setIsAltering } = useContext(SchemaContext);

	const primarySources = useRef<PrimarySources>(workspaces.find((w) => w.id === selectedWorkspace).primarySources);
	const rePaths = useRef<RePaths>({});
	const deletedAppliedKeys = useRef<string[]>([]);
	const initialSchemaEditStateKey = useRef<string>();
	const schemaPreviewInFlight = useRef<{
		key: string;
		promise: Promise<PreviewAlterProfileSchemaResponse>;
	}>();

	const requestSchemaPreview = useCallback(
		(request: SchemaPreviewRequest): Promise<PreviewAlterProfileSchemaResponse> => {
			if (schemaPreviewInFlight.current?.key === request.key) {
				return schemaPreviewInFlight.current.promise;
			}
			const promise = api.workspaces.previewAlterProfileSchema(request.schema, request.rePaths);
			schemaPreviewInFlight.current = { key: request.key, promise };
			const clearRequest = () => {
				if (schemaPreviewInFlight.current?.promise === promise) {
					schemaPreviewInFlight.current = undefined;
				}
			};
			void promise.then(clearRequest, clearRequest);
			return promise;
		},
		[api],
	);

	const schemaPreviewRequest = useMemo(() => {
		if (editableSchema == null) {
			return null;
		}
		try {
			return buildSchemaPreviewRequest(editableSchema, rePaths.current);
		} catch {
			return null;
		}
	}, [editableSchema]);

	const currentSchemaPreview = schemaPreview?.key === schemaPreviewRequest?.key ? schemaPreview : undefined;
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
	const warehouseDDLRequired = currentSchemaPreview != null && currentSchemaPreview.queries.length > 0;

	const rows = useMemo(() => {
		return getRows(editableSchema, primarySources.current, connections, onAddClick, onEditClick, onRemoveClick);
	}, [editableSchema]);

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
		const editable = transformSchema(s);
		const request = buildSchemaPreviewRequest(editable, rePaths.current);
		initialSchemaEditStateKey.current = buildSchemaEditStateKey(editable, primarySources.current, rePaths.current);
		setIsSchemaPreviewLoading(false);
		setSchemaPreview({ key: request.key, queries: [] });
		setEditableSchema(editable);
	}, [schema]);

	useEffect(() => {
		if (!hasSchemaChanges) {
			setIsSchemaPreviewLoading(false);
			if (schemaPreviewRequest != null && schemaPreview?.key !== schemaPreviewRequest.key) {
				setSchemaPreview({ key: schemaPreviewRequest.key, queries: [] });
			}
			return;
		}
		const shouldRequestPreview = schemaPreviewRequest != null && schemaPreview?.key !== schemaPreviewRequest.key;
		setIsSchemaPreviewLoading(shouldRequestPreview);
		if (!shouldRequestPreview) {
			return;
		}
		let active = true;
		const timeout = window.setTimeout(async () => {
			try {
				const res = await requestSchemaPreview(schemaPreviewRequest);
				if (active) {
					setSchemaPreview({ key: schemaPreviewRequest.key, queries: res.queries });
					setIsSchemaPreviewLoading(false);
				}
			} catch {
				// Automatic previews are best-effort. Errors are reported if
				// the user explicitly tries to apply the changes.
				if (active) {
					setIsSchemaPreviewLoading(false);
				}
			}
		}, 300);
		return () => {
			active = false;
			window.clearTimeout(timeout);
		};
	}, [hasSchemaChanges, requestSchemaPreview, schemaPreview, schemaPreviewRequest]);

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
			} else if (!current.isEditable && newKey !== key) {
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
		// Applied property types are read-only, so these fields are enough to
		// identify edits that cannot change the warehouse schema.
		const warehouseSchemaUnchanged =
			!current.isEditable &&
			property.name === current.name &&
			property.type === current.type &&
			property.nullable === current.nullable;
		if (warehouseSchemaUnchanged && currentSchemaPreview != null) {
			const request = buildSchemaPreviewRequest(s, rePaths.current);
			setSchemaPreview({ key: request.key, queries: currentSchemaPreview.queries });
		}
		setEditableSchema(s);
	};

	const onRemoveProperty = (propertyKey: string) => {
		const schema = { ...editableSchema };
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
		let request: SchemaPreviewRequest;
		try {
			request = buildSchemaPreviewRequest(editableSchema, rePaths.current);
		} catch (err) {
			handleError(err);
			return;
		}
		if (schemaPreview?.key === request.key) {
			setQueries(schemaPreview.queries);
			return;
		}
		setIsQueriesLoading(true);
		let res: PreviewAlterProfileSchemaResponse;
		try {
			res = await requestSchemaPreview(request);
		} catch (err) {
			setTimeout(() => {
				setQueries(null);
				setIsQueriesLoading(false);
				handleError(err);
			}, 300);
			return;
		}
		setTimeout(() => {
			setSchemaPreview({ key: request.key, queries: res.queries });
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

	return {
		rows: rows,
		columns: SCHEMA_COLUMNS,
		primarySources: primarySources.current,
		queries,
		hasSchemaChanges,
		isSchemaPreviewLoading,
		isQueriesLoading,
		isConfirmChangesLoading,
		warehouseDDLRequired,
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
	const schema = normalizeSchema(editableSchema);
	const sortedPrimarySources = Object.entries(primarySources).sort(([a], [b]) => a.localeCompare(b));
	const sortedRePaths = Object.entries(rePaths).sort(([a], [b]) => a.localeCompare(b));
	return JSON.stringify({ schema, primarySources: sortedPrimarySources, rePaths: sortedRePaths });
};

const buildSchemaPreviewRequest = (editableSchema: EditableSchema, rePaths: RePaths): SchemaPreviewRequest => {
	validateEditableSchema(editableSchema);
	const schema = normalizeSchema(editableSchema);
	const currentRePaths = { ...rePaths };
	return {
		key: JSON.stringify({ schema, rePaths: currentRePaths }),
		schema,
		rePaths: currentRePaths,
	};
};

const getRows = (
	schema: EditableSchema,
	primarySources: PrimarySources,
	connections: TransformedConnection[],
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeKind: TypeKind) => void,
): SortableGridRow[] => {
	const mappedRows = {};
	for (const propertyKey in schema) {
		if (!schema.hasOwnProperty(propertyKey)) {
			continue;
		}
		let primarySourceConnection: TransformedConnection | null = null;
		if (primarySources[propertyKey]) {
			primarySourceConnection = connections.find((c) => c.id === primarySources[propertyKey]);
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
			if (property.type.kind === 'object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					onAddClick,
					onEditClick,
					onRemoveClick,
				);
				m[propertyKey] = subMap;
			} else {
				m[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					null,
					onEditClick,
					onRemoveClick,
				);
			}
		} else {
			if (property.type.kind === 'object') {
				const subMap = {};
				subMap[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					onAddClick,
					onEditClick,
					onRemoveClick,
				);
				mappedRows[propertyKey] = subMap;
			} else {
				mappedRows[propertyKey] = buildRow(
					propertyKey,
					property,
					primarySourceConnection,
					null,
					onEditClick,
					onRemoveClick,
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
	onAddClick: (parentKey: string, indentation: number, root: string) => void,
	onEditClick: (propertyKey: string, property: EditableProperty) => void,
	onRemoveClick: (propertyKey: string, propertyName: string, typeKind: TypeKind) => void,
): SortableGridRow => {
	const buttons = (
		<div className='schema-edit__property-buttons'>
			<SlButton
				className='schema-edit__property-buttons-edit'
				size='small'
				onClick={() => onEditClick(propertyKey, property)}
			>
				Edit
			</SlButton>
			<SlButton
				size='small'
				className='schema-edit__property-buttons-remove'
				variant='danger'
				outline={true}
				onClick={() => onRemoveClick(propertyKey, property.name, property.type.kind)}
			>
				Remove
			</SlButton>
		</div>
	);
	let typeCell: ReactNode;
	if (property.type.kind === 'object') {
		typeCell = (
			<div className='schema-edit__editable-object-cell'>
				{property.type.kind}
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
		typeCell = toKrenalisStringType(property.type);
	}
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
			primarySourceCell = 'None';
		}
	}
	return {
		cells: [property.name, typeCell, primarySourceCell, buttons],
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
			const subProperties = keys.filter((k) => k.startsWith(key) && k !== key);
			if (subProperties.length === 0) {
				throw new Error(`object property "${p.name}" must have at least one sub property`);
			}
		}
	}
};

export { useSchemaEdit, PropertyToEdit, PropertyToRemove };
