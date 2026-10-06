import React, { useEffect, useRef, useState } from 'react';
import './PropertyTypeSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import Type, { DurationUnit, TypeKind, UnitOfMeasure } from '../../../lib/api/types/types';
import {
	getPropertyValueType,
	getTypeSemantic,
	toCompactPhysicalType,
	toProfileSchemaPhysicalType,
	toSemanticLabel,
} from '../../helpers/types';

type PropertyStructure = 'one' | 'array' | 'object' | 'map';

type PropertyTypeOptionID =
	| TypeKind
	| 'email'
	| 'phone'
	| 'url'
	| 'country'
	| 'duration'
	| 'money'
	| 'percentage'
	| 'measurement';

interface PropertyStructureOption {
	description: string;
	icon: string;
	id: PropertyStructure;
	label: string;
	triggerLabel: string;
}

interface PropertyTypeOption {
	create: () => Type;
	description?: string;
	id: PropertyTypeOptionID;
	kind: TypeKind;
	separated?: boolean;
}

interface PropertyTypeSelectorProps {
	canEditType: boolean;
	onChange: (type: Type | null) => void;
	type: Type | null;
}

// The empty value exists only while editing and cannot pass form validation.
const EMPTY_DURATION_UNIT = '' as DurationUnit;
const EMPTY_UNIT_OF_MEASURE = '' as UnitOfMeasure;
const PROFILE_SEMANTIC_DECIMAL_TYPE = { kind: 'decimal', precision: 18, scale: 4 } as const;

const PROPERTY_STRUCTURE_OPTIONS: PropertyStructureOption[] = [
	{
		id: 'one',
		label: 'one value',
		triggerLabel: 'one value',
		description: 'Exactly one value',
		icon: '1-circle',
	},
	{
		id: 'array',
		label: 'array',
		triggerLabel: 'array of',
		description: 'Ordered collection of values',
		icon: 'list-ul',
	},
	{
		id: 'object',
		label: 'object',
		triggerLabel: 'object',
		description: 'Related properties, each with its own type',
		icon: 'braces',
	},
	{
		id: 'map',
		label: 'map',
		triggerLabel: 'map of',
		description: 'Collection of values stored under text keys',
		icon: 'braces-asterisk',
	},
];

const PROPERTY_TYPE_OPTIONS: PropertyTypeOption[] = [
	{
		id: 'email',
		kind: 'string',
		create: () => ({ kind: 'string', semantic: 'email' }),
	},
	{
		id: 'phone',
		kind: 'string',
		create: () => ({ kind: 'string', semantic: 'phone' }),
	},
	{
		id: 'country',
		kind: 'string',
		create: () => ({ kind: 'string', semantic: 'country', format: 'alpha-2' }),
	},
	{
		id: 'url',
		kind: 'string',
		create: () => ({ kind: 'string', semantic: 'url' }),
	},
	{
		id: 'string',
		kind: 'string',
		description: 'Text value, such as a name or code',
		create: () => ({ kind: 'string' }),
	},
	{
		id: 'boolean',
		kind: 'boolean',
		description: 'True or false',
		separated: true,
		create: () => ({ kind: 'boolean' }),
	},
	{
		id: 'duration',
		kind: 'int',
		separated: true,
		create: () => ({ kind: 'int', bitSize: 64, unsigned: false, semantic: 'duration', unit: EMPTY_DURATION_UNIT }),
	},
	{
		id: 'int',
		kind: 'int',
		description: 'Number with no decimal places',
		create: () => ({ kind: 'int', bitSize: 32, unsigned: false }),
	},
	{
		id: 'float',
		kind: 'float',
		description: 'Number with approximate precision',
		separated: true,
		create: () => ({ kind: 'float', bitSize: 64, real: false }),
	},
	{
		id: 'money',
		kind: 'decimal',
		separated: true,
		create: () => ({ ...PROFILE_SEMANTIC_DECIMAL_TYPE, semantic: 'money' }),
	},
	{
		id: 'percentage',
		kind: 'decimal',
		create: () => ({ ...PROFILE_SEMANTIC_DECIMAL_TYPE, semantic: 'percentage' }),
	},
	{
		id: 'measurement',
		kind: 'decimal',
		create: () => ({
			...PROFILE_SEMANTIC_DECIMAL_TYPE,
			semantic: 'measurement',
			unit: EMPTY_UNIT_OF_MEASURE,
		}),
	},
	{
		id: 'decimal',
		kind: 'decimal',
		description: 'Decimal number with fixed precision',
		create: () => ({ kind: 'decimal', precision: 10, scale: 0 }),
	},
	{
		id: 'datetime',
		kind: 'datetime',
		description: 'Date and time',
		separated: true,
		create: () => ({ kind: 'datetime' }),
	},
	{
		id: 'date',
		kind: 'date',
		description: 'Date without a time',
		create: () => ({ kind: 'date' }),
	},
	{
		id: 'time',
		kind: 'time',
		description: 'Time without a date',
		create: () => ({ kind: 'time' }),
	},
	{
		id: 'year',
		kind: 'year',
		description: 'Year',
		create: () => ({ kind: 'year' }),
	},
	{
		id: 'uuid',
		kind: 'uuid',
		description: 'UUID',
		separated: true,
		create: () => ({ kind: 'uuid' }),
	},
	{
		id: 'json',
		kind: 'json',
		description: 'JSON value',
		create: () => ({ kind: 'json' }),
	},
	{
		id: 'ip',
		kind: 'ip',
		description: 'IPv4 or IPv6 address',
		create: () => ({ kind: 'ip' }),
	},
];

