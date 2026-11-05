import React, { CSSProperties, ReactNode, forwardRef, useId } from 'react';
import './GridHeaderCell.css';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

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
		const explanationID = useId();
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
						<>
							<WaTooltip className='grid__header-explanation-tooltip' for={explanationID} placement='top'>
								{explanation}
							</WaTooltip>
							<WaIcon
								id={explanationID}
								className='grid__header-explanation-icon'
								name='info-circle-fill'
							/>
						</>
					)}
					{reorderHandle}
				</div>
			</div>
		);
	},
);

export default GridHeaderCell;
