import { useState, useEffect } from 'react';
import './ConnectorSelect.css';
import { SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorSelect = ({ name, label, placeholder, helpText, options, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onSelectChange = (e) => {
		const v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='connectorSelect'>
			<SlSelect
				label={label}
				value={value}
				placeholder={placeholder}
				help-text={helpText}
				name={name}
				onSlChange={onSelectChange}
			>
				{options.map((opt) => {
					return <SlOption value={opt.Value}>{opt.Text}</SlOption>;
				})}
			</SlSelect>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorSelect;
