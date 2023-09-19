import React, { useState, useEffect } from 'react';
import './ConnectorRadios.css';
import SlRadio from '@shoelace-style/shoelace/dist/react/radio/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import { FieldOption } from '../../../../types/external/ui';

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
		<div className='connectorRadios'>
			<SlRadioGroup value={value} label={label} name={name} onSlChange={onRadioGroupChange}>
				{options.map((opt, i) => {
					return <SlRadio value={opt.Value}>{opt.Text}</SlRadio>;
				})}
			</SlRadioGroup>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorRadios;
