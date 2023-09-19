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
	Text: string;
	Value: any;
}

interface InputField {
	ComponentType: 'Input';
	Name: string;
	Type: InputType;
	Label: string;
	Placeholder: string;
	HelpText: string;
	Rows: number;
	MinLength: number;
	MaxLength: number;
	Error: string;
}

interface SelectField {
	ComponentType: 'Select';
	Name: string;
	Label: string;
	Placeholder: string;
	HelpText: string;
	Options: FieldOption[];
	Error: string;
}

interface CheckboxField {
	ComponentType: 'Checkbox';
	Name: string;
	Label: string;
	Error: string;
}

interface ColorPickerField {
	ComponentType: 'ColorPicker';
	Name: string;
	Label: string;
	Error: string;
}

interface RadiosField {
	ComponentType: 'Radios';
	Name: string;
	Label: string;
	Options: FieldOption[];
	Error: string;
}

interface RangeField {
	ComponentType: 'Range';
	Name: string;
	Label: string;
	HelpText: string;
	Min: number;
	Max: number;
	Step: number;
	Error: string;
}

interface SwitchField {
	ComponentType: 'Switch';
	Name: string;
	Label: string;
	Error: string;
}

interface KeyValueField {
	ComponentType: 'KeyValue';
	Name: string;
	Label: string;
	KeyLabel: string;
	KeyComponent: ConnectorField;
	ValueLabel: string;
	ValueComponent: ConnectorField;
	Error: string;
}

interface FieldSetField {
	ComponentType: 'FieldSet';
	Name: string;
	Label: string;
	Fields: ConnectorField[];
}

interface AlternativeFieldSetsField {
	ComponentType: 'AlternativeFieldSets';
	Label: string;
	HelpText: string;
	Sets: FieldSetField[];
}

interface TextField {
	ComponentType: 'Text';
	Label: string;
	Text: string;
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

interface ConnectorAction {
	Event: string;
	Text: string;
	Variant: Variant;
	Confirm: boolean;
	Role: ConnectionRole;
}

interface ConnectorAlert {
	Message: string;
	Variant: Variant;
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
	ConnectorAction,
	ConnectorAlert,
};
