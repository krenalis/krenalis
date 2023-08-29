import React from 'react';
import './GridHeaderCell.css';

interface GridHeaderCellProps {
	value: string;
	alignment?: string;
	className?: string;
}

const GridHeaderCell = ({ value, alignment, className }: GridHeaderCellProps) => {
	return (
		<div className={`${className}${value === '' ? ' empty' : ''}${alignment != null ? ` ${alignment}` : ''}`}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridHeaderCell;
