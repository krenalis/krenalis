import React, { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
import './PropertyTypeSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import Type, { DurationUnit, Semantic, TypeKind, UnitOfMeasure } from '../../../lib/api/types/types';
import { getPropertyValueType } from '../../helpers/types';
import { SchemaPropertyType } from '../Schema/SchemaPropertyType';

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

interface PropertyTypeSelection {
	semantic?: Semantic;
	type: Type;
}

interface PropertyTypeOption {
	create: () => PropertyTypeSelection;
	description?: string;
	id: PropertyTypeOptionID;
	kind: TypeKind;
	separated?: boolean;
}

interface PropertyTypeSelectorProps {
	canEditType: boolean;
	materializedSemantic?: Semantic;
	onChange: (type: Type | null, semantic?: Semantic) => void;
	semantic?: Semantic;
	type: Type | null;
}

interface PropertyTypeSelectorRef {
	focusStructureTrigger: () => void;
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
		id: 'string',
		kind: 'string',
		description: 'Text value, such as a name or code',
		create: () => ({ type: { kind: 'string' } }),
	},
	{
		id: 'email',
		kind: 'string',
		create: () => ({ type: { kind: 'string' }, semantic: { kind: 'email' } }),
	},
	{
		id: 'phone',
		kind: 'string',
		create: () => ({ type: { kind: 'string' }, semantic: { kind: 'phone' } }),
	},
	{
		id: 'url',
		kind: 'string',
		create: () => ({ type: { kind: 'string' }, semantic: { kind: 'url' } }),
	},
	{
		id: 'country',
		kind: 'string',
		create: () => ({
			type: { kind: 'string', maxLength: 2 },
			semantic: { kind: 'country', format: 'iso_3166_1_alpha_2' },
		}),
	},
	{
		id: 'boolean',
		kind: 'boolean',
		description: 'True or false',
		separated: true,
		create: () => ({ type: { kind: 'boolean' } }),
	},
	{
		id: 'int',
		kind: 'int',
		description: 'Number with no decimal places',
		separated: true,
		create: () => ({ type: { kind: 'int', bitSize: 32, unsigned: false } }),
	},
	{
		id: 'duration',
		kind: 'int',
		create: () => ({
			type: { kind: 'int', bitSize: 64, unsigned: false },
			semantic: { kind: 'duration', unit: EMPTY_DURATION_UNIT },
		}),
	},
	{
		id: 'float',
		kind: 'float',
		description: 'Number with approximate precision',
		separated: true,
		create: () => ({ type: { kind: 'float', bitSize: 64, real: false } }),
	},
	{
		id: 'decimal',
		kind: 'decimal',
		description: 'Decimal number with fixed precision',
		separated: true,
		create: () => ({ type: { kind: 'decimal', precision: 10, scale: 0 } }),
	},
	{
		id: 'money',
		kind: 'decimal',
		create: () => ({ type: { ...PROFILE_SEMANTIC_DECIMAL_TYPE }, semantic: { kind: 'money' } }),
	},
	{
		id: 'percentage',
		kind: 'decimal',
		create: () => ({
			type: { ...PROFILE_SEMANTIC_DECIMAL_TYPE },
			semantic: { kind: 'percentage' },
		}),
	},
	{
		id: 'measurement',
		kind: 'decimal',
		create: () => ({
			type: { ...PROFILE_SEMANTIC_DECIMAL_TYPE },
			semantic: { kind: 'measurement', unit: EMPTY_UNIT_OF_MEASURE },
		}),
	},
	{
		id: 'datetime',
		kind: 'datetime',
		description: 'Date and time',
		separated: true,
		create: () => ({ type: { kind: 'datetime' } }),
	},
	{
		id: 'date',
		kind: 'date',
		description: 'Date without a time',
		create: () => ({ type: { kind: 'date' } }),
	},
	{
		id: 'time',
		kind: 'time',
		description: 'Time without a date',
		create: () => ({ type: { kind: 'time' } }),
	},
	{
		id: 'year',
		kind: 'year',
		description: 'Year',
		create: () => ({ type: { kind: 'year' } }),
	},
	{
		id: 'uuid',
		kind: 'uuid',
		description: 'UUID',
		separated: true,
		create: () => ({ type: { kind: 'uuid' } }),
	},
	{
		id: 'json',
		kind: 'json',
		description: 'JSON value',
		create: () => ({ type: { kind: 'json' } }),
	},
	{
		id: 'ip',
		kind: 'ip',
		description: 'IPv4 or IPv6 address',
		create: () => ({ type: { kind: 'ip' } }),
	},
];

