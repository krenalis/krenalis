import React, { useId } from 'react';
import './GridHeaderCell.css';
import SlTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridHeaderCellProps {
	value: string;
	alignment?: string;
	className?: string;
	explanation?: string;
}

const GridHeaderCell = ({ value, alignment, explanation, className }: GridHeaderCellProps) => {
	const explanationID = useId();
	return (
		<div
			className={`${className}${value === '' ? ' grid__header-cell--empty' : ''}${alignment != null ? ` grid__cell--${alignment}` : ''}`}
		>
			<div className='grid__cell-content'>
				{value}
				{explanation && (
					<>
						<SlTooltip className='grid__header-explanation-tooltip' for={explanationID} placement='top'>
							{explanation}
						</SlTooltip>
						<SlIcon id={explanationID} className='grid__header-explanation-icon' name='info-circle-fill' />
					</>
				)}
			</div>
		</div>
	);
};

export default GridHeaderCell;
