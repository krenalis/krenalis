import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import './PropertyForm.css';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlRadioButton from '@shoelace-style/shoelace/dist/react/radio-button/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlTextarea from '@shoelace-style/shoelace/dist/react/textarea/index.js';
import type SlCheckboxElement from '@shoelace-style/shoelace/dist/components/checkbox/checkbox.component.js';
import type SlInputElement from '@shoelace-style/shoelace/dist/components/input/input.component.js';
import type SlTextareaElement from '@shoelace-style/shoelace/dist/components/textarea/textarea.component.js';
import AppContext from '../../../context/AppContext';
import Type, {
	CountryFormat,
	DecimalType,
	FloatBitSize,
	FloatType,
	IntBitSize,
	IntType,
	Semantic,
	StringType,
} from '../../../lib/api/types/types';
import { ProfileRoleAssignments, ProfileRoleID } from '../../../lib/api/types/workspace';
import TransformedConnection from '../../../lib/core/connection';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { COMMON_CURRENCY_OPTION_COUNT, CURRENCY_OPTIONS } from '../../helpers/currencies';
import {
	getCompatibleProfileRoles,
	getProfileRole,
	isProfileRoleCompatible,
	PROFILE_ROLES,
} from '../../helpers/profileRoles';
import {
	DURATION_UNIT_OPTIONS,
	getPropertyValueType,
	isSuitableAsIdentifier,
	replacePropertyValueType,
	UNIT_OF_MEASURE_OPTIONS,
} from '../../helpers/types';
import {
	SchemaPropertyIdentifierLabel,
	SchemaPropertyIdentifierValue,
	SchemaPropertyInfoTooltip,
	SchemaPropertyPrimarySourceLabel,
} from '../Schema/SchemaPropertyGrid';
import { ProfileRoleSelector } from './ProfileRoleSelector';
import { PropertyTypeSelector, type PropertyTypeSelectorRef } from './PropertyTypeSelector';
import { getParentPropertyKey } from './SchemaEdit.helpers';
import { PropertyFieldChanges, PropertyParent, PropertyToEdit } from './useSchemaEdit';

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
	location:
		| 'type'
		| 'string-constraints'
		| 'decimal-constraints'
		| 'numeric-range'
		| 'measurement-unit'
		| 'duration-unit';
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

const preventReadOnlyTypeControlFocus = (event: React.PointerEvent) => {
	event.preventDefault();
};

const removeReadOnlyTypeControlFromTabOrder = (control: SlCheckboxElement | SlInputElement | null) => {
	if (control == null) {
		return;
	}
	// Setting tabindex on the Shoelace host does not remove its internal input from the tab order.
	void control.updateComplete.then(() => {
		control.input.tabIndex = -1;
	});
};

interface PropertyFormProps {
	assignedRole: ProfileRoleID | null;
	assignedRoles: ProfileRoleAssignments;
	fieldChanges?: PropertyFieldChanges;
	formID: string;
	identifierPosition?: number;
	materializedSemantic?: Semantic;
	propertyToEdit: PropertyToEdit;
	primarySources: Record<string, string>;
	parents?: PropertyParent[];
	propertyPaths: Readonly<Record<string, string>>;
	showParent?: boolean;
	onSave: (
		property: PropertyToEdit,
		primarySource: string | null,
		assignedRole: ProfileRoleID | null,
		rolesToUnassign: readonly ProfileRoleID[],
	) => void;
	onDirtyChange?: (dirty: boolean) => void;
	onValidityChange?: (valid: boolean) => void;
}

interface PendingTypeChange {
	description: React.ReactNode;
	rolesToUnassign: ProfileRoleID[];
	semantic?: Semantic;
	type: Type;
}

