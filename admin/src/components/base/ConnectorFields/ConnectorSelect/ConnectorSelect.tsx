import React, { useState, useEffect } from 'react';
import './ConnectorSelect.css';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
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
			<SlSelect
				label={label}
				value={value}
				placeholder={placeholder}
				help-text={helpText}
				name={name}
				onSlChange={onSelectChange}
			>
				{options.map((opt) => {
					return (
						<SlOption key={opt.value} value={opt.value}>
							{opt.text}
						</SlOption>
					);
				})}
			</SlSelect>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorSelect;
