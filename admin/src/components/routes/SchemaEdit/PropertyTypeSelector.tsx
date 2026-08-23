import React, { useEffect, useMemo, useRef, useState } from 'react';
import './PropertyTypeSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import Type, { TypeKind } from '../../../lib/api/types/types';
import { TypeIcon } from '../../base/TypeIcon/TypeIcon';

type PropertyStructure = 'one' | 'array' | 'map';

interface PropertyTypeOption {
	id: TypeKind;
	group: 'Basic values' | 'Date and time' | 'Identifiers and network' | 'Structured values';
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

const PROPERTY_TYPE_OPTIONS: PropertyTypeOption[] = [
	{
		id: 'string',
		group: 'Basic values',
		label: 'Text',
		description: 'Text without a specific meaning',
		kind: 'string',
		create: () => ({ kind: 'string' }),
	},
	{
		id: 'int',
		group: 'Basic values',
		label: 'Integer',
		description: 'Whole numeric value',
		kind: 'int',
		create: () => ({ kind: 'int', bitSize: 32, unsigned: false }),
	},
	{
		id: 'decimal',
		group: 'Basic values',
		label: 'Decimal number',
		description: 'Fixed-point numeric value',
		kind: 'decimal',
		create: () => ({ kind: 'decimal', precision: 10, scale: 0 }),
	},
	{
		id: 'float',
		group: 'Basic values',
		label: 'Floating-point number',
		description: 'Floating-point numeric value',
		kind: 'float',
		create: () => ({ kind: 'float', bitSize: 64, real: false }),
	},
	{
		id: 'boolean',
		group: 'Basic values',
		label: 'True or false',
		description: 'Boolean value',
		kind: 'boolean',
		create: () => ({ kind: 'boolean' }),
	},
	{
		id: 'datetime',
		group: 'Date and time',
		label: 'Date and time',
		description: 'Native date-time value',
		kind: 'datetime',
		create: () => ({ kind: 'datetime' }),
	},
	{
		id: 'date',
		group: 'Date and time',
		label: 'Date',
		description: 'Date without a time',
		kind: 'date',
		create: () => ({ kind: 'date' }),
	},
	{
		id: 'time',
		group: 'Date and time',
		label: 'Time',
		description: 'Time without a date',
		kind: 'time',
		create: () => ({ kind: 'time' }),
	},
	{
		id: 'year',
		group: 'Date and time',
		label: 'Year',
		description: 'Year value',
		kind: 'year',
		create: () => ({ kind: 'year' }),
	},
	{
		id: 'uuid',
		group: 'Identifiers and network',
		label: 'UUID',
		description: 'Universally unique identifier',
		kind: 'uuid',
		create: () => ({ kind: 'uuid' }),
	},
	{
		id: 'ip',
		group: 'Identifiers and network',
		label: 'IP address',
		description: 'IPv4 or IPv6 network address',
		kind: 'ip',
		create: () => ({ kind: 'ip' }),
	},
	{
		id: 'json',
		group: 'Structured values',
		label: 'JSON value',
		description: 'Structured data stored as JSON',
		kind: 'json',
		create: () => ({ kind: 'json' }),
	},
	{
		id: 'object',
		group: 'Structured values',
		label: 'Object',
		description: 'Value with nested properties',
		kind: 'object',
		create: () => ({ kind: 'object', properties: [] }),
	},
];

const PropertyTypeSelector = ({ type, canEditType, onChange }: PropertyTypeSelectorProps) => {
	const [search, setSearch] = useState('');
	const [structure, setStructure] = useState<PropertyStructure>(() => getPropertyStructure(type));
	const dropdownRef = useRef<any>();

	const valueType = getPropertyValueType(type);
	const selectedOption = getPropertyTypeOption(type);
	const options = useMemo(() => {
		const term = search.trim().toLocaleLowerCase();
		return PROPERTY_TYPE_OPTIONS.filter((option) => {
			if (structure !== 'one' && option.kind === 'object') {
				return false;
			}
			if (!canEditType && !isOptionCompatibleWithType(option, valueType)) {
				return false;
			}
			return term === '' || `${option.label} ${option.description}`.toLocaleLowerCase().includes(term);
		});
	}, [canEditType, search, structure, valueType]);

	useEffect(() => {
		if (type != null) {
			setStructure(getPropertyStructure(type));
		}
	}, [type]);

	const onChangeStructure = (event) => {
		const newStructure = event.target.value as PropertyStructure;
		setStructure(newStructure);
		const newValueType = getPropertyValueType(type);
		if (newValueType == null || (newStructure !== 'one' && newValueType.kind === 'object')) {
			onChange(null);
			return;
		}
		onChange(wrapPropertyValueType(newValueType, newStructure));
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
			<div className='property-type-selector__controls'>
				<SlSelect
					className='property-type-selector__structure'
					label='Structure'
					value={structure}
					onSlChange={onChangeStructure}
					disabled={!canEditType}
				>
					<SlOption value='one'>One value</SlOption>
					<SlOption value='array'>Array</SlOption>
					<SlOption value='map'>Map</SlOption>
				</SlSelect>
				<SlDropdown
					className='property-type-selector__dropdown'
					ref={dropdownRef}
					hoist={true}
					placement='bottom-end'
					distance={6}
					onSlAfterHide={() => setSearch('')}
				>
					<SlButton className='property-type-selector__trigger' slot='trigger' caret={true}>
						<span className='property-type-selector__trigger-content'>
							{selectedOption != null ? (
								<>
									<PropertyTypeOptionIcon option={selectedOption} />
									{selectedOption.label}
								</>
							) : (
								<span className='property-type-selector__placeholder'>Select type</span>
							)}
						</span>
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
			</div>
			<div className='property-type-selector__help'>
				Structure defines whether the property stores one value, an array of values, or a map of values. Type
				describes the stored value.
			</div>
		</div>
	);
};

const PropertyTypeOptionIcon = ({ option }: { option: PropertyTypeOption }) => {
	if (option.icon != null) {
		return <SlIcon className='property-type-selector__option-icon' name={option.icon} />;
	}
	return (
		<span className='property-type-selector__option-icon'>
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
	return 'one';
};

const getPropertyTypeLabel = (type: Type | null): string => {
	const option = getPropertyTypeOption(type);
	return option?.label || 'Select type';
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

export {
	PropertyTypeSelector,
	getPropertyStructure,
	getPropertyTypeLabel,
	getPropertyValueType,
	replacePropertyValueType,
};