const PropertyTypeSelector = ({ type, canEditType, onChange }: PropertyTypeSelectorProps) => {
	const [structure, setStructure] = useState<PropertyStructure>(() => getPropertyStructure(type));
	const dropdownRef = useRef<any>();
	const focusTypeAfterStructureSelectionRef = useRef(false);

	const valueType = getPropertyValueType(type);
	const semantic = getTypeSemantic(valueType);
	let typeLabel: string | null = null;
	if (valueType != null) {
		typeLabel = semantic == null ? valueType.kind : toSemanticLabel(semantic);
	}
	const physicalType = semantic == null ? null : toCompactPhysicalType(valueType);
	const selectedOption = getPropertyTypeOption(type);
	const selectedStructureOption =
		PROPERTY_STRUCTURE_OPTIONS.find((option) => option.id === structure) || PROPERTY_STRUCTURE_OPTIONS[0];
	const showValueTypeSelector = structure !== 'object';

	useEffect(() => {
		if (type != null) {
			setStructure(getPropertyStructure(type));
		}
	}, [type]);

	const onSelectStructure = (event) => {
		if (!canEditType) {
			return;
		}
		const nextStructure = event.detail.item.value as PropertyStructure;
		focusTypeAfterStructureSelectionRef.current = nextStructure !== 'object';
		if (nextStructure === structure) {
			return;
		}
		setStructure(nextStructure);
		if (nextStructure === 'object') {
			onChange({ kind: 'object', properties: [] });
			return;
		}
		if (valueType == null || valueType.kind === 'object') {
			onChange(null);
			return;
		}
		onChange(wrapPropertyValueType(valueType, nextStructure));
	};

	const onStructureMenuAfterHide = () => {
		if (!focusTypeAfterStructureSelectionRef.current) {
			return;
		}
		focusTypeAfterStructureSelectionRef.current = false;
		dropdownRef.current?.focusOnTrigger();
	};

	const onSelectOption = (event) => {
		if (!canEditType) {
			return;
		}
		const option = PROPERTY_TYPE_OPTIONS.find((candidate) => candidate.id === event.detail.item.value);
		if (option == null || option.id === selectedOption?.id) {
			return;
		}
		const selection = option.create();
		let nextValueType = valueType;
		if (getTypeSemantic(selection) != null || valueType?.kind !== option.kind) {
			nextValueType = selection;
		} else if (valueType != null && 'semantic' in valueType) {
			nextValueType = { ...valueType };
			delete nextValueType.semantic;
			if (nextValueType.kind === 'string') {
				delete nextValueType.format;
			}
			if (nextValueType.kind === 'decimal') {
				delete nextValueType.currency;
			}
			if ('unit' in nextValueType) {
				delete nextValueType.unit;
			}
		}
		if (nextValueType == null) {
			return;
		}
		onChange(wrapPropertyValueType(nextValueType, structure));
	};

	return (
		<div className='property-type-selector'>
			{type != null && (
				<div
					className={`property-type-selector__type-change-note${
						canEditType ? '' : ' property-type-selector__type-change-note--read-only'
					}`}
				>
					{canEditType
						? "Can't be changed once the property has been applied."
						: 'The type of an existing property cannot be changed.'}
				</div>
			)}
			<div
				className={`property-type-selector__controls${
					canEditType ? '' : ' property-type-selector__controls--read-only'
				}${showValueTypeSelector ? '' : ' property-type-selector__controls--structure-only'}`}
			>
				<SlDropdown
					className='property-type-selector__structure-dropdown'
					hoist
					placement='bottom-start'
					distance={6}
					disabled={!canEditType}
					onSlAfterHide={onStructureMenuAfterHide}
				>
					<SlButton
						className='property-type-selector__structure-trigger'
						slot='trigger'
						caret={canEditType}
						disabled={!canEditType}
						aria-label={`Structure: ${selectedStructureOption.label}`}
					>
						<SlIcon slot='prefix' name={selectedStructureOption.icon} />
						{selectedStructureOption.triggerLabel}
					</SlButton>
					<SlMenu className='property-type-selector__structure-menu' onSlSelect={onSelectStructure}>
						{PROPERTY_STRUCTURE_OPTIONS.map((option) => (
							<SlMenuItem
								className={`property-type-selector__structure-option${
									structure === option.id ? ' property-type-selector__structure-option--selected' : ''
								}`}
								key={option.id}
								data-structure-option={option.id}
								value={option.id}
							>
								<SlIcon slot='prefix' name={option.icon} />
								<span className='property-type-selector__structure-option-content'>
									<span className='property-type-selector__structure-option-label'>
										{option.label}
									</span>
									<span className='property-type-selector__structure-option-description'>
										{option.description}
									</span>
								</span>
								{structure === option.id && <SlIcon slot='suffix' name='check-lg' />}
							</SlMenuItem>
						))}
					</SlMenu>
				</SlDropdown>
				{showValueTypeSelector && (
					<SlDropdown
						className='property-type-selector__dropdown'
						ref={dropdownRef}
						hoist
						placement='bottom-end'
						distance={6}
						disabled={!canEditType}
					>
						<SlButton
							className='property-type-selector__trigger'
							slot='trigger'
							caret={canEditType}
							disabled={!canEditType}
							aria-label={valueType == null ? 'Select type' : undefined}
						>
							{valueType == null ? (
								<span className='property-type-selector__placeholder'>Select type...</span>
							) : (
								<span
									className='property-type-selector__type'
									title={physicalType == null ? typeLabel : `${typeLabel} · ${physicalType}`}
								>
									<span className='property-type-selector__type-label'>{typeLabel}</span>
									{physicalType != null && (
										<span className='property-type-selector__type-metadata'>
											<span className='property-type-selector__type-separator' aria-hidden='true'>
												{' · '}
											</span>
											<span className='property-type-selector__type-physical'>
												{physicalType}
											</span>
										</span>
									)}
								</span>
							)}
						</SlButton>
						<SlMenu className='property-type-selector__browser' onSlSelect={onSelectOption}>
							{PROPERTY_TYPE_OPTIONS.map((option) => {
								const optionType = option.create();
								const optionSemantic = getTypeSemantic(optionType);
								const label =
									optionSemantic == null ? optionType.kind : toSemanticLabel(optionSemantic);
								let metadata = option.description;
								if (optionSemantic != null) {
									metadata =
										optionSemantic === 'country' || optionSemantic === 'duration'
											? toCompactPhysicalType(optionType)
											: toProfileSchemaPhysicalType(optionType);
								}
								return (
									<SlMenuItem
										className={`property-type-selector__option${
											option.separated ? ' property-type-selector__option--separated' : ''
										}${
											selectedOption?.id === option.id
												? ' property-type-selector__option--selected'
												: ''
										}`}
										key={option.id}
										data-type-option={option.id}
										value={option.id}
									>
										<span className='property-type-selector__type'>
											<span className='property-type-selector__type-label'>{label}</span>
											{metadata != null && (
												<span
													className={`property-type-selector__type-metadata${
														optionSemantic == null
															? ''
															: ' property-type-selector__type-metadata--physical'
													}`}
												>
													<span
														className='property-type-selector__type-separator'
														aria-hidden='true'
													>
														{' · '}
													</span>
													<span
														className={
															optionSemantic == null
																? ''
																: 'property-type-selector__type-physical'
														}
													>
														{metadata}
													</span>
												</span>
											)}
										</span>
										{selectedOption?.id === option.id && <SlIcon slot='suffix' name='check-lg' />}
									</SlMenuItem>
								);
							})}
						</SlMenu>
						<div className='property-type-selector__browser-fade' aria-hidden='true' />
					</SlDropdown>
				)}
			</div>
		</div>
	);
};

const getPropertyStructure = (type: Type | null): PropertyStructure => {
	if (type?.kind === 'array') {
		return 'array';
	}
	if (type?.kind === 'map') {
		return 'map';
	}
	if (type?.kind === 'object') {
		return 'object';
	}
	return 'one';
};

const getPropertyTypeOption = (type: Type | null): PropertyTypeOption | undefined => {
	const valueType = getPropertyValueType(type);
	const id: PropertyTypeOptionID | undefined = getTypeSemantic(valueType) ?? valueType?.kind;
	return PROPERTY_TYPE_OPTIONS.find((option) => option.id === id);
};

const wrapPropertyValueType = (type: Type, structure: PropertyStructure): Type => {
	if (structure === 'array') {
		return { kind: 'array', elementType: type };
	}
	if (structure === 'map') {
		return { kind: 'map', elementType: type };
	}
	return type;
};

export { PropertyTypeSelector };
