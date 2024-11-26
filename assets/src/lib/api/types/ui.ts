import { ConnectionRole } from './connection';

type Variant = 'neutral' | 'primary' | 'success' | 'warning' | 'danger';

type InputType =
	| 'number'
	| 'search'
	| 'time'
	| 'text'
	| 'date'
	| 'datetime-local'
	| 'email'
	| 'password'
	| 'tel'
	| 'url'
	| '';

interface FieldOption {
	text: string;
	value: any;
}

interface InputField {
	componentType: 'Input';
	name: string;
	type: InputType;
	onlyIntegerPart: boolean;
	label: string;
	placeholder: string;
	helpText: string;
	rows: number;
	minLength: number;
	maxLength: number;
	error: string;
}

interface SelectField {
	componentType: 'Select';
	name: string;
	label: string;
	placeholder: string;
	helpText: string;
	options: FieldOption[];
	error: string;
}

interface CheckboxField {
	componentType: 'Checkbox';
	name: string;
	label: string;
	error: string;
}

interface ColorPickerField {
	componentType: 'ColorPicker';
	name: string;
	label: string;
	error: string;
}

interface RadiosField {
	componentType: 'Radios';
	name: string;
	label: string;
	options: FieldOption[];
	error: string;
}

interface RangeField {
	componentType: 'Range';
	name: string;
	label: string;
	helpText: string;
	min: number;
	max: number;
	step: number;
	error: string;
}

interface SwitchField {
	componentType: 'Switch';
	name: string;
	label: string;
	error: string;
}

interface KeyValueField {
	componentType: 'KeyValue';
	name: string;
	label: string;
	keyLabel: string;
	keyComponent: ConnectorField;
	valueLabel: string;
	valueComponent: ConnectorField;
	error: string;
}

interface FieldSetField {
	componentType: 'FieldSet';
	name: string;
	label: string;
	fields: ConnectorField[];
}

interface AlternativeFieldSetsField {
	componentType: 'AlternativeFieldSets';
	label: string;
	helpText: string;
	sets: FieldSetField[];
}

interface TextField {
	componentType: 'Text';
	label: string;
	text: string;
}

type ConnectorField =
	| InputField
	| SelectField
	| CheckboxField
	| ColorPickerField
	| RadiosField
	| RangeField
	| SwitchField
	| KeyValueField
	| FieldSetField
	| AlternativeFieldSetsField
	| TextField;

interface ConnectorButton {
	event: string;
	text: string;
	variant: Variant;
	confirm: boolean;
	role: ConnectionRole;
}

interface ConnectorAlert {
	message: string;
	variant: Variant;
}

export default ConnectorField;
export type {
	FieldOption,
	InputType,
	InputField,
	SelectField,
	CheckboxField,
	ColorPickerField,
	RadiosField,
	RangeField,
	SwitchField,
	KeyValueField,
	FieldSetField,
	AlternativeFieldSetsField,
	TextField,
	ConnectorButton,
	ConnectorAlert,
};
