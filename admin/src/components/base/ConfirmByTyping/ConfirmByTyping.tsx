import React, { useRef, useEffect } from 'react';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import './ConfirmByTyping.css';

interface ConfirmByTypingProps {
	confirmText: string;
	value: string;
	onInput: (value: string) => void;
}

const ConfirmByTyping = ({ confirmText, value, onInput }: ConfirmByTypingProps) => {
	const inputRef = useRef<any>();

	useEffect(() => {
		const input = inputRef.current;
		if (!input) return;

		const dialog = input.closest('sl-dialog');
		if (dialog) {
			const onAfterShow = () => input.focus();
			dialog.addEventListener('sl-after-show', onAfterShow);
			return () => dialog.removeEventListener('sl-after-show', onAfterShow);
		} else {
			setTimeout(() => input.focus(), 50);
		}
	}, []);

	return (
		<div className='confirm-by-typing'>
			<p className='confirm-by-typing__instruction'>
				Type <span className='confirm-by-typing__confirm-text'>{confirmText}</span> to confirm:
			</p>
			<SlInput
				ref={inputRef}
				className='confirm-by-typing__input'
				value={value}
				onSlInput={(e) => onInput((e.target as HTMLInputElement).value)}
			/>
		</div>
	);
};

export default ConfirmByTyping;
