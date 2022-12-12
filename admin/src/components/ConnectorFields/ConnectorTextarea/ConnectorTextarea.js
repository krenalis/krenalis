import { useState, useEffect } from 'react';
import './ConnectorTextarea';
import { SlTextarea } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorTextarea = ({
	name,
	label,
	placeholder,
	helpText,
	rows,
	minlength,
	maxlength,
	error,
	val,
	onChange,
}) => {
	const [value, setValue] = useState(val);

	useEffect(() => {
		setValue(val);
	}, [val]);

	const onTextAreaChange = (e) => {
		let v = e.currentTarget.value;
		setValue(v);
		onChange(name, v, e);
	};

	return (
		<div className='ConnectorTextarea'>
			<SlTextarea
				name={name}
				value={value}
				label={label}
				placeholder={placeholder}
				help-text={helpText}
				rows={rows}
				minlength={minlength !== 0 && minlength}
				maxlength={maxlength !== 0 && maxlength}
				onSlChange={onTextAreaChange}
			/>
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorTextarea;
