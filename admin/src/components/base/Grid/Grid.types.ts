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
	onToggleExpansion?: () => void;
	animation?: string;
	selected?: boolean;
	expanded?: boolean;
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

interface GridNestedRowsIndentation {
	base: number;
	step: number;
}

interface GridRef {
	collapse: () => void;
	expand: () => void;
	focus: () => void;
	navigate: (key: string, shiftKey?: boolean) => boolean;
}

interface SortableGridRef extends GridRef {
	expandRow: (id: string) => void;
}

interface SortableRowComponent {
	id: string;
	row: ReactNode;
}

export type {
	GridColumn,
	GridRow,
	GridCell,
	GridNestedRowsIndentation,
	GridRef,
	StandardGridRow,
	NestedGridRows,
	SortableGridRef,
	SortableGridRow,
	SortableRowComponent,
};
