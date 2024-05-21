import React, { ReactNode } from 'react';
import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';
import { getChildIndexClassname } from '../Grid.helpers';
import { GridColumn } from '../Grid.types';

interface GridHeaderRowProps {
	columns: GridColumn[];
}

const GridHeaderRow = ({ columns }: GridHeaderRowProps) => {
	const gridHeaderCells = [] as ReactNode[];
	for (const [i, column] of columns.entries()) {
		const className = getChildIndexClassname(i, columns.length);
		gridHeaderCells.push(
			<GridHeaderCell
				key={column.name}
				className={`grid__header-cell ${className}`}
				value={column.name}
				alignment={column.alignment}
				explanation={column.explanation}
			/>,
		);
	}

	return <div className='grid__header-row'>{gridHeaderCells}</div>;
};

export default GridHeaderRow;
