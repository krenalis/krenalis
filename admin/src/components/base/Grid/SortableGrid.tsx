import React, { ReactNode, useMemo, useRef, useState, useImperativeHandle, forwardRef } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import {
	GridColumn,
	GridNestedRowsIndentation,
	NestedGridRows,
	SortableGridRef,
	StandardGridRow,
	SortableGridRow,
	SortableRowComponent,
} from './Grid.types';
import { useGrid } from './useGrid';
import { getChildIndexClassname } from './Grid.helpers';
import { focusGridForKeyboardNavigation, navigateGrid, navigateGridWithKeyboard } from './GridKeyboardNavigation';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import GridRow from './GridRow/GridRow';
import {
	DndContext,
	closestCenter,
	KeyboardSensor,
	PointerSensor,
	useSensor,
	useSensors,
	DragOverlay,
} from '@dnd-kit/core';
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { restrictToVerticalAxis, restrictToParentElement } from '@dnd-kit/modifiers';
import { DraggableWrapper } from './DraggableWrapper/DraggableWrapper';
import { OverlayRow } from '../OverlayRow/OverlayRow';

interface SortableGridProps {
	columns: GridColumn[];
	rows: SortableGridRow[];
	onSortRow: (overRowID: string, movedRowID: string) => void;
	gridColumnsWidths?: string;
	nestedRowsIndentation?: GridNestedRowsIndentation;
	keyboardNavigation?: boolean;
	reorderDisabled?: boolean;
}

const SortableGrid = forwardRef<SortableGridRef, SortableGridProps>(
	(
		{ columns, rows, onSortRow, gridColumnsWidths, nestedRowsIndentation, keyboardNavigation, reorderDisabled },
		ref,
	) => {
		const [activeRow, setActiveRow] = useState(null);
		const sensors = useSensors(
			useSensor(PointerSensor),
			useSensor(KeyboardSensor, {
				coordinateGetter: sortableKeyboardCoordinates,
			}),
		);

		const gridRef = useRef<any>(null);

		const { columnsWidths, reloadColumnsWidths } = useGrid(gridRef, rows, columns, gridColumnsWidths);

		useImperativeHandle(ref, () => {
			return {
				collapse: () => {
					const nestedRows = gridRef.current.querySelectorAll('.grid__nested-rows');
					for (const r of nestedRows) {
						const isExpanded = r.classList.contains('grid__nested-rows--expanded');
						if (isExpanded) {
							const expandIcon = r.querySelector('.grid__row-expand');
							expandIcon.click();
						}
					}
					reloadColumnsWidths();
				},
				expand: () => {
					const nestedRows = gridRef.current.querySelectorAll('.grid__nested-rows');
					for (const r of nestedRows) {
						const isExpanded = r.classList.contains('grid__nested-rows--expanded');
						if (!isExpanded) {
							const expandIcon = r.querySelector('.grid__row-expand');
							expandIcon.click();
						}
					}
					reloadColumnsWidths();
				},
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
				focus: () => {
					gridRef.current?.focus({ preventScroll: true });
				},
				navigate: (key: string, shiftKey = false) => {
					return gridRef.current == null
						? false
						: navigateGrid(gridRef.current, key, shiftKey, reorderDisabled ? undefined : onSortRow);
				},
			};
		}, [onSortRow, reorderDisabled]);

		const { rowComponents, sortableRowComponents } = useMemo(() => {
			const rowComponents = [] as ReactNode[];
			const sortableRowComponents = [] as SortableRowComponent[];
			for (const [i, row] of rows.entries()) {
				const className = getChildIndexClassname(i, rows.length);
				if (Array.isArray(row)) {
					const nestedRows = row as NestedGridRows;
					const parentRow = nestedRows[0] as SortableGridRow;
					const isSortable = parentRow.dragKey != null && parentRow.dragKey !== '';
					const component = (
						<GridNestedRows
							key={parentRow.id ?? i}
							rows={nestedRows}
							columns={columns}
							className={`grid__nested-rows ${className}`}
							nesting={1}
							onSortRow={onSortRow}
							isSortable={isSortable}
							reorderDisabled={reorderDisabled}
							indentation={nestedRowsIndentation}
							reloadColumnsWidths={reloadColumnsWidths}
						/>
					);
					if (isSortable) {
						sortableRowComponents.push({
							id: parentRow.dragKey,
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
		}, [rows, nestedRowsIndentation, reorderDisabled]);

		function onDragEnd(e) {
			const { over, active } = e;
			if (over != null && over.id !== active.id) {
				onSortRow(over.id, active.id);
			}
			setActiveRow(null);
		}

		function onDragStart(e) {
			const { active } = e;
			setActiveRow(active.id);
		}

		let widths = columnsWidths;
		if (gridColumnsWidths != null) {
			widths = gridColumnsWidths;
		}

		return (
			<div
				ref={gridRef}
				className={`grid grid--sortable${widths == null ? ' grid--hide-content' : ''}`}
				style={{ '--grid-columns': widths } as React.CSSProperties}
				tabIndex={keyboardNavigation ? 0 : undefined}
				onClick={keyboardNavigation ? focusGridForKeyboardNavigation : undefined}
				onKeyDown={
					keyboardNavigation
						? (event) => navigateGridWithKeyboard(event, reorderDisabled ? undefined : onSortRow)
						: undefined
				}
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
								<DraggableWrapper key={id} id={id} disabled={reorderDisabled}>
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
	},
);

export default SortableGrid;
