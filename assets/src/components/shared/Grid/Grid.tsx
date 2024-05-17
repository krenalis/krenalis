import React, { ReactNode, forwardRef, useMemo, useRef, useEffect, useImperativeHandle } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import { GridRow as GridRowType, GridColumn, NestedGridRows, StandardGridRow } from './Grid.types';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { useGrid } from './useGrid';
import getChildIndexClassname from '../../../lib/utils/getChildIndexClassname';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import GridRow from './GridRow/GridRow';

interface GridProps {
	columns: GridColumn[];
	rows: GridRowType[];
	showColumnBorder?: boolean;
	showRowBorder?: boolean;
	isLoading?: boolean;
	noRowsMessage?: string;

	// used to recompute the table if at first rendering it wasn't in the
	// viewport (for instance, because it was inside a tab panel group).
	isShown?: boolean;
}

interface gridMethods {
	expand: () => void;
	collapse: () => void;
}

type GridRef = gridMethods & any;

const Grid = forwardRef<GridRef, GridProps>(
	({ columns, rows, showColumnBorder, showRowBorder, isLoading, noRowsMessage, isShown }, ref) => {
		const gridRef = useRef<any>();

		const { columnsWidths } = useGrid(gridRef, rows, columns, isLoading, isShown);

		useImperativeHandle(
			ref,
			() => {
				return {
					expand: () => {
						const nestedRows = gridRef.current.querySelectorAll('.grid__nested-rows');
						for (const r of nestedRows) {
							const isExpanded = r.classList.contains('grid__nested-rows--expanded');
							if (!isExpanded) {
								const expandIcon = r.querySelector('.grid__row-expand');
								expandIcon.click();
							}
						}
					},
					collapse: () => {
						const nestedRows = gridRef.current.querySelectorAll('.grid__nested-rows');
						for (const r of nestedRows) {
							const isExpanded = r.classList.contains('grid__nested-rows--expanded');
							if (isExpanded) {
								const expandIcon = r.querySelector('.grid__row-expand');
								expandIcon.click();
							}
						}
					},
				};
			},
			[],
		);

		useEffect(() => {
			ref = gridRef.current;
		}, []);

		const rowComponents = useMemo(() => {
			const rowComponents = [] as ReactNode[];
			for (const [i, row] of rows.entries()) {
				const className = getChildIndexClassname(i, rows.length);
				if (Array.isArray(row)) {
					const r = row as NestedGridRows;
					rowComponents.push(
						<GridNestedRows
							key={i}
							rows={r}
							columns={columns}
							className={`grid__nested-rows ${className}`}
							nesting={1}
						/>,
					);
					continue;
				}
				const r = row as StandardGridRow;
				rowComponents.push(
					<GridRow
						key={i}
						row={r}
						columns={columns}
						className={`grid__row${className ? ' ' + className : ''}`}
					/>,
				);
			}
			return rowComponents;
		}, [rows]);

		return (
			<div
				ref={gridRef}
				className={`grid${showColumnBorder ? ' grid--show-column-border' : ''}${showRowBorder ? ' grid--show-row-border' : ''}`}
				style={{ '--grid-columns': columnsWidths } as React.CSSProperties}
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
					</div>
				) : (
					<>
						<GridHeaderRow columns={columns} />
						{rowComponents.length > 0 ? (
							rowComponents
						) : noRowsMessage ? (
							<div className='grid__no-rows'>
								<SlIcon name='exclamation-circle'></SlIcon>
								{noRowsMessage}
							</div>
						) : (
							<div className='grid__no-rows'>
								<SlIcon name='exclamation-circle'></SlIcon>
								No rows to show
							</div>
						)}
					</>
				)}
			</div>
		);
	},
);

export default Grid;
