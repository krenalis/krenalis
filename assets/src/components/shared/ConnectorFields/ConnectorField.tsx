import React, { ReactNode, useContext } from 'react';
import { ConnectorUIContext, ConnectorUIContextType } from '../../../context/ConnectorUIContext';
import { KeyContext, KeyContextType } from '../../../context/KeyContext';
import { ValueContext, ValueContextType } from '../../../context/ValueContext';
import { FieldSetContext, FieldSetContextType } from '../../../context/FieldSetContext';
import ConnectorInput from './ConnectorInput/ConnectorInput';
import ConnectorSelect from './ConnectorSelect/ConnectorSelect';
import ConnectorTextarea from './ConnectorTextarea/ConnectorTextarea';
import ConnectorKeyValue from './ConnectorKeyvalue/ConnectorKeyvalue';
import ConnectorCheckbox from './ConnectorCheckbox/ConnectorCheckbox';
import ConnectorColorPicker from './ConnectorColorPicker/ConnectorColorPicker';
import ConnectorRadios from './ConnectorRadios/ConnectorRadios';
import ConnectorRange from './ConnectorRange/ConnectorRange';
import ConnectorSwitch from './ConnectorSwitch/ConnectorSwitch';
import ConnectorText from './ConnectorText/ConnectorText';
import ConnectorAlternativeFieldSets from './ConnectorAlternativeFieldSets/ConnectorAlternativeFieldSets';
import ConnectorFieldInterface from '../../../types/external/ui';

interface ConnectorFieldProps {
	field: ConnectorFieldInterface;
}

// ConnectorField renders the right connector component, forwarding to it the
// proper value and 'onChange' callback extracted from the context.
const ConnectorField = ({ field: f }: ConnectorFieldProps) => {
	const connectorUIContext = useContext(ConnectorUIContext);
	const valueContext = useContext(ValueContext);
	const keyContext = useContext(KeyContext);
	const fieldSetContext = useContext(FieldSetContext);

	let value: any, onChange: (...args: any) => void;
	try {
		[value, onChange] = getContextValueAndCallback(
			f,
			connectorUIContext,
			valueContext,
			keyContext,
			fieldSetContext,
		);
	} catch (err) {
		console.error(err.message);
		return null;
	}

	let component: ReactNode;
	switch (f.ComponentType) {
		case 'Input':
			if (f.Rows === 0 || f.Rows === 1) {
				component = (
					<ConnectorInput
						name={f.Name}
						label={f.Label}
						placeholder={f.Placeholder}
						helpText={f.HelpText}
						type={f.Type === '' ? 'text' : f.Type}
						minlength={f.MinLength}
						maxlength={f.MaxLength}
						error={f.Error}
						val={value}
						onChange={onChange}
					/>
				);
			} else {
				component = (
					<ConnectorTextarea
						name={f.Name}
						label={f.Label}
						placeholder={f.Placeholder}
						helpText={f.HelpText}
						rows={f.Rows}
						minlength={f.MinLength}
						maxlength={f.MaxLength}
						error={f.Error}
						val={value}
						onChange={onChange}
					/>
				);
			}
			break;
		case 'Select':
			component = (
				<ConnectorSelect
					name={f.Name}
					label={f.Label}
					placeholder={f.Placeholder}
					helpText={f.HelpText}
					options={f.Options}
					error={f.Error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Switch':
			component = (
				<ConnectorSwitch name={f.Name} label={f.Label} error={f.Error} val={value} onChange={onChange} />
			);
			break;
		case 'Checkbox':
			component = (
				<ConnectorCheckbox name={f.Name} label={f.Label} error={f.Error} val={value} onChange={onChange} />
			);
			break;
		case 'ColorPicker':
			component = (
				<ConnectorColorPicker name={f.Name} label={f.Label} error={f.Error} val={value} onChange={onChange} />
			);
			break;
		case 'Radios':
			component = (
				<ConnectorRadios
					name={f.Name}
					label={f.Label}
					options={f.Options}
					error={f.Error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Range':
			component = (
				<ConnectorRange
					name={f.Name}
					label={f.Label}
					helpText={f.HelpText}
					min={f.Min}
					max={f.Max}
					step={f.Step}
					error={f.Error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'KeyValue':
			component = (
				<ConnectorKeyValue
					name={f.Name}
					label={f.Label}
					keyComponent={f.KeyComponent}
					keyLabel={f.KeyLabel}
					valueComponent={f.ValueComponent}
					valueLabel={f.ValueLabel}
					error={f.Error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'AlternativeFieldSets':
			component = (
				<ConnectorAlternativeFieldSets
					label={f.Label}
					helpText={f.HelpText}
					sets={f.Sets}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Text':
			component = <ConnectorText label={f.Label} text={f.Text} />;
			break;
		default:
			component = null;
	}

	return component;
};

// getContextValueAndCallback extracts the value of the component and the
// onChange callback from the first parent context.
//
// It throws an exception if the component is not placed inside a context or
// when the component cannot be used inside the first parent context.
const getContextValueAndCallback = (
	f: ConnectorFieldInterface,
	connectorUIContext: ConnectorUIContextType | undefined,
	keyContext: KeyContextType | undefined,
	valueContext: ValueContextType | undefined,
	fieldSetContext: FieldSetContextType | undefined,
) => {
	if (f.ComponentType === 'Text') return [null, null];

	let value: any, onChange: (...args: any) => void;
	if (fieldSetContext != null) {
		if (f.ComponentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.ComponentType} inside an AlternativeFieldSets component`);
		}
		const ctx = fieldSetContext;
		if (ctx.values == null || ctx.values[f.Name] == null) {
			value = '';
		} else {
			value = ctx.values[f.Name];
		}
		onChange = ctx.onChange;
		return [value, onChange];
	}

	if (keyContext != null) {
		if (f.ComponentType === 'KeyValue' || f.ComponentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.ComponentType} inside the key cell of a KeyValue component`);
		}
		({ value, onChange } = keyContext);
		return [value, onChange];
	}

	if (valueContext != null) {
		if (f.ComponentType === 'KeyValue' || f.ComponentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.ComponentType} inside the value cell of a KeyValue component`);
		}
		({ value, onChange } = valueContext);
		return [value, onChange];
	}

	if (connectorUIContext != null) {
		const ctx = connectorUIContext;
		if (ctx.values == null) {
			value = '';
		} else if (f.ComponentType === 'AlternativeFieldSets') {
			value = ctx.values;
		} else if (ctx.values[f.Name] == null) {
			value = '';
		} else {
			value = ctx.values[f.Name];
		}
		onChange = ctx.onChange;
		return [value, onChange];
	}

	throw new Error('[error] cannot render ConnectorField component without a proper context');
};

export default ConnectorField;
