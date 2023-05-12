import { useState } from 'react';
import './ClipboardInput.css';
import { SlInput, SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ClipboardInput = ({ value }) => {
	let [hasCopied, setHasCopied] = useState(false);
	let [hasError, setHasError] = useState(false);

	let onButtonClick = async () => {
		if (hasError) return;
		let err = await navigator.clipboard.writeText(String(value));
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
