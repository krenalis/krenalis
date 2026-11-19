import React from 'react';
import '../Grid/GridKeyboardHints.css';

interface ComboboxKeyboardHintsProps {
	hasTabNavigation: boolean;
	hasOptions: boolean;
	highlight: number | 'tabs' | null;
}

const ComboboxKeyboardHints = ({ hasTabNavigation, hasOptions, highlight }: ComboboxKeyboardHintsProps) => {
	const hints: [string[], string][] = [];
	if (highlight === 'tabs') {
		hints.push([['←', '→'], 'Switch tab']);
		if (hasOptions) {
			hints.push([['↓'], 'Browse options']);
		}
	} else if (highlight == null && hasTabNavigation) {
		if (hasOptions) {
			hints.push([['↓'], 'Browse options']);
		}
		hints.push([['↑'], 'Go to tabs']);
	} else {
		hints.push([['↑', '↓'], 'Navigate']);
		if (highlight != null) {
			hints.push([['Enter'], 'Select']);
		}
	}
	return (
		<div
			className='grid-keyboard-hints combobox-list__hints'
			role='note'
			aria-label='Keyboard shortcuts'
			onMouseDown={(event) => event.preventDefault()}
		>
			{hints.map(([keys, label]) => (
				<span key={label} className='grid-keyboard-hints__hint'>
					<span className='grid-keyboard-hints__keys'>
						{keys.map((key) => (
							<kbd key={key}>{key}</kbd>
						))}
					</span>
					{label}
				</span>
			))}
		</div>
	);
};

export { ComboboxKeyboardHints };
