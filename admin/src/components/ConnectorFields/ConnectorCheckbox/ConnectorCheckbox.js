import { useState, useEffect } from 'react';
import './ConnectorCheckbox.css';
import { SlCheckbox } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorCheckbox = ({ name, label, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onCheckboxChange = (e) => {
		let v = !value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorCheckbox'>
			<SlCheckbox name={name} onSlChange={onCheckboxChange} checked={value}>
				{label}
			</SlCheckbox>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorCheckbox;
