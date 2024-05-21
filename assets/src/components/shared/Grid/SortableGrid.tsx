import React, { ReactNode, useMemo, useRef, useState, useImperativeHandle, forwardRef, useEffect } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import { GridColumn, NestedGridRows, StandardGridRow, SortableGridRow } from './Grid.types';
import { useGrid } from './useGrid';
import { getChildIndexClassname } from './Grid.helpers';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import GridRow from './GridRow/GridRow';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import {
	DndContext,
	closestCenter,
	KeyboardSensor,
	PointerSensor,
	useSensor,
	useSensors,
	DragOverlay,
} from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import {
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { restrictToVerticalAxis, restrictToParentElement } from '@dnd-kit/modifiers';

interface SortableRowComponent {
	id: string;
	row: ReactNode;
}

interface SortableGridProps {
	columns: GridColumn[];
	rows: SortableGridRow[];
	onSortRow: (overRowID: string, movedRowID: string) => void;
	showColumnBorder?: boolean;
	showRowBorder?: boolean;
	isLoading?: boolean;
	noRowsMessage?: string;

	// used to recompute the table if at first rendering it wasn't in the
	// viewport (for instance, because it was inside a tab panel group).
	isShown?: boolean;
}

interface SortableGridMethods {
	expandRow: (id: string) => void;
}

type SortableGridRef = SortableGridMethods & any;

const SortableGrid = forwardRef<SortableGridRef, SortableGridProps>(({ columns, rows, onSortRow }, ref) => {
	const [activeRow, setActiveRow] = useState(null);
	const sensors = useSensors(
		useSensor(PointerSensor),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);

	const gridRef = useRef<any>(null);

	const { columnsWidths } = useGrid(gridRef, rows, columns);

	useImperativeHandle(
		ref,
		() => {
			return {
				expandRow: (id: string) => {
					const row = gridRef.current.querySelector(`[data-id="${id}"]`);
					const parent = row.closest('.grid__nested-rows');
					if (parent == null) {
						return;
					}
					const isExpanded = parent.classList.contains('grid__nested-rows--expanded');
					if (!isExpanded) {
						const expandIcon = parent.querySelector('.grid__row-expand');
						expandIcon.click();
					}
				},
			};
		},
		[],
	);

	useEffect(() => {
		ref = gridRef.current;
	}, []);

	const { rowComponents, sortableRowComponents } = useMemo(() => {
		const rowComponents = [] as ReactNode[];
		const sortableRowComponents = [] as SortableRowComponent[];
		for (const [i, row] of rows.entries()) {
			const className = getChildIndexClassname(i, rows.length);
			if (Array.isArray(row)) {
				const component = (
					<GridNestedRows
						key={i}
						rows={row as NestedGridRows}
						columns={columns}
						className={`grid__nested-rows ${className}`}
						nesting={1}
					/>
				);
				const isSortable = row[0].dragKey != null && row[0].dragKey !== '';
				if (isSortable) {
					sortableRowComponents.push({
						id: row[0].dragKey,
						row: component,
					});
				} else {
					rowComponents.push(component);
				}
				continue;
			}
			const component = (
				<GridRow
					key={i}
					row={row as StandardGridRow}
					columns={columns}
					className={`grid__row${className ? ' ' + className : ''}`}
					id={row.id}
				/>
			);
			const isSortable = row.dragKey != null && row.dragKey !== '';
			if (isSortable) {
				sortableRowComponents.push({
					id: row.dragKey,
					row: component,
				});
			} else {
				rowComponents.push(component);
			}
		}
		return { rowComponents, sortableRowComponents };
	}, [rows]);

	function onDragEnd(e) {
		const { over, active } = e;
		if (over.id !== active.id) {
			onSortRow(over.id, active.id);
		}
		setActiveRow(null);
	}

	function onDragStart(e) {
		const { active } = e;
		setActiveRow(active.id);
	}

	return (
		<div
			ref={gridRef}
			className='grid grid--sortable'
			style={{ '--grid-columns': columnsWidths } as React.CSSProperties}
		>
			<GridHeaderRow columns={columns} />
			{rowComponents}
			<div className='grid__sortable-rows'>
				<DndContext
					sensors={sensors}
					collisionDetection={closestCenter}
					modifiers={[restrictToVerticalAxis, restrictToParentElement]}
					onDragStart={onDragStart}
					onDragEnd={onDragEnd}
				>
					<SortableContext items={sortableRowComponents} strategy={verticalListSortingStrategy}>
						{sortableRowComponents.map(({ id, row }) => (
							<DraggableWrapper key={id} id={id}>
								{row}
							</DraggableWrapper>
						))}
					</SortableContext>
					<DragOverlay>
						{activeRow ? (
							<OverlayRow>{sortableRowComponents.find((c) => c.id === activeRow).row}</OverlayRow>
						) : null}
					</DragOverlay>
				</DndContext>
			</div>
		</div>
	);
});

interface DraggableRowProps {
	id: string;
	children: ReactNode;
}

const DraggableWrapper = ({ id, children }: DraggableRowProps) => {
	const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: id });

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
	};

	return (
		<div className='draggable-wrapper' ref={setNodeRef} style={style}>
			<button className='draggable-wrapper__handle' {...listeners} {...attributes}>
				<SlIcon name='grip-vertical' />
			</button>
			{children}
		</div>
	);
};

interface OverlayRowProps {
	children: ReactNode;
}

const OverlayRow = ({ children }: OverlayRowProps) => {
	return (
		<div className='overlay-row'>
			<button className='overlay-row__handle'>
				<SlIcon name='grip-vertical' />
			</button>
			<div className='overlay-row__name'>{children}</div>
		</div>
	);
};

export default SortableGrid;
export { SortableGridRef };
