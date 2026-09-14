import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import './PropertyForm.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlRadioButton from '@shoelace-style/shoelace/dist/react/radio-button/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlTextarea from '@shoelace-style/shoelace/dist/react/textarea/index.js';
import type SlTextareaElement from '@shoelace-style/shoelace/dist/components/textarea/textarea.component.js';
import AppContext from '../../../context/AppContext';
import Type, {
	DecimalType,
	FloatBitSize,
	FloatType,
	IntBitSize,
	IntType,
	StringType,
} from '../../../lib/api/types/types';
import TransformedConnection from '../../../lib/core/connection';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { isSuitableAsIdentifier } from '../../helpers/types';
import {
	SchemaPropertyIdentifierLabel,
	SchemaPropertyIdentifierValue,
	SchemaPropertyInfoTooltip,
	SchemaPropertyPrimarySourceLabel,
} from '../Schema/SchemaPropertyGrid';
import { getParentPropertyKey } from './SchemaEdit.helpers';
import { PropertyFieldChanges, PropertyParent, PropertyToEdit } from './useSchemaEdit';
import {
	getPropertyValueType,
	PropertyTypeSelector,
	type PropertyTypeSelectorRef,
	replacePropertyValueType,
} from './PropertyTypeSelector';

const INT_BITSIZES: string[] = ['8', '16', '24', '32', '64'];
const FLOAT_BITSIZES: string[] = ['32', '64'];
const MAX_DECIMAL_PRECISION: number = 76;
const MAX_DECIMAL_SCALE: number = 37;
const MAX_STRING_LENGTH: number = 4294967295;
interface DecimalTypeInputs {
	precision: string;
	precisionBadInput: boolean;
	scale: string;
	scaleBadInput: boolean;
}

interface NumericRangeInput {
	value: string;
	badInput: boolean;
}

interface NumericRangeInputs {
	minimum: NumericRangeInput;
	maximum: NumericRangeInput;
}

type NumericType = IntType | FloatType | DecimalType;

interface PropertyTypeError {
	location: 'type' | 'string-constraints' | 'decimal-constraints' | 'numeric-range';
	message: string;
}

const disableShoelaceTextareaHeightReset = (textarea: SlTextareaElement | null) => {
	if (textarea == null) {
		return;
	}
	// Shoelace 2.20 resets the inline height whenever its ResizeObserver runs,
	// which prevents the browser's vertical resize handle from working.
	// See https://github.com/shoelace-style/shoelace/pull/2465.
	Reflect.set(textarea, 'setTextareaHeight', () => undefined);
};

interface PropertyFormProps {
	fieldChanges?: PropertyFieldChanges;
	formID: string;
	identifierPosition?: number;
	propertyToEdit: PropertyToEdit;
	primarySources: Record<string, string>;
	parents?: PropertyParent[];
	showParent?: boolean;
	onSave: (property: PropertyToEdit, primarySource: string | null) => void;
	onDirtyChange?: (dirty: boolean) => void;
	onValidityChange?: (valid: boolean) => void;
}

