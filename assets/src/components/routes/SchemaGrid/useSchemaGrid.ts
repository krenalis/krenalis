import { ReactNode, useMemo } from 'react';
import { ObjectType, Property } from '../../../types/external/types';
import { GridColumn, GridRow } from '../../shared/Grid/Grid.types';
import { enrichPropertyType } from '../../helpers/enrichPropertyType';

const SCHEMA_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Type' },
	{ name: 'Nullable', alignment: 'center' },
	{ name: 'Label' },
];

const useSchemaGrid = (schema: ObjectType, isLoading: boolean) => {
	const rows = useMemo(() => {
		if (isLoading) {
			return [];
		}
		return getRows(schema.properties);
	}, [schema]);

	return {
		columns: SCHEMA_COLUMNS,
		rows: rows,
	};
};

const getRows = (properties: Property[]) => {
	const rows: GridRow[] = [];
	for (const p of properties) {
		if (p.type.name === 'Object') {
			const typ = p.type as ObjectType;
			if (typ.properties == null) {
				console.warn(`Object property ${p.name} of the warehouse schema has empty properties`);
				continue;
			}
			const nestedRows: GridRow[] = [buildRow(p), ...getRows(typ.properties)];
			rows.push(nestedRows);
		} else {
			rows.push(buildRow(p));
		}
	}

	return rows;
};

const buildRow = (property: Property) => {
	const typeCell = enrichPropertyType(property.type);
	let nullableCell: ReactNode;
	if (property.nullable) {
		nullableCell = 'Yes';
	} else {
		nullableCell = 'No';
	}
	return { cells: [property.name, typeCell, nullableCell, property.label] };
};

export { useSchemaGrid };
