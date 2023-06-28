import { useState, useLayoutEffect, useRef } from 'react';
import './Grid.css';
import GridHeaderRow from './GridHeaderRow/GridHeaderRow';
import GridRow from './GridRow/GridRow';
import GridNestedRows from './GridNestedRows/GridNestedRows';
import getChildIndexClassname from '../../../utils/getChildIndexClassname';
import { SlSpinner, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Grid = ({ columns, rows, isLoading, noRowsMessage }) => {
	const [columnsWidths, setColumnsWidths] = useState('');

	const gridRef = useRef();

	useLayoutEffect(() => {
		if (isLoading) {
			return;
		}
		const widthsOfColumn = {};
		for (let i = 0; i < columns.length; i++) {
			widthsOfColumn[i] = [];
		}
		const rows = gridRef.current.querySelectorAll('.gridHeaderRow, .gridRow');
		for (const r of rows) {
			const contents = r.querySelectorAll('.cellContent');
			for (const [i, c] of contents.entries()) {
				widthsOfColumn[i].push(c.offsetWidth);
			}
		}
		const maxWidths = [];
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

	const gridRows = [];
	for (const [i, row] of rows.entries()) {
		const className = getChildIndexClassname(i, rows.length);
		if (Array.isArray(row)) {
			gridRows.push(
				<GridNestedRows rows={row} columns={columns} className={`gridNestedRows ${className}`} nesting={1} />
			);
			continue;
		}
		gridRows.push(<GridRow row={row} columns={columns} className={`gridRow ${className}`} />);
	}

	return (
		<div ref={gridRef} className='grid' style={{ '--grid-columns': columnsWidths }}>
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
