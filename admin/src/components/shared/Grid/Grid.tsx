import React, { useState, useLayoutEffect, useRef, ReactNode } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import GridRow from './GridRow/GridRow';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import getChildIndexClassname from '../../../lib/utils/getChildIndexClassname';
import {
	GridRow as GridRowInterface,
	StandardGridRow,
	NestedGridRows,
	GridColumn,
} from '../../../types/componentTypes/Grid.types';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridProps {
	columns: GridColumn[];
	rows: GridRowInterface[];
	isLoading?: boolean;
	noRowsMessage?: string;
}

const Grid = ({ columns, rows, isLoading, noRowsMessage }: GridProps) => {
	const [columnsWidths, setColumnsWidths] = useState('');

	const gridRef = useRef<any>(null);

	useLayoutEffect(() => {
		if (isLoading) {
			return;
		}
		const widthsOfColumn = {};
		for (let i = 0; i < columns.length; i++) {
			widthsOfColumn[i] = [];
		}
		if (gridRef.current == null) return;
		const rowElements = gridRef.current.querySelectorAll('.gridHeaderRow, .gridRow');
		for (const r of rowElements) {
			const contents = r.querySelectorAll('.cellContent');
			for (const [i, c] of contents.entries()) {
				if (c instanceof HTMLElement) {
					widthsOfColumn[i].push(c.offsetWidth);
				}
			}
		}
		const maxWidths = [] as number[];
		for (const k in widthsOfColumn) {
			const widths = widthsOfColumn[k];
			maxWidths.push(Math.max(...widths) + 40); // 40 is the left/right padding of the cells.
		}
		let columnsWidths = '';
		for (let i = 0; i < maxWidths.length; i++) {
			if (i === 0) {
				columnsWidths += `${maxWidths[i]}px`;
			} else {
				columnsWidths += ` ${maxWidths[i]}px`;
			}
		}
		setColumnsWidths(columnsWidths);
	}, [isLoading, rows, columns]);

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
					className={`gridNestedRows ${className}`}
					nesting={1}
				/>,
			);
			continue;
		}
		const r = row as StandardGridRow;
		rowComponents.push(<GridRow key={i} row={r} columns={columns} className={`gridRow ${className}`} />);
	}

	return (
		<div ref={gridRef} className='grid' style={{ '--grid-columns': columnsWidths } as React.CSSProperties}>
			{isLoading ? (
				<div className='loading'>
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
						<div className='noRows'>
							<SlIcon name='exclamation-circle'></SlIcon>
							{noRowsMessage}
						</div>
					) : (
						<div className='noRows'>
							<SlIcon name='exclamation-circle'></SlIcon>
							No rows to show
						</div>
					)}
				</>
			)}
		</div>
	);
};

export default Grid;
