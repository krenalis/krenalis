import { useState, useEffect, useRef } from 'react';
import './Grid.css';
import GridHeaderRow from '../GridHeaderRow/GridHeaderRow';
import GridRow from '../GridRow/GridRow';
import GridNestedRows from '../GridNestedRows/GridNestedRows';
import getChildIndexClassname from '../../utils/getChildIndexClassname';
import { SlSpinner, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Grid = ({ columns, rows, isLoading, noRowsMessage }) => {
	let [gridColumns, setGridColumns] = useState('');

	let gridRef = useRef();

	useEffect(() => {
		if (isLoading) {
			return;
		}
		setTimeout(() => {
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
			let gridColumns = '';
			for (let i = 0; i < maxWidths.length; i++) {
				if (i === 0) {
					gridColumns += `${maxWidths[i]}px`;
				} else {
					gridColumns += ` ${maxWidths[i]}px`;
				}
			}
			setGridColumns(gridColumns);
		}, 0); // queue the execution in case the grid is contained within a modal in the redering phase.
	}, [isLoading, rows, columns]);

	let gridRows = [];
	for (let [i, cells] of rows.entries()) {
		let className = getChildIndexClassname(i, rows.length);
		if (Array.isArray(cells[0])) {
			gridRows.push(<GridNestedRows rows={cells} className={`GridNestedRows ${className}`} nesting={1} />);
			continue;
		}
		gridRows.push(<GridRow cells={cells} className={`GridRow ${className}`} />);
	}

	return (
		<div ref={gridRef} className='Grid' style={{ '--grid-columns': gridColumns }}>
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
