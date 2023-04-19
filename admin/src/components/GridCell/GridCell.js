import { isValidElement } from 'react';
import './GridCell.css';

const GridCell = ({ value, className }) => {
	let isObject = typeof value === 'object' && !Array.isArray(value) && value !== null && !isValidElement(value);
	if (isObject) {
		value = JSON.stringify(value);
	}

	return (
		<div className={className}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridCell;
