import React from 'react';
import './Grid.css';
import GridHeaderRow from '../GridHeaderRow/GridHeaderRow';
import GridRow from '../GridRow/GridRow';
import GridNestedRows from '../GridNestedRows/GridNestedRows';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const Grid = ({ columns, rows, isLoading }) => {
	let gridRows = [];
	for (let [i, cells] of rows.entries()) {
		let index = i + 1;
		let className = '';
		if (index === 1) {
			className += ' firstRow';
		}
		if (index === rows.length) {
			className += ' lastRow';
		}
		if (index % 2 === 0) {
			className += ' even';
		} else {
			className += ' odd';
		}
		if (Array.isArray(cells[0])) {
			gridRows.push(<GridNestedRows rows={cells} className={`GridNestedRows ${className}`} />);
			continue;
		}
		gridRows.push(<GridRow cells={cells} className={`GridRow ${className}`} />);
	}

	return (
		<div className='Grid' style={{ '--grid-style': `repeat(${columns.length}, 1fr)` }}>
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
					{gridRows.length > 0 ? gridRows : <div className='noRows'>No rows to show</div>}
				</>
			)}
		</div>
	);
};

export default Grid;