const PropertyTypeSelector = forwardRef<PropertyTypeSelectorRef, PropertyTypeSelectorProps>(
	({ type, semantic, canEditType, materializedSemantic, onChange }, ref) => {
		const [structure, setStructure] = useState<PropertyStructure>(() => getPropertyStructure(type));
		const structureDropdownRef = useRef<any>();
		const dropdownRef = useRef<any>();
		const typeMenuRef = useRef<any>();
		const focusTypeAfterStructureSelectionRef = useRef(false);

		useImperativeHandle(
			ref,
			() => ({
				focusStructureTrigger: () => {
					if (canEditType) {
						structureDropdownRef.current?.focusOnTrigger();
					}
				},
			}),
			[canEditType],
		);

		const valueType = getPropertyValueType(type);
		const selectedOption = getPropertyTypeOption(type, semantic);
		const materializedOption = getPropertyTypeOption(type, materializedSemantic);
		const baseOption = valueType == null ? undefined : getPropertyTypeOption(valueType);
		const selectedStructureOption =
			PROPERTY_STRUCTURE_OPTIONS.find((option) => option.id === structure) || PROPERTY_STRUCTURE_OPTIONS[0];
		const showValueTypeSelector = structure !== 'object';
		let typeOptions = PROPERTY_TYPE_OPTIONS;
		if (!canEditType) {
			typeOptions = [];
			if (materializedSemantic != null && materializedOption != null && baseOption != null) {
				typeOptions = semantic == null ? [baseOption, materializedOption] : [materializedOption, baseOption];
			}
		}
		const hasMaterializedSemanticTransition = !canEditType && typeOptions.length === 2;
		const canOnlyChangeBackToBaseType = canEditType ? semantic != null : hasMaterializedSemanticTransition;
		const showTypeDropdown = canEditType || hasMaterializedSemanticTransition;
		let appliedTypeNote: string | null = null;
		if (type != null) {
			appliedTypeNote =
				canOnlyChangeBackToBaseType && valueType != null
					? `This type can only be changed back to ${valueType.kind} once the property has been applied.`
					: "Type can't be changed once the property has been applied.";
		}

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
			onChange(wrapPropertyValueType(valueType, nextStructure), semantic);
		};

		const onStructureMenuAfterHide = () => {
			if (!focusTypeAfterStructureSelectionRef.current) {
				return;
			}
			focusTypeAfterStructureSelectionRef.current = false;
			dropdownRef.current?.focusOnTrigger();
		};

		const onTypeMenuAfterShow = () => {
			if (valueType != null) {
				return;
			}
			const firstOption = typeMenuRef.current?.getAllItems()[0];
			if (firstOption == null) {
				return;
			}
			typeMenuRef.current.setCurrentItem(firstOption);
			firstOption.focus();
		};

		const onSelectOption = (event) => {
			const option = PROPERTY_TYPE_OPTIONS.find((candidate) => candidate.id === event.detail.item.value);
			if (
				option == null ||
				option.id === selectedOption?.id ||
				!typeOptions.some((candidate) => candidate.id === option.id)
			) {
				return;
			}
			const selection = option.create();
			const nextSemantic =
				!canEditType && option.id === materializedOption?.id
					? structuredClone(materializedSemantic)
					: selection.semantic;
			let nextValueType = valueType;
			if (canEditType && (selection.semantic != null || valueType?.kind !== option.kind)) {
				nextValueType = selection.type;
			}
			if (nextValueType == null) {
				return;
			}
			onChange(wrapPropertyValueType(nextValueType, structure), nextSemantic);
		};

		return (
			<div className='property-type-selector'>
				<div
					className={`property-type-selector__controls${
						showValueTypeSelector ? '' : ' property-type-selector__controls--structure-only'
					}${showTypeDropdown ? '' : ' property-type-selector__controls--read-only'}`}
				>
					{canEditType ? (
						<SlDropdown
							className='property-type-selector__structure-dropdown'
							ref={structureDropdownRef}
							hoist
							placement='bottom-start'
							distance={6}
							onSlAfterHide={onStructureMenuAfterHide}
						>
							<SlButton
								className='property-type-selector__structure-trigger'
								slot='trigger'
								caret
								aria-label={`Structure: ${selectedStructureOption.label}`}
							>
								<SlIcon slot='prefix' name={selectedStructureOption.icon} />
								{selectedStructureOption.triggerLabel}
							</SlButton>
							<SlMenu className='property-type-selector__structure-menu' onSlSelect={onSelectStructure}>
								{PROPERTY_STRUCTURE_OPTIONS.map((option) => (
									<SlMenuItem
										className={`property-type-selector__structure-option${
											structure === option.id
												? ' property-type-selector__structure-option--selected'
												: ''
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
					) : (
						<div
							className='property-type-selector__structure-value'
							aria-label={`Structure: ${selectedStructureOption.label}`}
						>
							<SlIcon name={selectedStructureOption.icon} />
							<span>{selectedStructureOption.triggerLabel}</span>
						</div>
					)}
					{showValueTypeSelector &&
						(showTypeDropdown ? (
							<SlDropdown
								className='property-type-selector__dropdown'
								ref={dropdownRef}
								hoist
								placement='bottom-end'
								distance={6}
								onSlAfterShow={onTypeMenuAfterShow}
							>
								<SlButton
									className='property-type-selector__trigger'
									slot='trigger'
									caret
									aria-label={valueType == null ? 'Select type' : undefined}
								>
									{valueType == null ? (
										<span className='property-type-selector__placeholder'>Select type...</span>
									) : (
										<SchemaPropertyType
											context={canEditType ? 'trigger' : 'menu'}
											type={valueType}
											semantic={semantic}
										/>
									)}
								</SlButton>
								<SlMenu
									className='property-type-selector__browser'
									ref={typeMenuRef}
									onSlSelect={onSelectOption}
								>
									{typeOptions.map((option) => {
										const selection = option.create();
										const optionType =
											!canEditType && valueType != null ? valueType : selection.type;
										const optionSemantic =
											!canEditType && option.id === materializedOption?.id
												? materializedSemantic
												: selection.semantic;
										return (
											<SlMenuItem
												className={`property-type-selector__option${
													canEditType && option.separated
														? ' property-type-selector__option--separated'
														: ''
												}${
													selectedOption?.id === option.id
														? ' property-type-selector__option--selected'
														: ''
												}`}
												key={option.id}
												data-type-option={option.id}
												value={option.id}
											>
												<SchemaPropertyType
													context='menu'
													type={optionType}
													semantic={optionSemantic}
													description={option.description}
													catalogOption={canEditType}
												/>
												{selectedOption?.id === option.id && (
													<SlIcon slot='suffix' name='check-lg' />
												)}
											</SlMenuItem>
										);
									})}
								</SlMenu>
							</SlDropdown>
						) : (
							<div className='property-type-selector__type-value'>
								{valueType != null && (
									<SchemaPropertyType context='menu' type={valueType} semantic={semantic} />
								)}
							</div>
						))}
				</div>
				{appliedTypeNote != null && (
					<div className='property-type-selector__applied-note'>{appliedTypeNote}</div>
				)}
			</div>
		);
	},
);

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

const getPropertyTypeOption = (type: Type | null, semantic?: Semantic): PropertyTypeOption | undefined => {
	if (semantic?.kind === 'datetime') {
		return undefined;
	}
	const id: PropertyTypeOptionID | undefined = semantic == null ? getPropertyValueType(type)?.kind : semantic.kind;
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

export { PropertyTypeSelector, type PropertyTypeSelectorRef };
