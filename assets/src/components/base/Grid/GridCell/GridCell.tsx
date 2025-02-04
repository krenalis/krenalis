import React from 'react';
import './GridCell.css';
import { GridCell as GridCellInterface } from '../Grid.types';
import toJSDate from '../../../../utils/toJSDate';
import JSONbig from 'json-bigint';

interface GridCellProps {
	cell: GridCellInterface;
	className?: string;
}

const GridCell = ({ cell, className }: GridCellProps) => {
	let value;
	switch (cell.type) {
		case 'JSON':
		case 'Array':
		case 'Object':
		case 'Map':
			value = JSONbig.stringify(cell.value);
			break;
		case 'DateTime':
			if (cell.value != null) {
				const date = toJSDate(cell.value as string);
				value = date.toLocaleString();
			}
			break;
		case 'Date':
			if (cell.value != null) {
				const date = toJSDate(cell.value as string);
				value = date.toLocaleDateString();
			}
			break;
		case 'Time':
			value = cell.value;
			break;
		default:
			value = cell.value;
			break;
	}

	return (
		<div className={`${className}${cell.alignment != null ? ` grid__cell--${cell.alignment}` : ''}`}>
			<div className='grid__cell-content'>
				{cell.type === 'Object' ? <span className='grid__cell-content-object'> {value}</span> : value}
			</div>
		</div>
	);
};

export default GridCell;
