import React, { ReactNode } from 'react';
import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import { getChildIndexClassname } from '../Grid.helpers';
import { StandardGridRow, GridColumn } from '../Grid.types';

interface GridRowProps {
	row: StandardGridRow;
	columns: GridColumn[];
	className?: string;
}

const GridRow = ({ row, columns, className }: GridRowProps) => {
	const cellComponents = [] as ReactNode[];
	for (const [i, cell] of row.cells.entries()) {
		const type = columns[i].type;
		const alignment = columns[i].alignment;
		const typedCell = { value: cell, type: type, alignment: alignment };
		const className = getChildIndexClassname(i, row.cells.length);
		cellComponents.push(<GridCell key={i} cell={typedCell} className={`grid__cell ${className}`} />);
	}

	return (
		<div
			key={row.key}
			className={`${className}${row.onClick ? ' grid__row--clickable' : ''}${row.selected ? ' grid__row--selected' : ''}`}
			onClick={row.onClick}
			data-animation={row.animation}
			data-id={row.id}
		>
			{cellComponents}
		</div>
	);
};

export default GridRow;
