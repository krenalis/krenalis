import { useState, useEffect } from 'react';
import { FieldSetContext } from '../../../context/FieldSetContext';
import ConnectorField from '../ConnectorField';
import './ConnectorFieldSet.css';

const ConnectorFieldSet = ({ name, fields, val, onChange }) => {
	let [values, setValues] = useState(val);

	useEffect(() => {
		setValues(val);
	}, [val]);

	const onFieldChange = (fieldName, value) => {
		setValues((prevValues) => {
			let vals;
			if (prevValues == null) {
				vals = { [fieldName]: value };
			} else {
				vals = { ...prevValues, [fieldName]: value };
			}
			onChange(name, vals);
			return vals;
		});
	};

	let fieldSet = [];
	for (const [i, f] of fields.entries()) fieldSet.push(<ConnectorField key={i} field={f} />);

	return (
		<div class='ConnectorFieldSet'>
			<FieldSetContext.Provider value={{ values: values, onChange: onFieldChange }}>
				{fieldSet}
			</FieldSetContext.Provider>
		</div>
	);
};

export default ConnectorFieldSet;
