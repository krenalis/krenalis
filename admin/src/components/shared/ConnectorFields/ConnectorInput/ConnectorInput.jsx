import { useState, useEffect } from 'react';
import './ConnectorInput.css';
import { SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorInput = ({ name, label, placeholder, helpText, type, minlength, maxlength, error, val, onChange }) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onInputChange = (e) => {
		const v = e.currentTarget.value;
		setValue(type === 'number' ? Number(v) : v);
		onChange(name, v, e);
	};

	return (
		<div className='connectorInput'>
			<SlInput
				name={name}
				value={value}
				label={label}
				placeholder={placeholder}
				help-text={helpText}
				type={type === '' ? 'text' : type}
				minlength={minlength !== 0 && minlength}
				maxlength={maxlength !== 0 && maxlength}
				passwordToggle={type === 'password'}
				onSlChange={onInputChange}
			/>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorInput;
