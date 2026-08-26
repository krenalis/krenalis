import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import './PropertyDialog.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlRadioButton from '@shoelace-style/shoelace/dist/react/radio-button/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlTextarea from '@shoelace-style/shoelace/dist/react/textarea/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
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
const PRIMARY_SOURCE_TOOLTIP = {
	content: 'The selected source has the highest precedence when populating this property.',
	label: 'About primary source',
};

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
	const [isNameEditable, setIsNameEditable] = useState(propertyToEdit.key == null);
	const [nameError, setNameError] = useState('');
	const [typeError, setTypeError] = useState('');
	const initialState = useRef('');
	const nameInputRef = useRef<any>();
	const typeSelectorRef = useRef<PropertyTypeSelectorRef>();

	const { connections } = useContext(AppContext);
	const isEditing = propertyToEdit.key != null;
	const canEditType = !isEditing || propertyToEdit.isEditable === true;

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
		setProperty(nextProperty);
		setPrimarySource(nextPrimarySource);
		setIsNameEditable(propertyToEdit.key == null);
		setNameError('');
		setTypeError('');
		initialState.current = propertyFormStateKey(nextProperty, nextPrimarySource);
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
		onDirtyChange?.(propertyFormStateKey(property, primarySource) !== initialState.current);
	}, [property, primarySource]);

	useEffect(() => {
		onValidityChange?.(nameError === '' && typeError === '');
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
		try {
			validatePropertyName(name);
			setNameError('');
		} catch (err) {
			setNameError(err.message);
		}
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
		if (type?.kind === 'object' || type?.kind === 'array') {
			setPrimarySource(null);
		}
		setTypeError('');
	};

	const onChangeBitSize = (event) => {
		updateValueType((type: IntType | FloatType) => {
			type.bitSize = Number(event.target.value) as IntBitSize | FloatBitSize;
		});
	};

	const onInputPrecision = (event) => {
		updateValueType((type: DecimalType) => {
			type.precision = Number(event.target.value);
		});
		setTypeError('');
	};

	const onInputScale = (event) => {
		updateValueType((type: DecimalType) => {
			type.scale = Number(event.target.value);
		});
		setTypeError('');
	};

	const onRealChange = () => {
		updateValueType((type: FloatType) => {
			type.real = !type.real;
		});
	};

	const onUnsignedChange = (event) => {
		const unsigned = event.currentTarget.value === 'unsigned';
		updateValueType((type: IntType) => {
			type.unsigned = unsigned;
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
		setTypeError('');
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
		setTypeError('');
	};

	const onInputDescription = (event) => {
		updateProperty((nextProperty) => {
			nextProperty.description = event.target.value;
		});
	};

	const onChangePrimarySource = (event) => {
		setPrimarySource(event.target.value === 'none' ? null : event.target.value);
	};

	const onSubmit = (event: React.FormEvent) => {
		event.preventDefault();
		try {
			validatePropertyName(property.name);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		const error = validatePropertyType(property);
		if (error !== '') {
			setTypeError(error);
			return;
		}
		try {
			onSave(property, primarySource);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		initialState.current = propertyFormStateKey(property, primarySource);
		onDirtyChange?.(false);
	};

	const valueType = getPropertyValueType(property.type);
	let decimalDescription: string | null = null;
	if (
		valueType?.kind === 'decimal' &&
		Number.isInteger(valueType.precision) &&
		Number.isInteger(valueType.scale) &&
		checkDecimalType(valueType) == null
	) {
		const integerDigits = valueType.precision - valueType.scale;
		decimalDescription = `Up to ${integerDigits} ${integerDigits === 1 ? 'digit' : 'digits'} before and ${valueType.scale} after the decimal point.`;
	}
	const selectedConnection = sourceConnections.find((connection) => connection.id === primarySource);

	return (
		<form className='property-form' id={formID} onSubmit={onSubmit}>
			<div className='property-form__control property-dialog__control--name'>
				<SlInput
					className='property-dialog__name-input'
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
				{typeError !== '' && <PropertyFormError name='type'>{typeError}</PropertyFormError>}
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
				</div>
			)}
			{(valueType?.kind === 'int' || valueType?.kind === 'float') && canEditType && (
				<div
					className={`property-form__constraints${
						valueType.kind === 'int' ? ' property-form__constraints--integer' : ''
					}`}
				>
					<SlSelect
						className='property-dialog__bitsize'
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
					{valueType.kind === 'int' ? (
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
					) : (
						<SlCheckbox size='small' checked={!valueType.real} onSlChange={onRealChange}>
							Allow Infinity and NaN
						</SlCheckbox>
					)}
				</div>
			)}
			{valueType?.kind === 'decimal' && canEditType && (
				<div className='property-form__constraints property-form__constraints--decimal'>
					<SlInput
						className='property-dialog__precision'
						label='Precision'
						size='small'
						value={String(valueType.precision)}
						type='number'
						max={MAX_DECIMAL_PRECISION}
						maxlength={2}
						onSlInput={onInputPrecision}
					/>
					<SlInput
						className='property-dialog__scale'
						label='Scale'
						size='small'
						value={String(valueType.scale)}
						type='number'
						max={MAX_DECIMAL_SCALE}
						maxlength={2}
						onSlInput={onInputScale}
					/>
					{decimalDescription != null && (
						<div className='property-form__decimal-description'>{decimalDescription}</div>
					)}
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
			{property.type?.kind !== 'object' && property.type?.kind !== 'array' && (
				<div className='property-form__control' onKeyDownCapture={onKeyDownPrimarySource}>
					{sourceConnections.length === 0 ? (
						<>
							<div className='property-form__label'>
								<PropertyFormLabel
									modified={fieldChanges?.primarySource}
									tooltip={PRIMARY_SOURCE_TOOLTIP}
								>
									Primary source
								</PropertyFormLabel>
							</div>
							<div className='property-form__empty-value'>No source connections are available.</div>
						</>
					) : (
						<SlSelect
							className='property-dialog__primary-source'
							value={primarySource == null ? 'none' : primarySource}
							name='primary-source'
							onSlChange={onChangePrimarySource}
						>
							<PropertyFormLabel
								slot='label'
								modified={fieldChanges?.primarySource}
								tooltip={PRIMARY_SOURCE_TOOLTIP}
							>
								Primary source
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
			{tooltip != null && (
				<SlTooltip className='property-form__tooltip' content={tooltip.content} placement='top' hoist={true}>
					<button className='property-form__info' type='button' aria-label={tooltip.label}>
						<SlIcon name='info-circle' />
					</button>
				</SlTooltip>
			)}
			{modified && <span className='property-form__modified-dot' role='img' aria-label='Modified' />}
		</span>
	);
};

const PropertyFormError = ({ name, children }: { name: string; children: React.ReactNode }) => {
	return (
		<div className='property-dialog__control-error' data-error-on={name}>
			<SlIcon name='exclamation-circle' />
			{children}
		</div>
	);
};

const propertyFormStateKey = (property: PropertyToEdit, primarySource: string | null): string => {
	return JSON.stringify({ property, primarySource });
};

const validatePropertyType = (property: PropertyToEdit): string => {
	if (property.type == null) {
		return 'Type cannot be empty';
	}
	const type = getPropertyValueType(property.type);
	if (type.kind === 'string') {
		if (
			type.maxLength != null &&
			(!Number.isInteger(type.maxLength) || type.maxLength < 1 || type.maxLength > MAX_STRING_LENGTH)
		) {
			return `Max characters must be an integer in range [1, ${MAX_STRING_LENGTH}]`;
		}
		if (
			type.maxBytes != null &&
			(!Number.isInteger(type.maxBytes) || type.maxBytes < 1 || type.maxBytes > MAX_STRING_LENGTH)
		) {
			return `Max bytes must be an integer in range [1, ${MAX_STRING_LENGTH}]`;
		}
	}
	if (type.kind === 'decimal') {
		const error = checkDecimalType(type);
		if (error != null) {
			return error;
		}
	}
	return '';
};

const checkDecimalType = (type: DecimalType): string | undefined => {
	if (type.precision < 1 || type.precision > MAX_DECIMAL_PRECISION) {
		return `Precision must be in range [1, ${MAX_DECIMAL_PRECISION}]`;
	}
	if (type.scale < 0 || type.scale > MAX_DECIMAL_SCALE) {
		return `Scale must be in range [0, ${MAX_DECIMAL_SCALE}]`;
	}
	if (type.scale > type.precision) {
		return 'Scale cannot be greater than precision';
	}
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

export { PropertyForm };
