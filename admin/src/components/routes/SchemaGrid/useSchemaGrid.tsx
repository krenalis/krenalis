import React, { ReactNode, useContext, useMemo } from 'react';
import { ObjectType, Property } from '../../../lib/api/types/types';
import { GridColumn, GridRow, StandardGridRow } from '../../base/Grid/Grid.types';
import AppContext from '../../../context/AppContext';
import TransformedConnection from '../../../lib/core/connection';
import { PrimarySources } from '../../../lib/api/types/workspace';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { toKrenalisStringType } from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const SCHEMA_COLUMNS: GridColumn[] = [
	{ name: 'Property' },
	{ name: 'Type' },
	{ name: 'Description' },
	{ name: 'Primary source' },
];

const useSchemaGrid = (
	schema: ObjectType,
	isLoading: boolean,
	search: string,
	selectedPropertyPath: string | null,
	onSelectProperty: (path: string) => void,
) => {
	const { workspaces, selectedWorkspace, connections } = useContext(AppContext);
	const workspace = workspaces.find((candidate) => candidate.id === selectedWorkspace);

	const rows = useMemo(() => {
		if (isLoading || schema == null) {
			return [];
		}
		return getRows(
			schema,
			workspace.primarySources,
			connections,
			search.trim().toLocaleLowerCase(),
			selectedPropertyPath,
			onSelectProperty,
		);
	}, [connections, isLoading, onSelectProperty, schema, search, selectedPropertyPath, workspace]);
	const { objectCount, propertyCount } = useMemo(() => countProperties(schema), [schema]);
	const selectedProperty = useMemo(() => {
		if (schema == null || selectedPropertyPath == null) {
			return null;
		}
		const property = getPropertyByPath(schema, selectedPropertyPath);
		if (property == null) {
			return null;
		}
		return {
			path: selectedPropertyPath,
			primarySource: getPrimarySource(selectedPropertyPath, workspace.primarySources, connections),
			property,
		};
	}, [connections, schema, selectedPropertyPath, workspace]);

	return {
		columns: SCHEMA_COLUMNS,
		objectCount,
		propertyCount,
		rows,
		selectedProperty,
	};
};

const getRows = (
	schema: ObjectType,
	primarySources: PrimarySources,
	connections: TransformedConnection[],
	search: string,
	selectedPropertyPath: string | null,
	onSelectProperty: (path: string) => void,
	parent?: string,
	includeAll?: boolean,
): GridRow[] => {
	const rows: GridRow[] = [];
	for (const property of schema.properties || []) {
		const path = parent == null ? property.name : `${parent}.${property.name}`;
		const matches =
			includeAll ||
			search === '' ||
			`${property.name} ${property.description || ''} ${toKrenalisStringType(property.type)}`
				.toLocaleLowerCase()
				.includes(search);
		let nestedRows: GridRow[] = [];
		if (property.type.kind === 'object') {
			nestedRows = getRows(
				property.type,
				primarySources,
				connections,
				search,
				selectedPropertyPath,
				onSelectProperty,
				path,
				includeAll || matches,
			);
		}
		if (!matches && nestedRows.length === 0) {
			continue;
		}
		const primarySource = getPrimarySource(path, primarySources, connections);
		const row = buildRow(
			path,
			property,
			primarySource,
			selectedPropertyPath === path,
			search !== '',
			onSelectProperty,
		);
		if (property.type.kind === 'object') {
			rows.push([row, ...nestedRows]);
		} else {
			rows.push(row);
		}
	}
	return rows;
};

const buildRow = (
	path: string,
	property: Property,
	primarySource: TransformedConnection | null,
	selected: boolean,
	expanded: boolean,
	onSelectProperty: (path: string) => void,
): StandardGridRow => {
	const typeCell: ReactNode = (
		<span className='schema-grid__technical-type'>{toKrenalisStringType(property.type)}</span>
	);
	let primarySourceCell: ReactNode = <span className='schema-grid__empty-cell'>—</span>;
	if (property.type.kind !== 'object' && property.type.kind !== 'array') {
		if (primarySource != null) {
			primarySourceCell = (
				<div className='schema-grid__primary-source'>
					<LittleLogo code={primarySource.connector.code} path={CONNECTORS_ASSETS_PATH} />
					{primarySource.name}
				</div>
			);
		}
	}
	return {
		cells: [
			property.name,
			typeCell,
			property.description || <span className='schema-grid__empty-cell'>—</span>,
			primarySourceCell,
		],
		expanded,
		id: path,
		onClick: () => onSelectProperty(path),
		selected,
	};
};

const countProperties = (schema: ObjectType): { objectCount: number; propertyCount: number } => {
	let objectCount = 0;
	let propertyCount = 0;
	for (const property of schema?.properties || []) {
		propertyCount++;
		if (property.type.kind === 'object') {
			objectCount++;
			const nestedCounts = countProperties(property.type);
			objectCount += nestedCounts.objectCount;
			propertyCount += nestedCounts.propertyCount;
		}
	}
	return { objectCount, propertyCount };
};

const getPrimarySource = (
	path: string,
	primarySources: PrimarySources,
	connections: TransformedConnection[],
): TransformedConnection | null => {
	if (primarySources[path] == null) {
		return null;
	}
	return connections.find((connection) => connection.id === primarySources[path]) || null;
};

const getPropertyByPath = (schema: ObjectType, path: string): Property | null => {
	const fragments = path.split('.');
	let currentSchema = schema;
	for (const [index, fragment] of fragments.entries()) {
		const property = currentSchema.properties?.find((candidate) => candidate.name === fragment);
		if (property == null) {
			return null;
		}
		if (index === fragments.length - 1) {
			return property;
		}
		if (property.type.kind !== 'object') {
			return null;
		}
		currentSchema = property.type;
	}
	return null;
};

export { useSchemaGrid };
