import React, { useState, ReactNode, Fragment } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { NestedGridRows, GridColumn, StandardGridRow } from '../../../../types/componentTypes/Grid.types';
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
					className='gridNestedRows children'
					nesting={nesting + 1}
				/>,
			);
		} else {
			let rowComponent: ReactNode;
			const r = row as StandardGridRow;
			if (i === 0) {
				rowComponent = (
					<Fragment key={i}>
						<SlIcon
							className='expand'
							name='caret-right-fill'
							onClick={() => {
								setIsExpanded(!isExpanded);
							}}
						></SlIcon>
						<GridRow row={r} columns={columns} className='gridRow parent' />
					</Fragment>
				);
			} else {
				rowComponent = <GridRow key={i} row={r} columns={columns} className='gridRow children' />;
			}
			rowComponents.push(rowComponent);
		}
	}

	const parentIndentation = 50 + 30 * (nesting - 1) + 'px'; // takes the indentation of the previous level.
	const childrenIndentation = 50 + 30 * nesting + 'px'; // takes the incremented indentation.
	return (
		<div
			className={`${className}${isExpanded ? ' expanded' : ''}`}
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
