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
import ConnectorFieldInterface from '../../../lib/api/types/ui';

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
	switch (f.componentType) {
		case 'Input':
			if (f.rows === 0 || f.rows === 1) {
				component = (
					<ConnectorInput
						name={f.name}
						label={f.label}
						placeholder={f.placeholder}
						helpText={f.helpText}
						type={f.type === '' ? 'text' : f.type}
						minlength={f.minLength}
						maxlength={f.maxLength}
						onlyIntegerPart={f.onlyIntegerPart}
						error={f.error}
						val={value}
						onChange={onChange}
					/>
				);
			} else {
				component = (
					<ConnectorTextarea
						name={f.name}
						label={f.label}
						placeholder={f.placeholder}
						helpText={f.helpText}
						rows={f.rows}
						minlength={f.minLength}
						maxlength={f.maxLength}
						error={f.error}
						val={value}
						onChange={onChange}
					/>
				);
			}
			break;
		case 'Select':
			component = (
				<ConnectorSelect
					name={f.name}
					label={f.label}
					placeholder={f.placeholder}
					helpText={f.helpText}
					options={f.options}
					error={f.error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Switch':
			component = (
				<ConnectorSwitch name={f.name} label={f.label} error={f.error} val={value} onChange={onChange} />
			);
			break;
		case 'Checkbox':
			component = (
				<ConnectorCheckbox name={f.name} label={f.label} error={f.error} val={value} onChange={onChange} />
			);
			break;
		case 'ColorPicker':
			component = (
				<ConnectorColorPicker name={f.name} label={f.label} error={f.error} val={value} onChange={onChange} />
			);
			break;
		case 'Radios':
			component = (
				<ConnectorRadios
					name={f.name}
					label={f.label}
					options={f.options}
					error={f.error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Range':
			component = (
				<ConnectorRange
					name={f.name}
					label={f.label}
					helpText={f.helpText}
					min={f.min}
					max={f.max}
					step={f.step}
					error={f.error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'KeyValue':
			component = (
				<ConnectorKeyValue
					name={f.name}
					label={f.label}
					keyComponent={f.keyComponent}
					keyLabel={f.keyLabel}
					valueComponent={f.valueComponent}
					valueLabel={f.valueLabel}
					error={f.error}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'AlternativeFieldSets':
			component = (
				<ConnectorAlternativeFieldSets
					label={f.label}
					helpText={f.helpText}
					sets={f.sets}
					val={value}
					onChange={onChange}
				/>
			);
			break;
		case 'Text':
			component = <ConnectorText label={f.label} text={f.text} />;
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
	if (f.componentType === 'Text') return [null, null];

	let value: any, onChange: (...args: any) => void;
	if (fieldSetContext != null) {
		if (f.componentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.componentType} inside an AlternativeFieldSets component`);
		}
		const ctx = fieldSetContext;
		if (ctx.settings == null || ctx.settings[f.name] == null) {
			value = '';
		} else {
			value = ctx.settings[f.name];
		}
		onChange = ctx.onChange;
		return [value, onChange];
	}

	if (keyContext != null) {
		if (f.componentType === 'KeyValue' || f.componentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.componentType} inside the key cell of a KeyValue component`);
		}
		({ value, onChange } = keyContext);
		return [value, onChange];
	}

	if (valueContext != null) {
		if (f.componentType === 'KeyValue' || f.componentType === 'AlternativeFieldSets') {
			throw new Error(`[error] cannot render ${f.componentType} inside the value cell of a KeyValue component`);
		}
		({ value, onChange } = valueContext);
		return [value, onChange];
	}

	if (connectorUIContext != null) {
		const ctx = connectorUIContext;
		if (ctx.settings == null) {
			value = '';
		} else if (f.componentType === 'AlternativeFieldSets') {
			value = ctx.settings;
		} else if (ctx.settings[f.name] == null) {
			value = '';
		} else {
			value = ctx.settings[f.name];
		}
		onChange = ctx.onChange;
		return [value, onChange];
	}

	throw new Error('[error] cannot render ConnectorField component without a proper context');
};

export default ConnectorField;
