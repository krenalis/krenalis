import React, { ReactNode } from 'react';
import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';
import getChildIndexClassname from '../../../../lib/utils/getChildIndexClassname';
import { GridColumn } from '../../../../types/componentTypes/Grid.types';

interface GridHeaderRowProps {
	columns: GridColumn[];
}

const GridHeaderRow = ({ columns }: GridHeaderRowProps) => {
	const gridHeaderCells = [] as ReactNode[];
	for (const [i, column] of columns.entries()) {
		const className = getChildIndexClassname(i, columns.length);
		gridHeaderCells.push(
			<GridHeaderCell
				className={`gridHeaderCell ${className}`}
				value={column.name}
				alignment={column.alignment}
			/>,
		);
	}

	return <div className='gridHeaderRow'>{gridHeaderCells}</div>;
};

export default GridHeaderRow;