const PropertyForm = ({
	assignedRole: assignedRoleToEdit,
	assignedRoles,
	fieldChanges,
	formID,
	identifierPosition,
	materializedSemantic,
	propertyToEdit,
	primarySources,
	parents,
	propertyPaths,
	showParent,
	onSave,
	onDirtyChange,
	onValidityChange,
}: PropertyFormProps) => {
	const [property, setProperty] = useState<PropertyToEdit>(() => structuredClone(propertyToEdit));
	const [primarySource, setPrimarySource] = useState<string | null>(primarySources[propertyToEdit.key] || null);
	const [assignedRole, setAssignedRole] = useState<ProfileRoleID | null>(assignedRoleToEdit);
	const [descendantRolesToUnassign, setDescendantRolesToUnassign] = useState<ProfileRoleID[]>([]);
	const [roleToReassign, setRoleToReassign] = useState<ProfileRoleID | null>(null);
	const [pendingTypeChange, setPendingTypeChange] = useState<PendingTypeChange | null>(null);
	const [typeSelectorRevision, setTypeSelectorRevision] = useState(0);
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
		setAssignedRole(assignedRoleToEdit);
		setDescendantRolesToUnassign([]);
		setRoleToReassign(null);
		setPendingTypeChange(null);
		setTypeSelectorRevision(0);
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
			assignedRoleToEdit,
			[],
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
	}, [assignedRoleToEdit, propertyToEdit]);

	useEffect(() => {
		if (initialState.current === '') {
			return;
		}
		onDirtyChange?.(
			propertyFormStateKey(
				property,
				primarySource,
				decimalTypeInputs,
				numericRangeInputs,
				assignedRole,
				descendantRolesToUnassign,
			) !== initialState.current,
		);
	}, [assignedRole, descendantRolesToUnassign, property, primarySource, decimalTypeInputs, numericRangeInputs]);

	useEffect(() => {
		const type = getPropertyValueType(property.type);
		setTypeError((current) => {
			if (current != null && current.location !== 'decimal-constraints' && current.location !== 'numeric-range') {
				return current;
			}
			if ((property.semantic == null || hasSemanticDecimalRange(property.semantic)) && isNumericType(type)) {
				return getNumericTypeError(type, decimalTypeInputs, numericRangeInputs);
			}
			return null;
		});
	}, [property.type, property.semantic, decimalTypeInputs, numericRangeInputs]);

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
			showParent ||
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

	const applyTypeChange = (type: Type | null, semantic?: Semantic) => {
		updateProperty((nextProperty) => {
			nextProperty.type = type;
			if (semantic == null) {
				delete nextProperty.semantic;
			} else {
				nextProperty.semantic = semantic;
			}
		});
		setDecimalTypeInputs(getDecimalTypeInputs(type));
		setNumericRangeInputs(getNumericRangeInputs(type));
		if (type?.kind === 'object' || type?.kind === 'array') {
			setPrimarySource(null);
		}
		setTypeError(null);
	};

	const onChangeType = (type: Type | null, semantic?: Semantic) => {
		if (type == null) {
			applyTypeChange(type, semantic);
			return;
		}
		const proposedProperty = { ...property, type, semantic };
		const rolesToUnassign: ProfileRoleID[] = [];
		if (assignedRole != null && !isProfileRoleCompatible(assignedRole, proposedProperty)) {
			rolesToUnassign.push(assignedRole);
		}
		if (propertyToEdit.key != null && propertyToEdit.type?.kind === 'object' && type.kind !== 'object') {
			for (const role of PROFILE_ROLES) {
				if (
					assignedRoles[role.id].startsWith(`${propertyToEdit.key}.`) &&
					!descendantRolesToUnassign.includes(role.id)
				) {
					rolesToUnassign.push(role.id);
				}
			}
		}
		const uniqueRolesToUnassign = [...new Set(rolesToUnassign)];
		if (uniqueRolesToUnassign.length === 0) {
			if (type.kind === 'object') {
				setDescendantRolesToUnassign([]);
			}
			applyTypeChange(type, semantic);
			return;
		}

		const typeLabel = semantic?.kind || type.kind;
		let description: React.ReactNode;
		if (uniqueRolesToUnassign.length === 1 && uniqueRolesToUnassign[0] === assignedRole) {
			const role = getProfileRole(uniqueRolesToUnassign[0]);
			description =
				type.kind === 'array' ? (
					<>
						Changing this property to an array will remove the <code>{role.label}</code> role from this
						property.
					</>
				) : (
					<>
						Changing the type to <code>{typeLabel}</code> will remove the <code>{role.label}</code> role
						from this property.
					</>
				);
		} else {
			const roleLabels = uniqueRolesToUnassign.map((role) => getProfileRole(role).label).join(', ');
			description = (
				<>
					Changing the type to <code>{typeLabel}</code> will remove these roles: {roleLabels}.
				</>
			);
		}
		setTypeSelectorRevision((current) => current + 1);
		setPendingTypeChange({ description, rolesToUnassign: uniqueRolesToUnassign, semantic, type });
	};

	const onChangeAssignedRole = (role: ProfileRoleID | null) => {
		if (role == null || role === assignedRole) {
			setAssignedRole(role);
			return;
		}
		const assignedPropertyKey = assignedRoles[role];
		if (assignedPropertyKey !== '' && assignedPropertyKey !== propertyToEdit.key) {
			setRoleToReassign(role);
			return;
		}
		setAssignedRole(role);
	};

	const onConfirmReassign = () => {
		setAssignedRole(roleToReassign);
		setRoleToReassign(null);
	};

	const onConfirmTypeChange = () => {
		if (pendingTypeChange == null) {
			return;
		}
		if (assignedRole != null && pendingTypeChange.rolesToUnassign.includes(assignedRole)) {
			setAssignedRole(null);
		}
		setDescendantRolesToUnassign(pendingTypeChange.rolesToUnassign.filter((role) => role !== assignedRole));
		applyTypeChange(pendingTypeChange.type, pendingTypeChange.semantic);
		setPendingTypeChange(null);
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

	const onChangeCurrency = (event) => {
		const currency = event.target.value;
		updateProperty((nextProperty) => {
			if (nextProperty.semantic?.kind !== 'money') {
				return;
			}
			if (currency === 'none') {
				delete nextProperty.semantic.currency;
			} else {
				nextProperty.semantic.currency = currency;
			}
		});
	};

	const onChangeCountryFormat = (event) => {
		const format = event.target.value as CountryFormat;
		updateProperty((nextProperty) => {
			if (nextProperty.semantic?.kind !== 'country' || nextProperty.type == null) {
				return;
			}
			const valueType = structuredClone(getPropertyValueType(nextProperty.type));
			if (valueType?.kind !== 'string') {
				return;
			}
			nextProperty.semantic.format = format;
			valueType.maxLength = format === 'iso_3166_1_alpha_2' ? 2 : 3;
			delete valueType.maxBytes;
			nextProperty.type = replacePropertyValueType(nextProperty.type, valueType);
		});
	};

	const onChangeMeasurementUnit = (event) => {
		updateProperty((nextProperty) => {
			if (nextProperty.semantic?.kind === 'measurement') {
				nextProperty.semantic.unit = event.target.value;
			}
		});
		setTypeError((current) => (current?.location === 'measurement-unit' ? null : current));
	};

	const onChangeDurationUnit = (event) => {
		updateProperty((nextProperty) => {
			if (nextProperty.semantic?.kind === 'duration') {
				nextProperty.semantic.unit = event.target.value;
			}
		});
		setTypeError((current) => (current?.location === 'duration-unit' ? null : current));
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
			onSave(property, primarySource, assignedRole, descendantRolesToUnassign);
		} catch (err) {
			setNameError(err.message);
			return;
		}
		initialState.current = propertyFormStateKey(
			property,
			primarySource,
			decimalTypeInputs,
			numericRangeInputs,
			assignedRole,
			descendantRolesToUnassign,
		);
		onDirtyChange?.(false);
	};

	const valueType = getPropertyValueType(property.type);
	const showPercentageControls = valueType?.kind === 'decimal' && property.semantic?.kind === 'percentage';
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
	const semantic = property.semantic;
	const selectedCurrencyOption =
		semantic?.kind === 'money' ? CURRENCY_OPTIONS.find((option) => option.code === semantic.currency) : undefined;
	const selectedMeasurementUnitOption =
		semantic?.kind === 'measurement'
			? UNIT_OF_MEASURE_OPTIONS.find((option) => option.value === semantic.unit)
			: undefined;
	const selectedDurationUnitOption =
		semantic?.kind === 'duration'
			? DURATION_UNIT_OPTIONS.find((option) => option.value === semantic.unit)
			: undefined;
	const propertyParentPath = propertyPaths[parentKey] ?? '';
	const propertyPath = propertyParentPath === '' ? property.name : `${propertyParentPath}.${property.name}`;
	const showAssignedRole = getCompatibleProfileRoles(property).length > 0 || assignedRole != null;
	const reassignedRole = roleToReassign == null ? null : getProfileRole(roleToReassign);
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
				ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
				size='small'
				value={numericRangeInputs.minimum.value}
				type='number'
				readonly={!canEditType}
				tabIndex={canEditType ? undefined : -1}
				noSpinButtons={!canEditType}
				step={numericRangeStep}
				placeholder={minimumPlaceholder}
				onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
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
				ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
				size='small'
				value={numericRangeInputs.maximum.value}
				type='number'
				readonly={!canEditType}
				tabIndex={canEditType ? undefined : -1}
				noSpinButtons={!canEditType}
				step={numericRangeStep}
				placeholder={maximumPlaceholder}
				onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
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
			{showParent && parents != null && (
				<div className='property-form__control'>
					<SlSelect
						className='property-form__parent'
						label='Add property to'
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
			<div
				className={`property-form__control${
					showPercentageControls ? ' property-form__control--percentage-type' : ''
				}`}
			>
				<div className='property-form__label'>
					<PropertyFormLabel modified={fieldChanges?.type}>Type</PropertyFormLabel>
				</div>
				<PropertyTypeSelector
					key={typeSelectorRevision}
					ref={typeSelectorRef}
					type={property.type}
					semantic={property.semantic}
					canEditType={canEditType}
					materializedSemantic={materializedSemantic}
					onChange={onChangeType}
				/>
				{showPercentageControls && (
					<div className='property-form__percentage-description'>
						<SlBadge className='property-form__percentage-badge' pill variant='neutral'>
							<span className='property-form__percentage-badge-text'>0.9 represents 90%</span>
						</SlBadge>
					</div>
				)}
				{typeError?.location === 'type' && (
					<PropertyFormError name='type'>{typeError.message}</PropertyFormError>
				)}
			</div>
			{property.semantic?.kind === 'country' && (
				<div className='property-form__constraints property-form__constraints--country'>
					{canEditType ? (
						<SlSelect
							className='property-form__country-format'
							size='small'
							value={property.semantic.format}
							onSlChange={onChangeCountryFormat}
						>
							<PropertyFormLabel slot='label'>Format</PropertyFormLabel>
							<SlOption value='iso_3166_1_alpha_2'>2-letter ISO code</SlOption>
							<SlOption value='iso_3166_1_alpha_3'>3-letter ISO code</SlOption>
						</SlSelect>
					) : (
						<SlInput
							className='property-form__country-format'
							ref={removeReadOnlyTypeControlFromTabOrder}
							size='small'
							value={
								property.semantic.format === 'iso_3166_1_alpha_2'
									? '2-letter ISO code'
									: '3-letter ISO code'
							}
							readonly
							tabIndex={-1}
							onPointerDown={preventReadOnlyTypeControlFocus}
						>
							<PropertyFormLabel slot='label'>Format</PropertyFormLabel>
						</SlInput>
					)}
				</div>
			)}
			{valueType?.kind === 'string' && property.semantic == null && (
				<div className='property-form__constraints property-form__constraints--length'>
					<SlInput
						ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
						label='Max characters'
						size='small'
						value={valueType.maxLength == null ? '' : String(valueType.maxLength)}
						type='number'
						readonly={!canEditType}
						tabIndex={canEditType ? undefined : -1}
						noSpinButtons={!canEditType}
						min={1}
						max={MAX_STRING_LENGTH}
						step={1}
						onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
						onSlInput={onInputMaxLength}
					/>
					<SlInput
						ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
						label='Max bytes'
						size='small'
						value={valueType.maxBytes == null ? '' : String(valueType.maxBytes)}
						type='number'
						readonly={!canEditType}
						tabIndex={canEditType ? undefined : -1}
						noSpinButtons={!canEditType}
						min={1}
						max={MAX_STRING_LENGTH}
						step={1}
						onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
						onSlInput={onInputMaxBytes}
					/>
					{typeError?.location === 'string-constraints' && (
						<PropertyFormError name='string-constraints'>{typeError.message}</PropertyFormError>
					)}
				</div>
			)}
			{(valueType?.kind === 'int' || valueType?.kind === 'float') && property.semantic == null && (
				<div
					className={`property-form__constraints property-form__constraints--${
						valueType.kind === 'int' ? 'integer' : 'float'
					}`}
				>
					{valueType.kind === 'int' &&
						(canEditType ? (
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
							<SlInput
								className='property-form__integer-sign'
								ref={removeReadOnlyTypeControlFromTabOrder}
								label='Sign'
								size='small'
								value={valueType.unsigned ? 'unsigned' : 'signed'}
								readonly
								tabIndex={-1}
								onPointerDown={preventReadOnlyTypeControlFocus}
							/>
						))}
					{canEditType ? (
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
					) : (
						<SlInput
							className='property-form__bit-size'
							ref={removeReadOnlyTypeControlFromTabOrder}
							label={valueType.kind === 'int' ? 'Integer size' : 'Bit size'}
							size='small'
							value={`${valueType.bitSize}-bit`}
							readonly
							tabIndex={-1}
							onPointerDown={preventReadOnlyTypeControlFocus}
						/>
					)}
					{valueType.kind === 'float' && (
						<div
							className={`property-form__float-special-values${
								canEditType ? '' : ' property-form__float-special-values--read-only'
							}`}
							onClickCapture={
								canEditType
									? undefined
									: (event) => {
											event.preventDefault();
											event.stopPropagation();
										}
							}
							onPointerDownCapture={canEditType ? undefined : preventReadOnlyTypeControlFocus}
						>
							<SlCheckbox
								ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
								size='small'
								checked={!valueType.real}
								aria-readonly={!canEditType ? 'true' : undefined}
								tabIndex={canEditType ? undefined : -1}
								onSlChange={canEditType ? onRealChange : undefined}
							>
								<span className='property-form__float-special-values-label'>Allow ±Inf and NaN</span>
							</SlCheckbox>
						</div>
					)}
					{numericRangeControls}
				</div>
			)}
			{valueType?.kind === 'decimal' && property.semantic == null && (
				<div className='property-form__constraints property-form__constraints--decimal'>
					<SlInput
						className='property-form__precision'
						ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
						label='Precision'
						size='small'
						value={decimalTypeInputs.precision}
						type='number'
						readonly={!canEditType}
						tabIndex={canEditType ? undefined : -1}
						noSpinButtons={!canEditType}
						max={MAX_DECIMAL_PRECISION}
						maxlength={2}
						onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
						onSlInput={onInputPrecision}
					/>
					<SlInput
						className='property-form__scale'
						ref={canEditType ? undefined : removeReadOnlyTypeControlFromTabOrder}
						label='Scale'
						size='small'
						value={decimalTypeInputs.scale}
						type='number'
						readonly={!canEditType}
						tabIndex={canEditType ? undefined : -1}
						noSpinButtons={!canEditType}
						max={MAX_DECIMAL_SCALE}
						maxlength={2}
						onPointerDown={canEditType ? undefined : preventReadOnlyTypeControlFocus}
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
			{property.semantic?.kind === 'money' && (
				<div className='property-form__constraints property-form__constraints--money'>
					{canEditType ? (
						<SlSelect
							className='property-form__currency'
							size='small'
							value={property.semantic.currency || 'none'}
							onSlChange={onChangeCurrency}
						>
							<PropertyFormLabel slot='label'>Currency</PropertyFormLabel>
							<SlOption value='none'>No currency specified</SlOption>
							<SlDivider />
							{CURRENCY_OPTIONS.map((option, index) => (
								<React.Fragment key={option.code}>
									{index === COMMON_CURRENCY_OPTION_COUNT && <SlDivider />}
									<SlOption value={option.code}>
										<span className='property-form__currency-option-code'>{option.code}</span>
										<span className='property-form__currency-option-separator'> · </span>
										<span>{option.name}</span>
										{option.symbol != null && (
											<span className='property-form__currency-option-symbol' slot='suffix'>
												{option.symbol}
											</span>
										)}
									</SlOption>
								</React.Fragment>
							))}
						</SlSelect>
					) : (
						<SlInput
							className='property-form__currency'
							ref={removeReadOnlyTypeControlFromTabOrder}
							size='small'
							value={
								selectedCurrencyOption == null
									? 'No currency specified'
									: `${selectedCurrencyOption.code} · ${selectedCurrencyOption.name}`
							}
							readonly
							tabIndex={-1}
							onPointerDown={preventReadOnlyTypeControlFocus}
						>
							<PropertyFormLabel slot='label'>Currency</PropertyFormLabel>
							{selectedCurrencyOption?.symbol != null && (
								<span className='property-form__currency-option-symbol' slot='suffix'>
									{selectedCurrencyOption.symbol}
								</span>
							)}
						</SlInput>
					)}
					{numericRangeControls}
				</div>
			)}
			{showPercentageControls && (
				<div className='property-form__constraints property-form__constraints--percentage'>
					{numericRangeControls}
				</div>
			)}
			{property.semantic?.kind === 'measurement' && (
				<div className='property-form__constraints property-form__constraints--measurement'>
					{canEditType ? (
						<SlSelect
							className='property-form__measurement-unit'
							size='small'
							value={property.semantic.unit}
							placeholder='Select a unit...'
							onSlChange={onChangeMeasurementUnit}
						>
							<PropertyFormLabel slot='label'>Unit</PropertyFormLabel>
							{!property.semantic.unit && (
								<SlOption className='property-form__unit-placeholder' value='' disabled />
							)}
							{UNIT_OF_MEASURE_OPTIONS.map((option, index) => (
								<React.Fragment key={option.value}>
									{option.groupLabel != null && index > 0 && <SlDivider />}
									{option.groupLabel != null && (
										<div className='property-form__unit-option-group' role='presentation'>
											{option.groupLabel}
										</div>
									)}
									<SlOption value={option.value}>
										<span>{option.label}</span>
										<span className='property-form__unit-option-separator'> · </span>
										<span className='property-form__unit-option-symbol'>{option.value}</span>
									</SlOption>
								</React.Fragment>
							))}
						</SlSelect>
					) : (
						<SlInput
							className='property-form__measurement-unit'
							ref={removeReadOnlyTypeControlFromTabOrder}
							size='small'
							value={
								selectedMeasurementUnitOption == null
									? ''
									: `${selectedMeasurementUnitOption.label} · ${selectedMeasurementUnitOption.value}`
							}
							readonly
							tabIndex={-1}
							onPointerDown={preventReadOnlyTypeControlFocus}
						>
							<PropertyFormLabel slot='label'>Unit</PropertyFormLabel>
						</SlInput>
					)}
					{typeError?.location === 'measurement-unit' && (
						<PropertyFormError name='measurement-unit'>{typeError.message}</PropertyFormError>
					)}
					{numericRangeControls}
				</div>
			)}
			{property.semantic?.kind === 'duration' && (
				<div className='property-form__constraints property-form__constraints--duration'>
					{canEditType ? (
						<SlSelect
							className='property-form__duration-unit'
							size='small'
							value={property.semantic.unit}
							placeholder='Select a unit...'
							onSlChange={onChangeDurationUnit}
						>
							<PropertyFormLabel slot='label'>Unit</PropertyFormLabel>
							{!property.semantic.unit && (
								<SlOption className='property-form__unit-placeholder' value='' disabled />
							)}
							{DURATION_UNIT_OPTIONS.map((option) => (
								<SlOption key={option.value} value={option.value}>
									<span>{option.label}</span>
									<span className='property-form__unit-option-separator'> · </span>
									<span className='property-form__unit-option-symbol'>{option.symbol}</span>
								</SlOption>
							))}
						</SlSelect>
					) : (
						<SlInput
							className='property-form__duration-unit'
							ref={removeReadOnlyTypeControlFromTabOrder}
							size='small'
							value={
								selectedDurationUnitOption == null
									? ''
									: `${selectedDurationUnitOption.label} · ${selectedDurationUnitOption.symbol}`
							}
							readonly
							tabIndex={-1}
							onPointerDown={preventReadOnlyTypeControlFocus}
						>
							<PropertyFormLabel slot='label'>Unit</PropertyFormLabel>
						</SlInput>
					)}
					{typeError?.location === 'duration-unit' && (
						<PropertyFormError name='duration-unit'>{typeError.message}</PropertyFormError>
					)}
				</div>
			)}
			{showAssignedRole && (
				<div className='property-form__control property-form__assigned-role'>
					<div className='property-form__label'>
						<PropertyFormLabel modified={fieldChanges?.profileRole}>
							Assigned role <span className='property-form__optional-label'>(optional)</span>
						</PropertyFormLabel>
					</div>
					<div className='property-form__assigned-role-description'>
						Specifies which profile concept this property represents.
					</div>
					<ProfileRoleSelector
						assignedRole={assignedRole}
						assignedRoles={assignedRoles}
						onChange={onChangeAssignedRole}
						property={property}
						propertyPaths={propertyPaths}
					/>
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
			<AlertDialog
				isOpen={roleToReassign != null}
				onClose={() => setRoleToReassign(null)}
				title='Reassign role?'
				actions={
					<>
						<SlButton onClick={() => setRoleToReassign(null)}>Cancel</SlButton>
						<SlButton variant='primary' onClick={onConfirmReassign}>
							Reassign
						</SlButton>
					</>
				}
			>
				{reassignedRole != null && (
					<p>
						{reassignedRole.label} is currently assigned to{' '}
						<code>{propertyPaths[assignedRoles[reassignedRole.id]]}</code>. Reassign the{' '}
						{reassignedRole.label} role to <code>{propertyPath}</code>?
						{assignedRole != null && assignedRole !== reassignedRole.id && (
							<>
								{' '}
								This will remove the <code>{getProfileRole(assignedRole).label}</code> role from{' '}
								<code>{propertyPath}</code>.
							</>
						)}
					</p>
				)}
			</AlertDialog>
			<AlertDialog
				isOpen={pendingTypeChange != null}
				onClose={() => setPendingTypeChange(null)}
				title='Change type?'
				actions={
					<>
						<SlButton onClick={() => setPendingTypeChange(null)}>Cancel</SlButton>
						<SlButton variant='primary' onClick={onConfirmTypeChange}>
							Change type
						</SlButton>
					</>
				}
			>
				<p>{pendingTypeChange?.description}</p>
			</AlertDialog>
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
	assignedRole: ProfileRoleID | null,
	rolesToUnassign: readonly ProfileRoleID[],
): string => {
	return JSON.stringify({
		property,
		primarySource,
		decimalTypeInputs,
		numericRangeInputs,
		assignedRole,
		rolesToUnassign,
	});
};

const validatePropertyType = (
	property: PropertyToEdit,
	decimalTypeInputs: DecimalTypeInputs,
	numericRangeInputs: NumericRangeInputs,
): PropertyTypeError | null => {
	if (property.type == null) {
		return { location: 'type', message: 'Type cannot be empty' };
	}
	if (property.semantic?.kind === 'measurement' && !property.semantic.unit) {
		return { location: 'measurement-unit', message: 'Unit is required' };
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
		const error = getNumericTypeError(type, decimalTypeInputs, numericRangeInputs);
		if (error != null) {
			return error;
		}
	}
	if (property.semantic?.kind === 'duration' && !property.semantic.unit) {
		return { location: 'duration-unit', message: 'Unit is required' };
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

const hasSemanticDecimalRange = (semantic?: Semantic): boolean => {
	return semantic?.kind === 'money' || semantic?.kind === 'percentage' || semantic?.kind === 'measurement';
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
