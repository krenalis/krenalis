import React, { ReactNode, forwardRef, useMemo, useRef, useImperativeHandle, useState } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import {
	GridRow as GridRowType,
	GridColumn,
	GridKeyboardNavigationMode,
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
	scrollGridRowIntoView,
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
	keyboardNavigation?: GridKeyboardNavigationMode;
	reordering?: GridReordering;
	activeRowID?: string;
	ariaLabel?: string;
	gridID?: string;
	onNavigateActiveRow?: (direction: 'previous' | 'next') => void;
	onSortColumn?: (overColumnKey: string, movedColumnKey: string) => void;
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
			activeRowID,
			ariaLabel,
			gridID,
			onNavigateActiveRow,
			onSortColumn,
		}: GridProps,
		ref,
	) => {
		const gridRef = useRef<any>();
		const [isScrolledVertically, setIsScrolledVertically] = useState(false);
		const onSortRow = reordering?.onSortRow;
		const reorderDisabled = reordering?.disabled;
		const hasControlledActiveItem = onNavigateActiveRow != null;

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
						: navigateGrid(
								gridRef.current,
								key,
								shiftKey,
								keyboardNavigation ?? 'tree',
								reorderDisabled ? undefined : onSortRow,
							);
				},
				scrollRowIntoView: (id: string) => {
					const rows = gridRef.current?.querySelectorAll('.grid__row[data-id]') as
						| NodeListOf<HTMLElement>
						| undefined;
					const row =
						rows == null ? undefined : Array.from(rows).find((candidate) => candidate.dataset.id === id);
					if (row != null) {
						scrollGridRowIntoView(row);
					}
				},
			};
		}, [keyboardNavigation, onSortRow, reorderDisabled]);

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
						key={row.key ?? row.id ?? i}
						row={row as StandardGridRow}
						columns={columns}
						className={`grid__row${className ? ' ' + className : ''}`}
						domID={
							gridID != null && row.id != null ? `${gridID}-row-${encodeURIComponent(row.id)}` : undefined
						}
						semanticRow={hasControlledActiveItem}
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
		}, [rows, nestedRowsIndentation, onSortRow, reorderDisabled, gridID, hasControlledActiveItem]);

		let widths = columnsWidths;
		if (gridColumnsWidths != null) {
			widths = gridColumnsWidths;
		}
		const onGridKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
			if (hasControlledActiveItem) {
				if (
					event.target === event.currentTarget &&
					!event.altKey &&
					!event.ctrlKey &&
					!event.metaKey &&
					!event.shiftKey &&
					(event.key === 'ArrowDown' || event.key === 'ArrowUp')
				) {
					event.preventDefault();
					onNavigateActiveRow?.(event.key === 'ArrowDown' ? 'next' : 'previous');
				}
				return;
			}
			if (keyboardNavigation != null) {
				navigateGridWithKeyboard(event, keyboardNavigation, reorderDisabled ? undefined : onSortRow);
			}
		};

		return (
			<div
				ref={gridRef}
				id={gridID}
				className={`grid${onSortRow == null ? '' : ' grid--sortable'}${hasControlledActiveItem ? ' grid--active-items' : ''}${isScrolledVertically ? ' grid--scrolled-vertically' : ''}${className ? ' ' + className : ''}${showColumnBorder ? ' grid--show-column-border' : ''}${showRowBorder ? ' grid--show-row-border' : ''}${widths == null ? ' grid--hide-content' : ''}`}
				style={{ '--grid-columns': widths } as React.CSSProperties}
				role={hasControlledActiveItem ? 'grid' : undefined}
				aria-label={hasControlledActiveItem ? ariaLabel : undefined}
				aria-activedescendant={
					hasControlledActiveItem && activeRowID != null && activeRowID !== '' && gridID != null
						? `${gridID}-row-${encodeURIComponent(activeRowID)}`
						: undefined
				}
				tabIndex={keyboardNavigation || hasControlledActiveItem ? 0 : undefined}
				onClick={keyboardNavigation || hasControlledActiveItem ? focusGridForKeyboardNavigation : undefined}
				onScroll={(event) => setIsScrolledVertically(event.currentTarget.scrollTop > 0)}
				onKeyDown={keyboardNavigation || hasControlledActiveItem ? onGridKeyDown : undefined}
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
						<GridHeaderRow
							columns={columns}
							onSortColumn={onSortColumn}
							semanticRow={hasControlledActiveItem}
						/>
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
