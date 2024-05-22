import React, { useState, useEffect } from 'react';
import './ConnectorTextarea';
import SlTextarea from '@shoelace-style/shoelace/dist/react/textarea/index.js';

interface ConnectorTextAreaProps {
	name: string;
	label: string;
	placeholder: string;
	helpText: string;
	rows: number;
	minlength: number;
	maxlength: number;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorTextarea = ({
	name,
	label,
	placeholder,
	helpText,
	rows,
	minlength,
	maxlength,
	error,
	val,
	onChange,
}: ConnectorTextAreaProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onTextAreaChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-textarea'>
			<SlTextarea
				name={name}
				value={value}
				label={label}
				placeholder={placeholder}
				help-text={helpText}
				rows={rows}
				minlength={minlength !== 0 ? minlength : undefined}
				maxlength={maxlength !== 0 ? maxlength : undefined}
				onSlChange={onTextAreaChange}
			/>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorTextarea;
