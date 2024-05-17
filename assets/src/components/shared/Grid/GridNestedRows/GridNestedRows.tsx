import React, { useState, ReactNode, Fragment } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { NestedGridRows, GridColumn } from '../Grid.types';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridNestedRowsProps {
	rows: NestedGridRows;
	columns: GridColumn[];
	className?: string;
	nesting: number;
}

const GridNestedRows = ({ rows, columns, className, nesting }: GridNestedRowsProps) => {
	const [isExpanded, setIsExpanded] = useState(false);

	const rowComponents = [] as ReactNode[];
	for (const [i, row] of rows.entries()) {
		if (Array.isArray(row)) {
			const r = row as NestedGridRows;
			rowComponents.push(
				<GridNestedRows
					key={i}
					rows={r}
					columns={columns}
					className='grid__nested-rows grid__nested-rows--children'
					nesting={nesting + 1}
				/>,
			);
		} else {
			let rowComponent: ReactNode;
			const r = row as any;
			if (i === 0) {
				rowComponent = (
					<Fragment key={i}>
						<SlIcon
							className='grid__row-expand'
							name='caret-right-fill'
							onClick={() => {
								setIsExpanded(!isExpanded);
							}}
						></SlIcon>
						<GridRow row={r} columns={columns} className='grid__row grid__row--parent' id={r.id} />
					</Fragment>
				);
			} else {
				rowComponent = (
					<GridRow key={i} row={r} columns={columns} className='grid__row grid__row--children' id={r.id} />
				);
			}
			rowComponents.push(rowComponent);
		}
	}

	const parentIndentation = 50 + 30 * (nesting - 1) + 'px'; // takes the indentation of the previous level.
	const childrenIndentation = 50 + 30 * nesting + 'px'; // takes the incremented indentation.
	return (
		<div
			className={`${className}${isExpanded ? ' grid__nested-rows--expanded' : ''}`}
			style={
				{
					'--parent-indentation': parentIndentation,
					'--children-indentation': childrenIndentation,
				} as React.CSSProperties
			}
		>
			{rowComponents}
		</div>
	);
};

export default GridNestedRows;
