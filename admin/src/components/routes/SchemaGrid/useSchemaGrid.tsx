import React, { ReactNode, useMemo, useContext } from 'react';
import { ObjectType } from '../../../lib/api/types/types';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import AppContext from '../../../context/AppContext';
import TransformedConnection from '../../../lib/core/connection';
import { FlatSchema, TransformedProperty, flattenSchema } from '../../../lib/core/pipeline';
import { PrimarySources } from '../../../lib/api/types/workspace';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { toKrenalisStringType } from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const SCHEMA_COLUMNS: GridColumn[] = [{ name: 'Name' }, { name: 'Type' }, { name: 'Primary source' }];

const useSchemaGrid = (schema: ObjectType, isLoading: boolean) => {
	const { workspaces, selectedWorkspace, connections } = useContext(AppContext);

	const rows = useMemo(() => {
		if (isLoading) {
			return [];
		}
		const flatSchema = flattenSchema(schema);
		const w = workspaces.find((ws) => ws.id === selectedWorkspace);
		return getRows(flatSchema, w.primarySources, connections);
	}, [schema]);

	return {
		columns: SCHEMA_COLUMNS,
		rows: rows,
	};
};

const getRows = (
	schema: FlatSchema,
	primarySources: PrimarySources,
	connections: TransformedConnection[],
	parent?: string,
) => {
	const rows: GridRow[] = [];
	for (const k in schema) {
		if (!schema.hasOwnProperty(k)) {
			return;
		}
		const path = parent ? `${parent}.${k}` : k;
		const p = schema[k];
		if (p.indentation !== 0) {
			continue;
		}
		if (p.type === 'object') {
			const typ = p.full.type as ObjectType;
			if (typ.properties == null) {
				console.warn(`object property ${p.full.name} of the warehouse schema has empty properties`);
				continue;
			}
			const nestedRows: GridRow[] = [
				buildRow(p),
				...getRows(flattenSchema(typ), primarySources, connections, path),
			];
			rows.push(nestedRows);
		} else {
			let primarySource: TransformedConnection | null;
			if (primarySources[path]) {
				primarySource = connections.find((c) => c.id === primarySources[path]);
			}
			rows.push(buildRow(p, primarySource));
		}
	}

	return rows;
};

const buildRow = (property: TransformedProperty, primarySource?: TransformedConnection | null) => {
	const typeCell = toKrenalisStringType(property.full.type);
	let primarySourceCell: ReactNode;
	if (property.full.type.kind !== 'object' && property.full.type.kind !== 'array') {
		if (primarySource) {
			primarySourceCell = (
				<div className='schema-grid__primary-source'>
					<LittleLogo code={primarySource.connector.code} path={CONNECTORS_ASSETS_PATH} />
					{primarySource.name}
				</div>
			);
		} else {
			primarySourceCell = 'None';
		}
	}
	return { cells: [property.full.name, typeCell, primarySourceCell] };
};

export { useSchemaGrid };
