import React from 'react';
import './GridKeyboardHints.css';

interface GridKeyboardHintsProps {
	canReorder?: boolean;
	disabled?: boolean;
	expansionDisabled?: boolean;
	reorderDisabled?: boolean;
}

const GridKeyboardHints = ({ canReorder, disabled, expansionDisabled, reorderDisabled }: GridKeyboardHintsProps) => {
	const isExpansionDisabled = disabled || expansionDisabled;
	const isReorderDisabled = disabled || reorderDisabled;

	return (
		<div className='grid-keyboard-hints' aria-label='Keyboard shortcuts'>
			<div
				className={`grid-keyboard-hints__hint${disabled ? ' grid-keyboard-hints__hint--disabled' : ''}`}
				aria-disabled={disabled || undefined}
			>
				<span className='grid-keyboard-hints__keys'>
					<kbd>↑</kbd>
					<kbd>↓</kbd>
				</span>
				<span>Navigate</span>
			</div>
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
			{canReorder && (
				<div
					className={`grid-keyboard-hints__hint${isReorderDisabled ? ' grid-keyboard-hints__hint--disabled' : ''}`}
					aria-disabled={isReorderDisabled || undefined}
				>
					<span className='grid-keyboard-hints__keys'>
						<kbd>Shift</kbd>
						<span className='grid-keyboard-hints__plus'>+</span>
						<kbd>↑</kbd>
						<kbd>↓</kbd>
					</span>
					<span>Reorder</span>
				</div>
			)}
		</div>
	);
};

export { GridKeyboardHints };
