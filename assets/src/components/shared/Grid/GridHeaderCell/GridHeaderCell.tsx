import React from 'react';
import './GridHeaderCell.css';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridHeaderCellProps {
	value: string;
	alignment?: string;
	className?: string;
	explanation?: string;
}

const GridHeaderCell = ({ value, alignment, explanation, className }: GridHeaderCellProps) => {
	return (
		<div
			className={`${className}${value === '' ? ' grid__header-cell--empty' : ''}${alignment != null ? ` grid__cell--${alignment}` : ''}`}
		>
			<div className='grid__cell-content'>
				{value}
				{explanation && (
					<SlTooltip className='grid__header-explanation-tooltip' content={explanation} placement='top'>
						<SlIcon className='grid__header-explanation-icon' name='info-circle-fill' />
					</SlTooltip>
				)}
			</div>
		</div>
	);
};

export default GridHeaderCell;
