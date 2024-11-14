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
	id?: string; // the id inserted in the 'data-id' attribute of the row. Can be used to select the row via JS and CSS.
	key?: string;
	onClick?: () => void;
	animation?: string;
	selected?: boolean;
}

interface SortableGridRow extends StandardGridRow {
	dragKey: string; // the key used to identify the row in the drag and drop.
}

type NestedGridRows = GridRow[];

interface GridCell {
	value: ReactNode;
	type?: string;
	alignment?: string;
}

interface SortableRowComponent {
	id: string;
	row: ReactNode;
}

export type { GridColumn, GridRow, GridCell, StandardGridRow, NestedGridRows, SortableGridRow, SortableRowComponent };
