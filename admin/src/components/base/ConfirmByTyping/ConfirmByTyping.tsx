import React, { useRef, useEffect } from 'react';
import WaInput from '@awesome.me/webawesome/dist/react/input/index.js';
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

		const dialog = input.closest('wa-dialog');
		if (dialog) {
			const onAfterShow = () => input.focus();
			dialog.addEventListener('wa-after-show', onAfterShow);
			return () => dialog.removeEventListener('wa-after-show', onAfterShow);
		} else {
			setTimeout(() => input.focus(), 50);
		}
	}, []);

	return (
		<div className='confirm-by-typing'>
			<p className='confirm-by-typing__instruction'>
				Type <span className='confirm-by-typing__confirm-text'>{confirmText}</span> to confirm:
			</p>
			<WaInput
				ref={inputRef}
				className='confirm-by-typing__input'
				value={value}
				onInput={(e) => onInput((e.target as HTMLInputElement).value)}
			/>
		</div>
	);
};

export default ConfirmByTyping;
