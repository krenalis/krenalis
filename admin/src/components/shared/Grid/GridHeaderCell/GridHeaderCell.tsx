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
		<div className={`${className}${value === '' ? ' empty' : ''}${alignment != null ? ` ${alignment}` : ''}`}>
			<div className='cellContent'>
				{value}
				{explanation && (
					<SlTooltip className='gridHeaderExplanationTooltip' content={explanation} placement='top'>
						<SlIcon className='gridHeaderExplanationIcon' name='info-circle-fill' />
					</SlTooltip>
				)}
			</div>
		</div>
	);
};

export default GridHeaderCell;
