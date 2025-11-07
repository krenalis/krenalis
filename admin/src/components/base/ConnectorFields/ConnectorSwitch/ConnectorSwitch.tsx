import React, { useState, useEffect } from 'react';
import './ConnectorSwitch.css';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';

interface ConnectorSwitchProps {
	name: string;
	label: string;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorSwitch = ({ name, label, error, val, onChange }: ConnectorSwitchProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onSwitchChange = (e) => {
		const v = !value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-switch'>
			<SlSwitch name={name} onSlChange={onSwitchChange} checked={value}>
				{label}
			</SlSwitch>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorSwitch;
