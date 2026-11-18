import React from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import './GridKeyboardHints.css';

interface GridKeyboardHintsProps {
	canExpand?: boolean;
	canReorder?: boolean;
	disabled?: boolean;
	expansionDisabled?: boolean;
	navigationAriaLabel?: string;
	navigationLabel?: string;
	reorderDisabled?: boolean;
}

const GridKeyboardHints = ({
	canExpand = true,
	canReorder,
	disabled,
	expansionDisabled,
	navigationAriaLabel,
	navigationLabel = 'Navigate',
	reorderDisabled,
}: GridKeyboardHintsProps) => {
	const hasNavigationAriaLabel = navigationAriaLabel !== undefined;
	const isExpansionDisabled = disabled || expansionDisabled;
	const isReorderDisabled = disabled || reorderDisabled;

	return (
		<div className='grid-keyboard-hints' aria-label={hasNavigationAriaLabel ? undefined : 'Keyboard shortcuts'}>
			<div
				className={`grid-keyboard-hints__hint${disabled ? ' grid-keyboard-hints__hint--disabled' : ''}`}
				aria-disabled={disabled || undefined}
				aria-label={navigationAriaLabel}
				role={hasNavigationAriaLabel ? 'note' : undefined}
			>
				<span className='grid-keyboard-hints__keys' aria-hidden={hasNavigationAriaLabel || undefined}>
					<kbd>
						<WaIcon name='arrow-up' label='Up arrow' />
					</kbd>
					<kbd>
						<WaIcon name='arrow-down' label='Down arrow' />
					</kbd>
				</span>
				<span aria-hidden={hasNavigationAriaLabel || undefined}>{navigationLabel}</span>
			</div>
			{canExpand && (
				<div
					className={`grid-keyboard-hints__hint${isExpansionDisabled ? ' grid-keyboard-hints__hint--disabled' : ''}`}
					aria-disabled={isExpansionDisabled || undefined}
				>
					<span className='grid-keyboard-hints__keys'>
						<kbd>←</kbd>
						<kbd>→</kbd>
					</span>
					<span>Collapse / expand</span>
				</div>
			)}
			{canReorder && (
				<div
					className={`grid-keyboard-hints__hint${isReorderDisabled ? ' grid-keyboard-hints__hint--disabled' : ''}`}
					aria-disabled={isReorderDisabled || undefined}
				>
					<span className='grid-keyboard-hints__keys'>
						<kbd>Shift</kbd>
						<span className='grid-keyboard-hints__plus'>+</span>
						<kbd>
							<WaIcon name='arrow-up' label='Up arrow' />
						</kbd>
						<kbd>
							<WaIcon name='arrow-down' label='Down arrow' />
						</kbd>
					</span>
					<span>Reorder</span>
				</div>
			)}
		</div>
	);
};

export { GridKeyboardHints };
