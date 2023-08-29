import React, { useState, useEffect } from 'react';
import './ConnectorCheckbox.css';
import { SlCheckbox } from '@shoelace-style/shoelace/dist/react/index.js';

interface ConnectorCheckboxProps {
	name: string;
	label: string;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorCheckbox = ({ name, label, error, val, onChange }: ConnectorCheckboxProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onCheckboxChange = (e) => {
		const v = !value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connectorCheckbox'>
			<SlCheckbox name={name} onSlChange={onCheckboxChange} checked={value}>
				{label}
			</SlCheckbox>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorCheckbox;
