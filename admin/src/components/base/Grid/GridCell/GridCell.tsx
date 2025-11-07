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
		case 'json':
		case 'array':
		case 'object':
		case 'map':
		case 'decimal':
			value = JSONbig.stringify(cell.value);
			break;
		case 'datetime':
			if (cell.value != null) {
				const date = toJSDate(cell.value as string);
				value = date.toLocaleString();
			}
			break;
		case 'date':
			if (cell.value != null) {
				const date = toJSDate(cell.value as string);
				value = date.toLocaleDateString();
			}
			break;
		case 'time':
			value = cell.value;
			break;
		default:
			value = cell.value;
			break;
	}

	return (
		<div className={`${className}${cell.alignment != null ? ` grid__cell--${cell.alignment}` : ''}`}>
			<div className='grid__cell-content'>
				{cell.type === 'object' ? (
					<span className='grid__cell-content-object'> {value}</span>
				) : cell.type === 'html' ? (
					<span dangerouslySetInnerHTML={{ __html: value }} />
				) : (
					value
				)}
			</div>
		</div>
	);
};

export default GridCell;
