import { ReactNode } from 'react';

interface GridColumn {
	name: string;
	type?: string;
	alignment?: string;
}

type GridRow = StandardGridRow | NestedGridRows;

interface StandardGridRow {
	cells: ReactNode[];
	key?: string;
	onClick?: () => void;
	animation?: string;
}

type NestedGridRows = GridRow[];

interface GridCell {
	value: ReactNode;
	type?: string;
	alignment?: string;
}

export { GridColumn, GridRow, GridCell, StandardGridRow, NestedGridRows };