const PropertyForm = ({
	fieldChanges,
	formID,
	identifierPosition,
	propertyToEdit,
	primarySources,
	parents,
	showParent,
	onSave,
	onDirtyChange,
	onValidityChange,
}: PropertyFormProps) => {
	const [property, setProperty] = useState<PropertyToEdit>(() => structuredClone(propertyToEdit));
	const [primarySource, setPrimarySource] = useState<string | null>(primarySources[propertyToEdit.key] || null);
	const [decimalTypeInputs, setDecimalTypeInputs] = useState<DecimalTypeInputs>(() =>
		getDecimalTypeInputs(propertyToEdit.type),
	);
	const [numericRangeInputs, setNumericRangeInputs] = useState<NumericRangeInputs>(() =>
		getNumericRangeInputs(propertyToEdit.type),
	);
	const [isNameEditable, setIsNameEditable] = useState(propertyToEdit.key == null);
	const [nameError, setNameError] = useState('');
	const [typeError, setTypeError] = useState<PropertyTypeError | null>(null);
	const initialState = useRef('');
	const nameInputRef = useRef<any>();
	const typeSelectorRef = useRef<PropertyTypeSelectorRef>();

	const { connections } = useContext(AppContext);
	const isEditing = propertyToEdit.key != null;
	const canEditType = !isEditing || propertyToEdit.isEditable === true;
	const parentKey = property.key == null ? property.parentKey || '' : getParentPropertyKey(property.key);
	const showIdentityResolution =
		isEditing && propertyToEdit.isEditable !== true && isSuitableAsIdentifier(propertyToEdit.type);

	const sourceConnections = useMemo(() => {
		const sources: TransformedConnection[] = [];
		for (const connection of connections) {
			if (connection.role === 'Source' && connection.connector.asSource?.targets.includes('User')) {
				sources.push(connection);
			}
		}
		return sources;
	}, [connections]);

	useEffect(() => {
		const nextProperty = structuredClone(propertyToEdit);
		const nextPrimarySource = primarySources[propertyToEdit.key] || null;
		const nextDecimalTypeInputs = getDecimalTypeInputs(propertyToEdit.type);
		const nextNumericRangeInputs = getNumericRangeInputs(propertyToEdit.type);
		setProperty(nextProperty);
		setPrimarySource(nextPrimarySource);
		setDecimalTypeInputs(nextDecimalTypeInputs);
		setNumericRangeInputs(nextNumericRangeInputs);
		setIsNameEditable(propertyToEdit.key == null);
		setNameError('');
		setTypeError(null);
		initialState.current = propertyFormStateKey(
			nextProperty,
			nextPrimarySource,
			nextDecimalTypeInputs,
			nextNumericRangeInputs,
		);
		onDirtyChange?.(false);
		setTimeout(() => {
			const input = nameInputRef.current?.shadowRoot?.querySelector('input');
			if (input != null) {
				input.setAttribute('data-1p-ignore', '');
				input.setAttribute('data-bwignore', '');
				input.setAttribute('data-form-type', 'other');
				input.setAttribute('data-lpignore', 'true');
			}
			if (propertyToEdit.key == null) {
				nameInputRef.current?.focus();
			} else {
				input?.blur();
				if (input != null) {
					input.scrollLeft = 0;
				}
			}
		});
	}, [propertyToEdit]);

	useEffect(() => {
		if (initialState.current === '') {
			return;
		}
		onDirtyChange?.(
			propertyFormStateKey(property, primarySource, decimalTypeInputs, numericRangeInputs) !==
				initialState.current,
		);
	}, [property, primarySource, decimalTypeInputs, numericRangeInputs]);

	useEffect(() => {
		const type = getPropertyValueType(property.type);
		if (isNumericType(type)) {
			setTypeError(getNumericTypeError(type, decimalTypeInputs, numericRangeInputs));
		}
	}, [property.type, decimalTypeInputs, numericRangeInputs]);

	useEffect(() => {
		onValidityChange?.(nameError === '' && typeError == null);
	}, [nameError, typeError]);

	const updateProperty = (update: (property: PropertyToEdit) => void) => {
		setProperty((current) => {
			const next = structuredClone(current);
			update(next);
			return next;
		});
	};

	const updateValueType = (update: (type: any) => void) => {
		updateProperty((nextProperty) => {
			const valueType = structuredClone(getPropertyValueType(nextProperty.type));
			update(valueType);
			nextProperty.type = replacePropertyValueType(nextProperty.type, valueType);
		});
	};

	const onChangeName = () => {
		setIsNameEditable(true);
		setTimeout(() => {
			const input = nameInputRef.current?.shadowRoot?.querySelector('input');
			input?.focus();
			input?.setSelectionRange(0, 0);
			if (input != null) {
				input.scrollLeft = 0;
			}
		});
	};

	const onBlurName = () => {
		if (isEditing && property.name === propertyToEdit.name) {
			setIsNameEditable(false);
			const input = nameInputRef.current?.shadowRoot?.querySelector('input');
			if (input != null) {
				input.scrollLeft = 0;
			}
		}
	};

	const onFocusName = (event) => {
		if (!isNameEditable) {
			const input = event.currentTarget.shadowRoot?.querySelector('input');
			input?.blur();
			if (input != null) {
				input.scrollLeft = 0;
			}
		}
	};

	const onInputName = (event) => {
		const name = event.target.value;
		setNameError(getPropertyNameError(name, parentKey, isEditing ? propertyToEdit.name : undefined, parents));
		updateProperty((nextProperty) => {
			nextProperty.name = name;
		});
	};

	const onKeyDownName = (event: React.KeyboardEvent<any>) => {
		if (
			isEditing ||
			event.currentTarget.value === '' ||
			event.key !== 'Tab' ||
			event.shiftKey ||
			event.altKey ||
			event.ctrlKey ||
			event.metaKey
		) {
			return;
		}
		requestAnimationFrame(() => typeSelectorRef.current?.focusStructureTrigger());
	};

	const onKeyDownPrimarySource = (event: React.KeyboardEvent<HTMLDivElement>) => {
		if (event.key !== 'Tab' || event.shiftKey || event.altKey || event.ctrlKey || event.metaKey) {
			return;
		}
		const panel = event.currentTarget.closest('.property-panel') as HTMLElement | null;
		const cancelButton = panel?.querySelector('.property-panel__cancel') as HTMLElement | null;
		if (cancelButton == null) {
			return;
		}
		event.preventDefault();
		cancelButton.focus();
	};

	const onChangeParent = (event) => {
		const parentKey = event.target.value === '__root__' ? '' : event.target.value;
		const parent = parents.find((candidate) => candidate.key === parentKey);
		if (parent == null) {
			return;
		}
		if (property.name !== '') {
			setNameError(
				getPropertyNameError(property.name, parent.key, isEditing ? propertyToEdit.name : undefined, parents),
			);
		}
		updateProperty((nextProperty) => {
			nextProperty.parentKey = parent.key;
			nextProperty.indentation = parent.indentation;
			nextProperty.root = parent.root;
		});
	};

	const onChangeType = (type: Type | null) => {
		updateProperty((nextProperty) => {
			nextProperty.type = type;
		});
		setDecimalTypeInputs(getDecimalTypeInputs(type));
		setNumericRangeInputs(getNumericRangeInputs(type));
		if (type?.kind === 'object' || type?.kind === 'array') {
			setPrimarySource(null);
		}
		setTypeError(null);
	};

	const onChangeBitSize = (event) => {
		updateValueType((type: IntType | FloatType) => {
			type.bitSize = Number(event.target.value) as IntBitSize | FloatBitSize;
		});
	};

	const onInputPrecision = (event) => {
		const value = event.currentTarget.value;
		const badInput = event.currentTarget.validity.badInput;
		setDecimalTypeInputs((current) => ({
			...current,
			precision: value,
			precisionBadInput: badInput,
		}));
		if (value === '') {
			return;
		}
		updateValueType((type: DecimalType) => {
			type.precision = Number(value);
		});
	};

	const onInputScale = (event) => {
		const value = event.currentTarget.value;
		const badInput = event.currentTarget.validity.badInput;
		setDecimalTypeInputs((current) => ({
			...current,
			scale: value,
			scaleBadInput: badInput,
		}));
		if (value === '') {
			return;
		}
		updateValueType((type: DecimalType) => {
			type.scale = Number(value);
		});
	};

	const onInputNumericRange = (name: keyof NumericRangeInputs, event) => {
		const input = {
			value: event.currentTarget.value,
			badInput: event.currentTarget.validity.badInput,
		};
		setNumericRangeInputs((current) => ({ ...current, [name]: input }));
		if (input.badInput) {
			return;
		}
		updateValueType((type: NumericType) => {
			if (input.value === '') {
				delete type[name];
			} else {
				type[name] = Number(input.value);
			}
		});
	};

	const onRealChange = () => {
		updateValueType((type: FloatType) => {
			type.real = !type.real;
		});
	};

	const onUnsignedChange = (event) => {
		const unsigned = event.currentTarget.value === 'unsigned';
		const minimum = numericRangeInputs.minimum;
		const clearMinimum = unsigned && !minimum.badInput && minimum.value.startsWith('-');
		if (clearMinimum) {
			setNumericRangeInputs((current) => ({
				...current,
				minimum: { value: '', badInput: false },
			}));
		}
		updateValueType((type: IntType) => {
			type.unsigned = unsigned;
			if (clearMinimum) {
				delete type.minimum;
			}
		});
	};

	const onInputMaxBytes = (event) => {
		const value = event.currentTarget.value;
		updateValueType((type: StringType) => {
			if (value === '') {
				delete type.maxBytes;
			} else {
				type.maxBytes = Number(value);
			}
		});
		setTypeError(null);
	};

	const onInputMaxLength = (event) => {
		const value = event.currentTarget.value;
		updateValueType((type: StringType) => {
			if (value === '') {
				delete type.maxLength;
			} else {
				type.maxLength = Number(value);
			}
		});
		setTypeError(null);
	};

	const onInputDescription = (event) => {
		updateProperty((nextProperty) => {
			nextProperty.description = event.target.value;
		});
	};

	const onInputDisplayName = (event) => {
		updateProperty((nextProperty) => {
			nextProperty.displayName = event.target.value;
		});
	};

	const onChangePrimarySource = (event) => {
		setPrimarySource(event.target.value === 'none' ? null : event.target.value);
	};

	const onSubmit = (event: React.FormEvent) => {
		event.preventDefault();
		const propertyNameError = getPropertyNameError(
			property.name,
			parentKey,
			isEditing ? propertyToEdit.name : undefined,
			parents,
		);
		if (propertyNameError !== '') {
			setNameError(propertyNameError);
			return;
		}
		const error = validatePropertyType(property, decimalTypeInputs, numericRangeInputs);
		if (error != null) {
			setTypeError(error);
			return;
		}
		try {
			onSave(property, primarySource);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		initialState.current = propertyFormStateKey(property, primarySource, decimalTypeInputs, numericRangeInputs);
		onDirtyChange?.(false);
	};

	const valueType = getPropertyValueType(property.type);
	let decimalDescription: string | null = null;
	if (valueType?.kind === 'decimal' && checkDecimalType(valueType) == null) {
		const scale = valueType.scale ?? 0;
		const precisionDescription = `${valueType.precision} ${valueType.precision === 1 ? 'digit' : 'digits'} total`;
		decimalDescription =
			scale === 0
				? `${precisionDescription}, with no decimal places`
				: `${precisionDescription}, with ${scale} ${scale === 1 ? 'digit' : 'digits'} after the decimal point`;
	}
	const selectedConnection = sourceConnections.find((connection) => connection.id === primarySource);
	let minimumPlaceholder = '';
	let maximumPlaceholder = '';
	let minimumTooltip: { content: string; label: string } | undefined;
	let maximumTooltip: { content: string; label: string } | undefined;
	if (valueType?.kind === 'int') {
		const [minimum, maximum] = getIntegerTypeRange(valueType);
		if (valueType.unsigned || valueType.bitSize === 8 || valueType.bitSize === 16) {
			minimumPlaceholder = minimum.toString();
		}
		if (valueType.bitSize === 8 || valueType.bitSize === 16) {
			maximumPlaceholder = maximum.toString();
		}
		minimumTooltip = {
			content: `Minimum allowed: ${formatIntegerLimit(minimum)}. Leave blank to use this minimum value.`,
			label: 'About the minimum value',
		};
		maximumTooltip = {
			content: `Maximum allowed: ${formatIntegerLimit(maximum)}. Leave blank to use this maximum value.`,
			label: 'About the maximum value',
		};
	}
	let numericRangeStep: number | 'any' = 1;
	if (valueType?.kind === 'float') {
		numericRangeStep = 'any';
	} else if (valueType?.kind === 'decimal') {
		numericRangeStep = 10 ** -(valueType.scale ?? 0);
	}
	const numericRangeControls = isNumericType(valueType) ? (
		<div className='property-form__numeric-range'>
			<SlInput
				className='property-form__minimum'
				size='small'
				value={numericRangeInputs.minimum.value}
				type='number'
				step={numericRangeStep}
				placeholder={minimumPlaceholder}
				onSlInput={(event) => onInputNumericRange('minimum', event)}
			>
				<PropertyFormLabel slot='label' tooltip={minimumTooltip}>
					Min
				</PropertyFormLabel>
			</SlInput>
			<span className='property-form__numeric-range-separator' aria-hidden='true'>
				–
			</span>
			<SlInput
				className='property-form__maximum'
				size='small'
				value={numericRangeInputs.maximum.value}
				type='number'
				step={numericRangeStep}
				placeholder={maximumPlaceholder}
				onSlInput={(event) => onInputNumericRange('maximum', event)}
			>
				<PropertyFormLabel slot='label' tooltip={maximumTooltip}>
					Max
				</PropertyFormLabel>
			</SlInput>
			{typeError?.location === 'numeric-range' && (
				<PropertyFormError name='numeric-range'>{typeError.message}</PropertyFormError>
			)}
		</div>
	) : null;

	return (
		<form className='property-form' id={formID} onSubmit={onSubmit}>
			<div className='property-form__control property-form__control--name'>
				<SlInput
					className='property-form__name-input'
					ref={nameInputRef}
					value={property.name}
					autocomplete='off'
					name='name'
					placeholder='first_name'
					readonly={!isNameEditable}
					onSlBlur={onBlurName}
					onSlFocus={onFocusName}
					onSlInput={onInputName}
					onKeyDown={onKeyDownName}
				>
					<PropertyFormLabel slot='label' modified={fieldChanges?.name}>
						Name
					</PropertyFormLabel>
					{isEditing && !isNameEditable && (
						<SlButton
							className='property-form__change-name'
							size='small'
							slot='suffix'
							variant='text'
							onPointerDown={(event) => event.preventDefault()}
							onClick={onChangeName}
						>
							Change
						</SlButton>
					)}
				</SlInput>
				{nameError !== '' && <PropertyFormError name='name'>{nameError}</PropertyFormError>}
			</div>
			<div className='property-form__control'>
				<div className='property-form__label'>
					<PropertyFormLabel modified={fieldChanges?.type}>Type</PropertyFormLabel>
				</div>
				<PropertyTypeSelector
					ref={typeSelectorRef}
					type={property.type}
					canEditType={canEditType}
					onChange={onChangeType}
				/>
				{typeError?.location === 'type' && (
					<PropertyFormError name='type'>{typeError.message}</PropertyFormError>
				)}
			</div>
			{valueType?.kind === 'string' && canEditType && (
				<div className='property-form__constraints property-form__constraints--length'>
					<SlInput
						label='Max characters'
						size='small'
						value={valueType.maxLength == null ? '' : String(valueType.maxLength)}
						type='number'
						min={1}
						max={MAX_STRING_LENGTH}
						step={1}
						onSlInput={onInputMaxLength}
					/>
					<SlInput
						label='Max bytes'
						size='small'
						value={valueType.maxBytes == null ? '' : String(valueType.maxBytes)}
						type='number'
						min={1}
						max={MAX_STRING_LENGTH}
						step={1}
						onSlInput={onInputMaxBytes}
					/>
					{typeError?.location === 'string-constraints' && (
						<PropertyFormError name='string-constraints'>{typeError.message}</PropertyFormError>
					)}
				</div>
			)}
			{(valueType?.kind === 'int' || valueType?.kind === 'float') && canEditType && (
				<div
					className={`property-form__constraints property-form__constraints--${
						valueType.kind === 'int' ? 'integer' : 'float'
					}`}
				>
					{valueType.kind === 'int' && (
						<SlRadioGroup
							className='property-form__integer-sign'
							label='Sign'
							size='small'
							value={valueType.unsigned ? 'unsigned' : 'signed'}
							onSlChange={onUnsignedChange}
						>
							<SlRadioButton value='signed'>signed</SlRadioButton>
							<SlRadioButton value='unsigned'>unsigned</SlRadioButton>
						</SlRadioGroup>
					)}
					<SlSelect
						className='property-form__bit-size'
						label={valueType.kind === 'int' ? 'Integer size' : 'Bit size'}
						size='small'
						value={String(valueType.bitSize)}
						onSlChange={onChangeBitSize}
					>
						{(valueType.kind === 'int' ? INT_BITSIZES : FLOAT_BITSIZES).map((bitSize) => (
							<SlOption key={bitSize} value={bitSize}>
								{bitSize}-bit
							</SlOption>
						))}
					</SlSelect>
					{valueType.kind === 'float' && (
						<SlCheckbox size='small' checked={!valueType.real} onSlChange={onRealChange}>
							<span className='property-form__float-special-values-label'>Allow ±Inf and NaN</span>
						</SlCheckbox>
					)}
					{numericRangeControls}
				</div>
			)}
			{valueType?.kind === 'decimal' && canEditType && (
				<div className='property-form__constraints property-form__constraints--decimal'>
					<SlInput
						className='property-form__precision'
						label='Precision'
						size='small'
						value={decimalTypeInputs.precision}
						type='number'
						max={MAX_DECIMAL_PRECISION}
						maxlength={2}
						onSlInput={onInputPrecision}
					/>
					<SlInput
						className='property-form__scale'
						label='Scale'
						size='small'
						value={decimalTypeInputs.scale}
						type='number'
						max={MAX_DECIMAL_SCALE}
						maxlength={2}
						onSlInput={onInputScale}
					/>
					{typeError?.location === 'decimal-constraints' ? (
						<PropertyFormError name='decimal-constraints'>{typeError.message}</PropertyFormError>
					) : decimalDescription != null ? (
						<div className='property-form__decimal-description'>{decimalDescription}</div>
					) : null}
					{numericRangeControls}
				</div>
			)}
			{showParent && parents != null && (
				<div className='property-form__control'>
					<SlSelect
						className='property-form__parent'
						label='Add to'
						value={property.parentKey || '__root__'}
						onSlChange={onChangeParent}
					>
						{parents.map((parent) => (
							<SlOption key={parent.key || '__root__'} value={parent.key || '__root__'}>
								{parent.label}
							</SlOption>
						))}
					</SlSelect>
				</div>
			)}
			<SlInput
				className='property-form__control property-form__display-name'
				value={property.displayName || ''}
				name='displayName'
				placeholder='First name'
				onSlInput={onInputDisplayName}
			>
				<PropertyFormLabel slot='label' modified={fieldChanges?.displayName}>
					Display name <span className='property-form__optional-label'>(optional)</span>
				</PropertyFormLabel>
			</SlInput>
			<SlTextarea
				className='property-form__control property-form__description'
				ref={disableShoelaceTextareaHeightReset}
				value={property.description || ''}
				name='description'
				placeholder='Describe what this property represents…'
				onSlInput={onInputDescription}
			>
				<PropertyFormLabel slot='label' modified={fieldChanges?.description}>
					Description <span className='property-form__optional-label'>(optional)</span>
				</PropertyFormLabel>
			</SlTextarea>
			{showIdentityResolution && (
				<div className='property-form__control'>
					<div className='property-form__label'>
						<SchemaPropertyIdentifierLabel />
					</div>
					<div className='property-form__read-only-value'>
						<SchemaPropertyIdentifierValue position={identifierPosition} />
					</div>
				</div>
			)}
			{property.type?.kind !== 'object' && property.type?.kind !== 'array' && (
				<div className='property-form__control' onKeyDownCapture={onKeyDownPrimarySource}>
					{sourceConnections.length === 0 ? (
						<>
							<div className='property-form__label'>
								<PropertyFormLabel modified={fieldChanges?.primarySource}>
									<SchemaPropertyPrimarySourceLabel />
								</PropertyFormLabel>
							</div>
							<div className='property-form__empty-value'>No source connections are available.</div>
						</>
					) : (
						<SlSelect
							className='property-form__primary-source'
							value={primarySource == null ? 'none' : primarySource}
							name='primary-source'
							onSlChange={onChangePrimarySource}
						>
							<PropertyFormLabel slot='label' modified={fieldChanges?.primarySource}>
								<SchemaPropertyPrimarySourceLabel />
							</PropertyFormLabel>
							<div slot='prefix'>
								{selectedConnection != null && (
									<LittleLogo
										code={selectedConnection.connector.code}
										path={CONNECTORS_ASSETS_PATH}
									/>
								)}
							</div>
							<SlOption value='none'>No primary source</SlOption>
							{sourceConnections.map((connection) => (
								<SlOption key={connection.id} value={connection.id}>
									<div slot='prefix'>
										<LittleLogo code={connection.connector.code} path={CONNECTORS_ASSETS_PATH} />
									</div>
									{connection.name}
								</SlOption>
							))}
						</SlSelect>
					)}
				</div>
			)}
		</form>
	);
};

const PropertyFormLabel = ({
	children,
	modified,
	slot,
	tooltip,
}: {
	children: React.ReactNode;
	modified?: boolean;
	slot?: string;
	tooltip?: {
		content: string;
		label: string;
	};
}) => {
	return (
		<span className='property-form__label-content' slot={slot}>
			{children}
			{tooltip != null && <SchemaPropertyInfoTooltip content={tooltip.content} label={tooltip.label} />}
			{modified && <span className='property-form__modified-dot' role='img' aria-label='Modified' />}
		</span>
	);
};

const PropertyFormError = ({ name, children }: { name: string; children: React.ReactNode }) => {
	return (
		<div className='property-form__control-error' data-error-on={name}>
			<SlIcon name='exclamation-circle' />
			{children}
		</div>
	);
};

const propertyFormStateKey = (
	property: PropertyToEdit,
	primarySource: string | null,
	decimalTypeInputs: DecimalTypeInputs,
	numericRangeInputs: NumericRangeInputs,
): string => {
	return JSON.stringify({ property, primarySource, decimalTypeInputs, numericRangeInputs });
};

const validatePropertyType = (
	property: PropertyToEdit,
	decimalTypeInputs: DecimalTypeInputs,
	numericRangeInputs: NumericRangeInputs,
): PropertyTypeError | null => {
	if (property.type == null) {
		return { location: 'type', message: 'Type cannot be empty' };
	}
	const type = getPropertyValueType(property.type);
	if (type.kind === 'string') {
		if (
			type.maxLength != null &&
			(!Number.isInteger(type.maxLength) || type.maxLength < 1 || type.maxLength > MAX_STRING_LENGTH)
		) {
			return {
				location: 'string-constraints',
				message: `Max characters must be an integer in range [1, ${MAX_STRING_LENGTH}]`,
			};
		}
		if (
			type.maxBytes != null &&
			(!Number.isInteger(type.maxBytes) || type.maxBytes < 1 || type.maxBytes > MAX_STRING_LENGTH)
		) {
			return {
				location: 'string-constraints',
				message: `Max bytes must be an integer in range [1, ${MAX_STRING_LENGTH}]`,
			};
		}
	}
	if (isNumericType(type)) {
		return getNumericTypeError(type, decimalTypeInputs, numericRangeInputs);
	}
	return null;
};

const checkDecimalType = (type: DecimalType): string | undefined => {
	if (!Number.isInteger(type.precision) || type.precision < 1 || type.precision > MAX_DECIMAL_PRECISION) {
		return `Precision must be in range [1, ${MAX_DECIMAL_PRECISION}]`;
	}
	const scale = type.scale ?? 0;
	if (!Number.isInteger(scale) || scale < 0 || scale > MAX_DECIMAL_SCALE) {
		return `Scale must be in range [0, ${MAX_DECIMAL_SCALE}]`;
	}
	if (scale > type.precision) {
		return 'Scale cannot be greater than precision';
	}
};

const checkDecimalTypeInputs = (inputs: DecimalTypeInputs): string | undefined => {
	if (inputs.precisionBadInput) {
		return 'Precision must be a number';
	}
	if (inputs.scaleBadInput) {
		return 'Scale must be a number';
	}
	if (inputs.precision === '' && inputs.scale === '') {
		return 'Precision and scale cannot be empty';
	}
	if (inputs.precision === '') {
		return 'Precision is required';
	}
	if (inputs.scale === '') {
		return 'Scale is required';
	}
};

const getDecimalTypeInputs = (type: Type | null | undefined): DecimalTypeInputs => {
	const valueType = getPropertyValueType(type);
	if (valueType?.kind !== 'decimal') {
		return { precision: '', precisionBadInput: false, scale: '', scaleBadInput: false };
	}
	return {
		precision: valueType.precision == null ? '' : String(valueType.precision),
		precisionBadInput: false,
		scale: String(valueType.scale ?? 0),
		scaleBadInput: false,
	};
};

const checkNumericRange = (type: NumericType, inputs: NumericRangeInputs): string | undefined => {
	const values: { input: string; label: string; number: number }[] = [];
	const inputEntries: [keyof NumericRangeInputs, NumericRangeInput][] = [
		['minimum', inputs.minimum],
		['maximum', inputs.maximum],
	];
	for (const [name, input] of inputEntries) {
		const label = name === 'minimum' ? 'Minimum' : 'Maximum';
		if (input.badInput) {
			const shortLabel = name === 'minimum' ? 'Min' : 'Max';
			return `${shortLabel} must be ${type.kind === 'int' ? 'an integer' : 'a number'}`;
		}
		if (input.value === '') {
			continue;
		}
		const number = Number(input.value);
		if (!Number.isFinite(number)) {
			return `${label} must be a finite number`;
		}
		values.push({ input: input.value, label, number });
	}

	if (type.kind === 'int') {
		const [minimum, maximum] = getIntegerTypeRange(type);
		for (const value of values) {
			if (
				getDecimalPlaces(value.input) !== 0 ||
				value.number < Number(minimum) ||
				value.number > Number(maximum)
			) {
				const label = value.label === 'Minimum' ? 'Min' : 'Max';
				return `${label} must be an integer between ${minimum} and ${maximum}`;
			}
		}
	} else if (type.kind === 'float') {
		for (const value of values) {
			const represented = type.bitSize === 32 ? Math.fround(value.number) : value.number;
			// Inspect the coefficient because Number may have underflowed a nonzero input to zero.
			const inputIsZero = !/[1-9]/.test(value.input.split(/[eE]/)[0]);
			if (!Number.isFinite(represented) || (!inputIsZero && represented === 0)) {
				return `${value.label} must fit a ${type.bitSize}-bit float`;
			}
		}
	} else {
		const scale = type.scale ?? 0;
		const maximum = 10 ** (type.precision - scale) - 10 ** -scale;
		for (const value of values) {
			if (Math.abs(value.number) > maximum || getDecimalPlaces(value.input) > scale) {
				return `${value.label} does not fit decimal(${type.precision},${scale})`;
			}
		}
	}

	if (
		inputs.minimum.value !== '' &&
		inputs.maximum.value !== '' &&
		Number(inputs.maximum.value) < Number(inputs.minimum.value)
	) {
		return 'Max must be greater than or equal to Min';
	}
};

const formatIntegerLimit = (limit: bigint): string => {
	const negative = limit < BigInt(0);
	const digits = (negative ? -limit : limit).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
	return (negative ? '−' : '') + digits;
};

const getDecimalPlaces = (value: string): number => {
	const [coefficient, exponentText] = value.toLowerCase().split('e');
	const digits = coefficient.replace(/^[+-]/, '').replace('.', '');
	if (!/[1-9]/.test(digits)) {
		return 0;
	}
	const fractionalDigits = coefficient.split('.')[1]?.length ?? 0;
	const decimalPlaces = Math.max(0, fractionalDigits - Number(exponentText || 0));
	const trailingZeros = digits.match(/0+$/)?.[0].length ?? 0;
	return Math.max(0, decimalPlaces - trailingZeros);
};

const getIntegerTypeRange = (type: IntType): [bigint, bigint] => {
	const one = BigInt(1);
	const magnitude = one << BigInt(type.bitSize - (type.unsigned ? 0 : 1));
	return type.unsigned ? [BigInt(0), magnitude - one] : [-magnitude, magnitude - one];
};

const getNumericRangeInputs = (type: Type | null | undefined): NumericRangeInputs => {
	const valueType = getPropertyValueType(type);
	if (!isNumericType(valueType)) {
		return {
			minimum: { value: '', badInput: false },
			maximum: { value: '', badInput: false },
		};
	}
	return {
		minimum: { value: valueType.minimum == null ? '' : String(valueType.minimum), badInput: false },
		maximum: { value: valueType.maximum == null ? '' : String(valueType.maximum), badInput: false },
	};
};

const getNumericTypeError = (
	type: NumericType,
	decimalTypeInputs: DecimalTypeInputs,
	numericRangeInputs: NumericRangeInputs,
): PropertyTypeError | null => {
	if (type.kind === 'decimal') {
		const inputError = checkDecimalTypeInputs(decimalTypeInputs);
		if (inputError != null) {
			return { location: 'decimal-constraints', message: inputError };
		}
		const error = checkDecimalType(type);
		if (error != null) {
			return { location: 'decimal-constraints', message: error };
		}
	}
	const error = checkNumericRange(type, numericRangeInputs);
	if (error != null) {
		return { location: 'numeric-range', message: error };
	}
	return null;
};

const isNumericType = (type: Type | null | undefined): type is NumericType => {
	return type?.kind === 'int' || type?.kind === 'float' || type?.kind === 'decimal';
};

const validatePropertyName = (name: string) => {
	if (name === '') {
		throw new Error('Name cannot be empty');
	}
	if (/\s/.test(name)) {
		throw new Error('Name cannot contain spaces');
	}
	if (name.startsWith('_')) {
		throw new Error('Profile schema property names cannot start with an underscore');
	}
	if (/^[0-9]/.test(name)) {
		throw new Error('Name cannot start with a number');
	}
	if (!/^[A-Za-z_]/.test(name)) {
		throw new Error('Name must start with an ASCII alphabet character or an underscore');
	}
	if (!/^.[A-Za-z0-9_]*$/.test(name)) {
		throw new Error('Name must contain only ASCII alphabet characters, digits and underscores');
	}
};

const getPropertyNameError = (
	name: string,
	parentKey: string,
	originalName: string | undefined,
	parents: PropertyParent[] | undefined,
): string => {
	try {
		validatePropertyName(name);
	} catch (err) {
		return err.message;
	}
	if (name === originalName) {
		return '';
	}
	const parent = parents?.find((candidate) => candidate.key === parentKey);
	if (parent == null || !parent.propertyNames.includes(name)) {
		return '';
	}
	const parentLabel = parent.key === '' ? parent.label : `“${parent.label}”`;
	return `A property named “${name}” already exists in ${parentLabel}.`;
};

export { PropertyForm };
