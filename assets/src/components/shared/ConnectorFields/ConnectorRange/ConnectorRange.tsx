import React, { useState, useEffect } from 'react';
import SlRange from '@shoelace-style/shoelace/dist/react/range/index.js';

interface ConnectorRangeProps {
	name: string;
	label: string;
	helpText: string;
	min: number;
	max: number;
	step: number;
	error: string;
	val: any;
	onChange: (...args: any) => void;
}

const ConnectorRange = ({ name, label, helpText, min, max, step, error, val, onChange }: ConnectorRangeProps) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onRangeChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connector-range'>
			<SlRange
				name={name}
				value={value}
				label={label}
				help-text={helpText}
				min={min}
				max={max}
				step={step}
				onSlChange={onRangeChange}
			/>
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

export default ConnectorRange;
