import { useState } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const GridNestedRows = ({ rows, columns, className, nesting }) => {
	let [isExpanded, setIsExpanded] = useState(false);

	let icon = (
		<SlIcon
			className='expand'
			name='caret-right-fill'
			onClick={() => {
				setIsExpanded(!isExpanded);
			}}
		></SlIcon>
	);

	let rws = [];
	for (let [i, cells] of rows.entries()) {
		if (Array.isArray(cells)) {
			rws.push(
				<GridNestedRows
					rows={cells}
					columns={columns}
					className='GridNestedRows children'
					nesting={nesting + 1}
				/>
			);
		} else {
			let row;
			if (i === 0) {
				row = (
					<>
						{icon}
						<GridRow row={cells} columns={columns} className='GridRow parent' />
					</>
				);
			} else {
				row = <GridRow row={cells} columns={columns} className='GridRow children' />;
			}
			rws.push(row);
		}
	}

	let parentIndentation = 50 + 30 * (nesting - 1) + 'px'; // takes the indentation of the previous level.
	let childrenIndentation = 50 + 30 * nesting + 'px'; // takes the incremented indentation.
	return (
		<div
			className={`${className}${isExpanded ? ' expanded' : ''}`}
			style={{
				'--parent-indentation': parentIndentation,
				'--children-indentation': childrenIndentation,
			}}
		>
			{rws}
		</div>
	);
};

export default GridNestedRows;
