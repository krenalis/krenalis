import { useState, useEffect } from 'react';
import './ConnectorSelect.css';
import { SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorSelect = ({ name, label, placeholder, helpText, options, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onSelectChange = (e) => {
		let v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorSelect'>
			<SlSelect
				label={label}
				value={value}
				placeholder={placeholder}
				help-text={helpText}
				name={name}
				onSlChange={onSelectChange}
			>
				{options.map((opt) => {
					return <SlMenuItem value={opt.Value}>{opt.Text}</SlMenuItem>;
				})}
			</SlSelect>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorSelect;
