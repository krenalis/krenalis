import React, { ReactNode } from 'react';
import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import getChildIndexClassname from '../../../../lib/utils/getChildIndexClassname';
import { StandardGridRow, GridColumn } from '../../../../types/componentTypes/Grid.types';

interface GridRowProps {
	row: StandardGridRow;
	columns: GridColumn[];
	className?: string;
	id?: string;
}

const GridRow = ({ row, columns, className, id }: GridRowProps) => {
	const cellComponents = [] as ReactNode[];
	for (const [i, cell] of row.cells.entries()) {
		const type = columns[i].type;
		const alignment = columns[i].alignment;
		const typedCell = { value: cell, type: type, alignment: alignment };
		const className = getChildIndexClassname(i, row.cells.length);
		cellComponents.push(<GridCell key={i} cell={typedCell} className={`gridCell ${className}`} />);
	}

	return (
		<div
			key={row.key}
			className={`${className}${row.onClick ? ' clickable' : ''}${row.selected ? ' selected' : ''}`}
			onClick={row.onClick}
			data-animation={row.animation}
			data-id={id}
		>
			{cellComponents}
		</div>
	);
};

export default GridRow;
