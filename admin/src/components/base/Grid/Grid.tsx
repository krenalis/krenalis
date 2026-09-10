import React, { ReactNode, forwardRef, useMemo, useRef, useImperativeHandle, useState } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import {
	GridRow as GridRowType,
	GridColumn,
	GridNestedRowsIndentation,
	GridRef,
	NestedGridRows,
	SortableGridRow,
	SortableRowComponent,
	StandardGridRow,
} from './Grid.types';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { useGrid } from './useGrid';
import { getChildIndexClassname } from './Grid.helpers';
import {
	focusGridForKeyboardNavigation,
	navigateGrid,
	navigateGridWithKeyboard,
} from './GridKeyboardNavigation.helpers';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import GridRow from './GridRow/GridRow';
import { SortableRows } from './SortableRows';

interface GridReordering {
	disabled?: boolean;
	onSortRow: (overRowID: string, movedRowID: string) => void;
}

interface GridProps {
	columns: GridColumn[];
	rows: GridRowType[];
	showColumnBorder?: boolean;
	showRowBorder?: boolean;
	gridColumnsWidths?: string; // the widths of the columns in the 'grid-template-columns' CSS rule format.
	isLoading?: boolean;
	noRowsIcon?: string;
	noRowsMessage?: string;
	className?: string;

	// used to recompute the table if at first rendering it wasn't in the
	// viewport (for instance, because it was inside a tab panel group).
	isShown?: boolean;
	loadingText?: string;
	nestedRowsIndentation?: GridNestedRowsIndentation;
	keyboardNavigation?: boolean;
	reordering?: GridReordering;
}

const Grid = forwardRef<GridRef, GridProps>(
	(
		{
			columns,
			rows,
			showColumnBorder,
			showRowBorder,
			gridColumnsWidths,
			isLoading,
			noRowsIcon,
			noRowsMessage,
			className,
			isShown,
			loadingText,
			nestedRowsIndentation,
			keyboardNavigation,
			reordering,
		}: GridProps,
		ref,
	) => {
		const gridRef = useRef<any>();
		const [isScrolledVertically, setIsScrolledVertically] = useState(false);
		const onSortRow = reordering?.onSortRow;
		const reorderDisabled = reordering?.disabled;

		const { columnsWidths, reloadColumnsWidths } = useGrid(
			gridRef,
			rows,
			columns,
			gridColumnsWidths,
			isLoading,
			isShown,
		);

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
					const r = row as NestedGridRows;
					const parentRow = r[0] as SortableGridRow;
					const isSortable = onSortRow != null && parentRow.dragKey != null && parentRow.dragKey !== '';
					const component = (
						<GridNestedRows
							key={parentRow.id ?? i}
							rows={r}
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
				const sortableRow = row as SortableGridRow;
				const isSortable = onSortRow != null && sortableRow.dragKey != null && sortableRow.dragKey !== '';
				if (isSortable) {
					sortableRowComponents.push({
						id: sortableRow.dragKey,
						row: component,
					});
				} else {
					rowComponents.push(component);
				}
			}
			return { rowComponents, sortableRowComponents };
		}, [rows, nestedRowsIndentation, onSortRow, reorderDisabled]);

		let widths = columnsWidths;
		if (gridColumnsWidths != null) {
			widths = gridColumnsWidths;
		}

		return (
			<div
				ref={gridRef}
				className={`grid${onSortRow == null ? '' : ' grid--sortable'}${isScrolledVertically ? ' grid--scrolled-vertically' : ''}${className ? ' ' + className : ''}${showColumnBorder ? ' grid--show-column-border' : ''}${showRowBorder ? ' grid--show-row-border' : ''}${widths == null ? ' grid--hide-content' : ''}`}
				style={{ '--grid-columns': widths } as React.CSSProperties}
				tabIndex={keyboardNavigation ? 0 : undefined}
				onClick={keyboardNavigation ? focusGridForKeyboardNavigation : undefined}
				onScroll={(event) => setIsScrolledVertically(event.currentTarget.scrollTop > 0)}
				onKeyDown={
					keyboardNavigation
						? (event) => navigateGridWithKeyboard(event, reorderDisabled ? undefined : onSortRow)
						: undefined
				}
			>
				{isLoading ? (
					<div className='grid__loading'>
						<SlSpinner
							style={
								{
									fontSize: '3rem',
									'--track-width': '6px',
								} as React.CSSProperties
							}
						/>
						{loadingText != null && <div className='grid__loading-text'>{loadingText}</div>}
					</div>
				) : (
					<>
						<GridHeaderRow columns={columns} />
						{rows.length === 0 && noRowsMessage ? (
							<div className='grid__no-rows'>
								<div className='grid__no-rows-text'>
									<SlIcon name={noRowsIcon ?? 'exclamation-circle'}></SlIcon>
									{noRowsMessage}
								</div>
							</div>
						) : onSortRow != null ? (
							<>
								{rowComponents}
								<SortableRows
									className='grid__sortable-rows'
									disabled={reorderDisabled}
									onSortRow={onSortRow}
								>
									{sortableRowComponents}
								</SortableRows>
							</>
						) : rowComponents.length > 0 ? (
							rowComponents
						) : (
							<div className='grid__no-rows'>
								<div className='grid__no-rows-text'>
									<SlIcon name='exclamation-circle'></SlIcon>
									No rows to show
								</div>
							</div>
						)}
					</>
				)}
			</div>
		);
	},
);

export default Grid;
