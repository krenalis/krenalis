import React, { useState, useEffect } from 'react';
import './ConnectorSelect.css';
import WaSelect from '@awesome.me/webawesome/dist/react/select/index.js';
import WaOption from '@awesome.me/webawesome/dist/react/option/index.js';
import { FieldOption } from '../../../../lib/api/types/ui';

interface ConnectorSelectProps {
	name: string;
	label: string;
	placeholder: string;
	helpText: string;
	options: FieldOption[];
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorSelect = ({
	name,
	label,
	placeholder,
	helpText,
	options,
	error,
	val,
	onChange,
}: ConnectorSelectProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onSelectChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-select'>
			<WaSelect
				label={label}
				value={value}
				placeholder={placeholder}
				hint={helpText}
				name={name}
				onChange={onSelectChange}
			>
				{options.map((opt) => {
					return (
						<WaOption key={opt.value} value={opt.value}>
							{opt.text}
						</WaOption>
					);
				})}
			</WaSelect>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorSelect;
