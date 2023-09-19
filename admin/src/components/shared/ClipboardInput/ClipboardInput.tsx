import React, { useState } from 'react';
import './ClipboardInput.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface ClipboardInputProps {
	value: string;
}

const ClipboardInput = ({ value }: ClipboardInputProps) => {
	const [hasCopied, setHasCopied] = useState<boolean>(false);
	const [hasError, setHasError] = useState<boolean>(false);

	const onButtonClick = async () => {
		if (hasError) return;
		const err = await navigator.clipboard.writeText(String(value));
		if (err != null) {
			console.error(`error while copying to clipboard: ${err}`);
			setHasError(true);
			return;
		}
		setHasCopied(true);
		setTimeout(() => {
			setHasCopied(false);
		}, 3000);
	};

	return (
		<div className='clipboardInput'>
			<SlInput value={value} readonly />
			<SlButton
				disabled={hasError}
				variant={hasCopied ? 'success' : hasError ? 'danger' : 'neutral'}
				onClick={onButtonClick}
			>
				<SlIcon slot='suffix' name={hasCopied ? 'clipboard-check' : hasError ? 'clipboard-x' : 'clipboard'} />
				{hasCopied ? 'Copied' : hasError ? "Can't copy" : 'Click to copy'}
			</SlButton>
		</div>
	);
};

export default ClipboardInput;
