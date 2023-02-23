import React from 'react';
import './Grid.css';
import GridHeaderRow from '../GridHeaderRow/GridHeaderRow';
import GridRow from '../GridRow/GridRow';
import GridNestedRows from '../GridNestedRows/GridNestedRows';
import getChildIndexClassname from '../../utils/getChildIndexClassname';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const Grid = ({ columns, rows, isLoading, actions }) => {
	let gridRows = [];
	for (let [i, cells] of rows.entries()) {
		let className = getChildIndexClassname(i, rows.length);
		if (Array.isArray(cells[0])) {
			gridRows.push(<GridNestedRows rows={cells} className={`GridNestedRows ${className}`} nesting={1} />);
			continue;
		}
		gridRows.push(<GridRow cells={cells} className={`GridRow ${className}`} actions={actions} />);
	}

	let columnsLength = columns.length;
	if (actions) columnsLength += 1;

	return (
		<div className='Grid' style={{ '--grid-columns': `repeat(${columnsLength}, 1fr)` }}>
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
