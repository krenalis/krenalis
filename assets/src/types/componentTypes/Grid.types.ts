import { ReactNode } from 'react';

interface GridColumn {
	name: string;
	type?: string;
	alignment?: 'left' | 'center' | 'right' | 'header-left' | 'header-center' | 'header-right';
	explanation?: string;
}

type GridRow = StandardGridRow | NestedGridRows;

interface StandardGridRow {
	cells: ReactNode[];
	key?: string;
	onClick?: () => void;
	animation?: string;
	selected?: boolean;
}

interface SortableGridRow extends StandardGridRow {
	isSortable: boolean;
	dragKey: string; // the key used to identify the row in the drag and drop.
	id: string;
}

type NestedGridRows = GridRow[];

interface GridCell {
	value: ReactNode;
	type?: string;
	alignment?: string;
}

export type { GridColumn, GridRow, GridCell, StandardGridRow, NestedGridRows, SortableGridRow };
