import React from 'react';
import './GridKeyboardHints.css';

interface GridKeyboardHintsProps {
	canReorder?: boolean;
}

const GridKeyboardHints = ({ canReorder }: GridKeyboardHintsProps) => {
	return (
		<div className='grid-keyboard-hints' aria-label='Keyboard shortcuts'>
			<div className='grid-keyboard-hints__hint'>
				<span className='grid-keyboard-hints__keys'>
					<kbd>↑</kbd>
					<kbd>↓</kbd>
				</span>
				<span>Navigate</span>
			</div>
			<div className='grid-keyboard-hints__hint'>
				<span className='grid-keyboard-hints__keys'>
					<kbd>←</kbd>
					<kbd>→</kbd>
				</span>
				<span>Collapse / expand</span>
			</div>
			{canReorder && (
				<div className='grid-keyboard-hints__hint'>
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
