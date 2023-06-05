import { useState, useLayoutEffect, useRef } from 'react';
import './Grid.css';
import GridHeaderRow from '../GridHeaderRow/GridHeaderRow';
import GridRow from '../GridRow/GridRow';
import GridNestedRows from '../GridNestedRows/GridNestedRows';
import getChildIndexClassname from '../../utils/getChildIndexClassname';
import { SlSpinner, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Grid = ({ columns, rows, isLoading, noRowsMessage }) => {
	let [columnsWidths, setColumnsWidths] = useState('');

	let gridRef = useRef();

	useLayoutEffect(() => {
		if (isLoading) {
			return;
		}
		let widthsOfColumn = {};
		for (let i = 0; i < columns.length; i++) {
			widthsOfColumn[i] = [];
		}
		let rows = gridRef.current.querySelectorAll('.GridHeaderRow, .GridRow');
		for (let r of rows) {
			let contents = r.querySelectorAll('.cellContent');
			for (let [i, c] of contents.entries()) {
				widthsOfColumn[i].push(c.offsetWidth);
			}
		}
		let maxWidths = [];
		for (let k in widthsOfColumn) {
			let widths = widthsOfColumn[k];
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

	let gridRows = [];
	for (let [i, row] of rows.entries()) {
		let className = getChildIndexClassname(i, rows.length);
		if (Array.isArray(row)) {
			gridRows.push(
				<GridNestedRows rows={row} columns={columns} className={`GridNestedRows ${className}`} nesting={1} />
			);
			continue;
		}
		gridRows.push(<GridRow row={row} columns={columns} className={`GridRow ${className}`} />);
	}

	return (
		<div ref={gridRef} className='Grid' style={{ '--grid-columns': columnsWidths }}>
			{isLoading ? (
				<div className='loading'>
					<SlSpinner
						style={{
							fontSize: '3rem',
							'--track-width': '6px',
						}}
					/>
				</div>
			) : (
				<>
					<GridHeaderRow columns={columns} />
					{gridRows.length > 0 ? (
						gridRows
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
