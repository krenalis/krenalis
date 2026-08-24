import React, { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import './PropertyTypeSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import Type, { TypeKind } from '../../../lib/api/types/types';
import { TypeIcon } from '../../base/TypeIcon/TypeIcon';

type PropertyStructure = 'one' | 'array' | 'object' | 'map';

interface PropertyStructureOption {
	id: PropertyStructure;
	label: string;
	triggerLabel: string;
	description: string;
	icon: string;
}

interface PropertyTypeOption {
	id: TypeKind;
	group: 'Basic values' | 'Date and time' | 'Specialized values';
	label: string;
	description: string;
	kind: TypeKind;
	icon?: string;
	create: () => Type;
}

interface PropertyTypeSelectorProps {
	type: Type | null;
	canEditType: boolean;
	onChange: (type: Type | null) => void;
}

interface PropertyTypeSelectorRef {
	openStructureMenu: () => void;
}

const PROPERTY_STRUCTURE_OPTIONS: PropertyStructureOption[] = [
	{
		id: 'one',
		label: 'one value',
		triggerLabel: 'one value',
		description: 'The property contains a single value',
		icon: '1-circle',
	},
	{
		id: 'array',
		label: 'array',
		triggerLabel: 'array of',
		description: 'The property contains an ordered collection of values',
		icon: 'list-ul',
	},
	{
		id: 'object',
		label: 'object',
		triggerLabel: 'object',
		description: 'The property contains nested properties',
		icon: 'braces',
	},
	{
		id: 'map',
		label: 'map',
		triggerLabel: 'map of',
		description: 'The property contains key-value pairs with string keys',
		icon: 'braces-asterisk',
	},
];

const PROPERTY_TYPE_OPTIONS: PropertyTypeOption[] = [
	{
		id: 'string',
		group: 'Basic values',
		label: 'string',
		description: 'Text without a specific meaning',
		kind: 'string',
		create: () => ({ kind: 'string' }),
	},
	{
		id: 'int',
		group: 'Basic values',
		label: 'int',
		description: 'Number without decimal places',
		kind: 'int',
		create: () => ({ kind: 'int', bitSize: 32, unsigned: false }),
	},
	{
		id: 'decimal',
		group: 'Basic values',
		label: 'decimal',
		description: 'Fixed-point numeric value',
		kind: 'decimal',
		create: () => ({ kind: 'decimal', precision: 10, scale: 0 }),
	},
	{
		id: 'float',
		group: 'Basic values',
		label: 'float',
		description: 'Floating-point numeric value',
		kind: 'float',
		create: () => ({ kind: 'float', bitSize: 64, real: false }),
	},
	{
		id: 'boolean',
		group: 'Basic values',
		label: 'boolean',
		description: 'True or false value',
		kind: 'boolean',
		create: () => ({ kind: 'boolean' }),
	},
	{
		id: 'datetime',
		group: 'Date and time',
		label: 'datetime',
		description: 'Native date-time value',
		kind: 'datetime',
		create: () => ({ kind: 'datetime' }),
	},
	{
		id: 'date',
		group: 'Date and time',
		label: 'date',
		description: 'Date without a time',
		kind: 'date',
		create: () => ({ kind: 'date' }),
	},
	{
		id: 'time',
		group: 'Date and time',
		label: 'time',
		description: 'Time without a date',
		kind: 'time',
		create: () => ({ kind: 'time' }),
	},
	{
		id: 'year',
		group: 'Date and time',
		label: 'year',
		description: 'Year value',
		kind: 'year',
		create: () => ({ kind: 'year' }),
	},
	{
		id: 'uuid',
		group: 'Specialized values',
		label: 'uuid',
		description: 'Universally unique identifier',
		kind: 'uuid',
		create: () => ({ kind: 'uuid' }),
	},
	{
		id: 'json',
		group: 'Specialized values',
		label: 'json',
		description: 'Structured data stored as JSON',
		kind: 'json',
		create: () => ({ kind: 'json' }),
	},
	{
		id: 'ip',
		group: 'Specialized values',
		label: 'ip',
		description: 'IPv4 or IPv6 network address',
		kind: 'ip',
		create: () => ({ kind: 'ip' }),
	},
];

const PropertyTypeSelector = forwardRef<PropertyTypeSelectorRef, PropertyTypeSelectorProps>(
	({ type, canEditType, onChange }, ref) => {
		const [search, setSearch] = useState('');
		const [structure, setStructure] = useState<PropertyStructure>(() => getPropertyStructure(type));
		const structureDropdownRef = useRef<any>();
		const dropdownRef = useRef<any>();
		const openTypeAfterStructureSelectionRef = useRef(false);

		useImperativeHandle(
			ref,
			() => ({
				openStructureMenu: () => {
					if (!canEditType) {
						return;
					}
					structureDropdownRef.current?.focusOnTrigger();
					void structureDropdownRef.current?.show();
				},
			}),
			[canEditType],
		);

		const valueType = getPropertyValueType(type);
		const selectedOption = getPropertyTypeOption(type);
		const selectedStructureOption =
			PROPERTY_STRUCTURE_OPTIONS.find((option) => option.id === structure) || PROPERTY_STRUCTURE_OPTIONS[0];
		const showValueTypeSelector = structure !== 'object';
		const options = useMemo(() => {
			const term = search.trim().toLocaleLowerCase();
			return PROPERTY_TYPE_OPTIONS.filter((option) => {
				if (!canEditType && !isOptionCompatibleWithType(option, valueType)) {
					return false;
				}
				return term === '' || `${option.label} ${option.description}`.toLocaleLowerCase().includes(term);
			});
		}, [canEditType, search, valueType]);

		useEffect(() => {
			if (type != null) {
				setStructure(getPropertyStructure(type));
			}
		}, [type]);

		const onSelectStructure = (event) => {
			if (!canEditType) {
				return;
			}
			const newStructure = event.detail.item.value as PropertyStructure;
			openTypeAfterStructureSelectionRef.current = newStructure !== 'object';
			if (newStructure === structure) {
				return;
			}
			setStructure(newStructure);
			if (newStructure === 'object') {
				onChange({ kind: 'object', properties: [] });
				return;
			}
			const newValueType = getPropertyValueType(type);
			if (newValueType == null || newValueType.kind === 'object') {
				onChange(null);
				return;
			}
			onChange(wrapPropertyValueType(newValueType, newStructure));
		};

		const onStructureMenuAfterHide = () => {
			if (!openTypeAfterStructureSelectionRef.current) {
				return;
			}
			openTypeAfterStructureSelectionRef.current = false;
			dropdownRef.current?.focusOnTrigger();
			void dropdownRef.current?.show();
		};

		const onSelectOption = (option: PropertyTypeOption) => {
			const newType = canEditType ? wrapPropertyValueType(option.create(), structure) : type;
			onChange(newType);
			dropdownRef.current?.hide();
		};

		const groupedOptions = options.reduce<Record<string, PropertyTypeOption[]>>((groups, option) => {
			if (groups[option.group] == null) {
				groups[option.group] = [];
			}
			groups[option.group].push(option);
			return groups;
		}, {});

		return (
			<div className='property-type-selector'>
				<SlTooltip
					className='schema-edit__toolbar-tooltip'
					content='The type of an existing property cannot be changed.'
					disabled={canEditType}
					hoist
				>
					<div
						className={`property-type-selector__controls${
							canEditType ? '' : ' property-type-selector__controls--read-only'
						}${showValueTypeSelector ? '' : ' property-type-selector__controls--structure-only'}`}
						onPointerDown={(event) => {
							if (!canEditType) {
								event.preventDefault();
							}
						}}
					>
						<SlDropdown
							className='property-type-selector__structure-dropdown'
							ref={structureDropdownRef}
							hoist={true}
							placement='bottom-start'
							distance={6}
							disabled={!canEditType}
							onSlAfterHide={onStructureMenuAfterHide}
						>
							<SlButton
								className='property-type-selector__structure-trigger'
								slot='trigger'
								caret={canEditType}
								aria-disabled={!canEditType || undefined}
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
								hoist={true}
								placement='bottom-end'
								distance={6}
								disabled={!canEditType}
								onSlAfterHide={() => setSearch('')}
							>
								<SlButton
									className='property-type-selector__trigger'
									slot='trigger'
									caret={canEditType}
									aria-disabled={!canEditType || undefined}
									aria-label={
										selectedOption != null ? `Type: ${selectedOption.label}` : 'Select type'
									}
								>
									{selectedOption != null && (
										<PropertyTypeOptionIcon option={selectedOption} slot='prefix' />
									)}
									{selectedOption != null ? (
										selectedOption.label
									) : (
										<span className='property-type-selector__placeholder'>Select type</span>
									)}
								</SlButton>
								<div className='property-type-selector__browser'>
									<SlInput
										className='property-type-selector__search'
										placeholder='Search types...'
										value={search}
										onSlInput={(event: any) => setSearch(event.target.value)}
									>
										<SlIcon slot='prefix' name='search' />
									</SlInput>
									<div className='property-type-selector__options'>
										{Object.entries(groupedOptions).map(([group, groupOptions]) => (
											<div className='property-type-selector__group' key={group}>
												<div className='property-type-selector__group-label'>{group}</div>
												{groupOptions.map((option) => (
													<button
														className={`property-type-selector__option${
															selectedOption?.id === option.id
																? ' property-type-selector__option--selected'
																: ''
														}`}
														key={option.id}
														type='button'
														data-type-option={option.id}
														onClick={() => onSelectOption(option)}
													>
														<PropertyTypeOptionIcon option={option} />
														<span className='property-type-selector__option-content'>
															<span className='property-type-selector__option-label'>
																{option.label}
															</span>
															<span className='property-type-selector__option-description'>
																{option.description}
															</span>
														</span>
														{selectedOption?.id === option.id && <SlIcon name='check-lg' />}
													</button>
												))}
											</div>
										))}
										{options.length === 0 && (
											<div className='property-type-selector__no-options'>No matching types</div>
										)}
									</div>
								</div>
							</SlDropdown>
						)}
					</div>
				</SlTooltip>
			</div>
		);
	},
);

const PropertyTypeOptionIcon = ({ option, slot }: { option: PropertyTypeOption; slot?: string }) => {
	if (option.icon != null) {
		return <SlIcon className='property-type-selector__option-icon' name={option.icon} slot={slot} />;
	}
	return (
		<span className='property-type-selector__option-icon' slot={slot}>
			<TypeIcon kind={option.kind} />
		</span>
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
	const id = getPropertyValueType(type)?.kind;
	return PROPERTY_TYPE_OPTIONS.find((option) => option.id === id);
};

const getPropertyValueType = (type: Type | null): Type | null => {
	if (type?.kind === 'array' || type?.kind === 'map') {
		return type.elementType;
	}
	return type;
};

const isOptionCompatibleWithType = (option: PropertyTypeOption, type: Type | null): boolean => {
	if (type == null || option.kind !== type.kind) {
		return false;
	}
	return true;
};

const replacePropertyValueType = (type: Type, valueType: Type): Type => {
	if (type.kind === 'array') {
		return { ...type, elementType: valueType };
	}
	if (type.kind === 'map') {
		return { ...type, elementType: valueType };
	}
	return valueType;
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

export { PropertyTypeSelector, getPropertyValueType, replacePropertyValueType };
export type { PropertyTypeSelectorRef };
