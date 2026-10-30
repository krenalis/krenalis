import React, { useState, useEffect } from 'react';
import './ConnectorRadios.css';
import WaRadio from '@awesome.me/webawesome/dist/react/radio/index.js';
import WaRadioGroup from '@awesome.me/webawesome/dist/react/radio-group/index.js';
import { FieldOption } from '../../../../lib/api/types/ui';

interface ConnectorRadiosProps {
	name: string;
	label: string;
	options: FieldOption[];
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorRadios = ({ name, label, options, error, val, onChange }: ConnectorRadiosProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onRadioGroupChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-radios'>
			<WaRadioGroup value={value} label={label} name={name} onChange={onRadioGroupChange}>
				{options.map((opt, _) => {
					return (
						<WaRadio key={opt.value} value={opt.value}>
							{opt.text}
						</WaRadio>
					);
				})}
			</WaRadioGroup>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorRadios;
