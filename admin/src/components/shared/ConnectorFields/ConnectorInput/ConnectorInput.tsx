import React, { useState, useEffect } from 'react';
import './ConnectorInput.css';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { InputType } from '../../../../types/external/ui';

interface ConnectorInputProps {
	name: string;
	label: string;
	placeholder: string;
	helpText: string;
	type: InputType;
	minlength: number;
	maxlength: number;
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
	error,
	val,
	onChange,
}: ConnectorInputProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onInputChange = (e) => {
		let v = e.currentTarget.value;
		if (type === 'number') {
			v = Number(v);
		}
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className={`connectorInput`}>
			<SlInput
				name={name}
				value={value}
				label={label}
				placeholder={placeholder}
				help-text={helpText}
				type={type === '' ? 'text' : type}
				minlength={minlength !== 0 ? minlength : undefined}
				maxlength={maxlength !== 0 ? maxlength : undefined}
				passwordToggle={type === 'password'}
				onSlChange={onInputChange}
			/>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorInput;
