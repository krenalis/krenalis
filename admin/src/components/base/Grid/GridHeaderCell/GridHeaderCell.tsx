import React, { CSSProperties, ReactNode, forwardRef } from 'react';
import './GridHeaderCell.css';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridHeaderCellProps {
	value: string;
	alignment?: string;
	className?: string;
	explanation?: string;
	reorderHandle?: ReactNode;
	semanticCell?: boolean;
	style?: CSSProperties;
}

const GridHeaderCell = forwardRef<HTMLDivElement, GridHeaderCellProps>(
	({ value, alignment, explanation, className, reorderHandle, semanticCell, style }, ref) => {
		return (
			<div
				ref={ref}
				role={semanticCell ? 'columnheader' : undefined}
				aria-label={semanticCell && reorderHandle != null ? value : undefined}
				className={`${className}${value === '' ? ' grid__header-cell--empty' : ''}${alignment != null ? ` grid__cell--${alignment}` : ''}`}
				style={style}
			>
				<div className='grid__cell-content'>
					{value}
					{explanation && (
						<SlTooltip className='grid__header-explanation-tooltip' content={explanation} placement='top'>
							<SlIcon className='grid__header-explanation-icon' name='info-circle-fill' />
						</SlTooltip>
					)}
					{reorderHandle}
				</div>
			</div>
		);
	},
);

export default GridHeaderCell;
