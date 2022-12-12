import { useState, useEffect } from 'react';
import './ConnectorSwitch.css';
import { SlSwitch } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorSwitch = ({ name, label, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onSwitchChange = (e) => {
		let v = !value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorSwitch'>
			<SlSwitch name={name} onSlChange={onSwitchChange} checked={value}>
				{label}
			</SlSwitch>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorSwitch;
