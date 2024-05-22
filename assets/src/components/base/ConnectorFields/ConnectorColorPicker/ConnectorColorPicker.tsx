import React, { useState, useEffect } from 'react';
import './ConnectorColorPicker.css';
import SlColorPicker from '@shoelace-style/shoelace/dist/react/color-picker/index.js';

interface ConnectorColorPickerProps {
	name: string;
	label: string;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorColorPicker = ({ name, label, error, val, onChange }: ConnectorColorPickerProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onColorPickerChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-color-picker'>
			<SlColorPicker value={value} name={name} label={label} onSlChange={onColorPickerChange} />
			<div className='connector-color-picker__label'>{label}</div>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorColorPicker;
