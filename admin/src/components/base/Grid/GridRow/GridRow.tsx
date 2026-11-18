import React, { ReactNode } from 'react';
import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import { getChildIndexClassname } from '../Grid.helpers';
import { StandardGridRow, GridColumn } from '../Grid.types';

interface GridRowProps {
	row: StandardGridRow;
	columns: GridColumn[];
	className?: string;
	domID?: string;
	isInteractiveGrid?: boolean;
}

const GridRow = ({ row, columns, className, domID, isInteractiveGrid }: GridRowProps) => {
	const cellComponents = [] as ReactNode[];
	for (const [i, cell] of row.cells.entries()) {
		const type = columns[i].type;
		const alignment = columns[i].alignment;
		const typedCell = { value: cell, type: type, alignment: alignment };
		const className = getChildIndexClassname(i, row.cells.length);
		cellComponents.push(
			<GridCell
				key={columns[i].key ?? i}
				cell={typedCell}
				className={`grid__cell ${className}`}
				isInteractiveGrid={isInteractiveGrid}
			/>,
		);
	}

	return (
		<div
			id={domID}
			role={isInteractiveGrid ? 'row' : undefined}
			aria-selected={isInteractiveGrid ? row.active === true : undefined}
			className={`${className}${row.onClick ? ' grid__row--clickable' : ''}${row.active ? ' grid__row--active' : ''}${row.selected ? ' grid__row--selected' : ''}`}
			onClick={row.onClick}
			data-animation={row.animation}
			data-id={row.id}
		>
			{cellComponents}
		</div>
	);
};

export default GridRow;
