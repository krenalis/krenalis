import React from 'react';
import './GridCell.css';
import { GridCell as GridCellInterface } from '../Grid.types';
import toJSDateString from '../../../../utils/toJSDateString';
import JSONbig from 'json-bigint';

interface GridCellProps {
	cell: GridCellInterface;
	className?: string;
}

const GridCell = ({ cell, className }: GridCellProps) => {
	let value, date;
	switch (cell.type) {
		case 'JSON':
		case 'Array':
		case 'Object':
		case 'Map':
			value = JSONbig.stringify(cell.value);
			break;
		case 'DateTime':
			if (cell.value != null) {
				date = new Date(toJSDateString(cell.value as string));
				value = date.toLocaleString('it-IT', { timeZone: 'Europe/Rome' });
			}
			break;
		case 'Date':
			if (cell.value != null) {
				date = new Date(toJSDateString(cell.value as string));
				value = date.toLocaleDateString('it-IT', { timeZone: 'Europe/Rome' });
			}
			break;
		case 'Time':
			if (cell.value != null) {
				date = new Date(toJSDateString(cell.value as string));
				value = date.toLocaleTimeString('it-IT', { timeZone: 'Europe/Rome' });
			}
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
