import React from 'react';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import './ConfirmByTyping.css';

interface ConfirmByTypingProps {
	confirmText: string;
	value: string;
	onInput: (value: string) => void;
}

// TODO: focus direttamente nella casella

const ConfirmByTyping = ({ confirmText, value, onInput }: ConfirmByTypingProps) => {
	return (
		<div className='confirm-by-typing'>
			<p className='confirm-by-typing__instruction'>
				Type <span className='confirm-by-typing__confirm-text'>{confirmText}</span> to confirm:
			</p>
			<SlInput
				className='confirm-by-typing__input'
				// placeholder={confirmText}
				value={value}
				onSlInput={(e) => onInput((e.target as HTMLInputElement).value)}
			/>
		</div>
	);
};

export default ConfirmByTyping;
