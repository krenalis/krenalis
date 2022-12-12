import { useState, useEffect } from 'react';
import './ConnectorRadios.css';
import { SlRadio, SlRadioGroup } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorRadios = ({ name, label, options, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onRadioGroupChange = (e) => {
		let v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorRadios'>
			<SlRadioGroup value={value} label={label} name={name} onSlChange={onRadioGroupChange} fieldset>
				{options.map((opt, i) => {
					return <SlRadio value={opt.Value}>{opt.Text}</SlRadio>;
				})}
			</SlRadioGroup>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorRadios;
