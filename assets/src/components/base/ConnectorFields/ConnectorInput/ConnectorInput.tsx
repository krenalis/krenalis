import React, { useState } from 'react';
import './ConnectorInput.css';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { InputType } from '../../../../lib/api/types/ui';

interface ConnectorInputProps {
	name: string;
	label: string;
	placeholder: string;
	helpText: string;
	type: InputType;
	minlength: number;
	maxlength: number;
	onlyIntegerPart: boolean;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorInput = ({
	name,
	label,
	placeholder,
	helpText,
	type,
	minlength,
	maxlength,
	onlyIntegerPart,
	error,
	val,
	onChange,
}: ConnectorInputProps) => {
	const [value, setValue] = useState(val);

	const onInput = (e) => {
		let v = e.currentTarget.value;
		if (type === 'number') {
			let toShow: string;
			let toSave: number;
			if (onlyIntegerPart) {
				let val = v.replace(/[^0-9]/g, ''); // Prevent input of text, "," and "."
				toShow = String(Number(val));
				toSave = Number(val);
			} else {
				let val = v.replace(/[^0-9.,]/g, ''); // Prevent input of text
				toShow = val;
				toSave = Number(val);
			}
			setValue(toShow);
			onChange(name, toSave, e);
		} else {
			setValue(v);
			onChange(name, v, e);
		}
	};

	return (
		<div className='connector-input'>
			<SlInput
				name={name}
				value={value}
				label={label}
				placeholder={placeholder}
				help-text={helpText}
				type={type === '' || type === 'number' ? 'text' : type} // Use the text input in case of numbers to handle the value without interferences from Shoelace or the browser.
				minlength={minlength !== 0 ? minlength : undefined}
				maxlength={maxlength !== 0 ? maxlength : undefined}
				passwordToggle={type === 'password'}
				onInput={onInput}
			/>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorInput;
