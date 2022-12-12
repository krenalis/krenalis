import { useState, useEffect } from 'react';
import { SlRange } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorRange = ({ name, label, helpText, min, max, step, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onRangeChange = (e) => {
		let v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorRange'>
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
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorRange;
