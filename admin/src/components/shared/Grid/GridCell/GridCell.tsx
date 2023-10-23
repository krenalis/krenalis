import React from 'react';
import './GridCell.css';
import { GridCell as GridCellInterface } from '../../../../types/componentTypes/Grid.types';
import toJSDateString from '../../../../lib/utils/toJSDateString';

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
			value = JSON.stringify(cell.value);
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
		<div className={`${className}${cell.alignment != null ? ` ${cell.alignment}` : ''}`}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridCell;
