import React from 'react';
import './SchemaPropertyType.css';
import Type from '../../../lib/api/types/types';
import {
	DURATION_UNIT_OPTIONS,
	getPropertyValueType,
	getTypeSemantic,
	toKrenalisStringType,
} from '../../helpers/types';

type SchemaPropertyTypeContext = 'menu' | 'trigger' | 'grid' | 'details';

interface SchemaPropertyTypePresentation {
	metadata?: string;
	primary: string;
}

interface SchemaPropertyTypeProps {
	catalogOption?: boolean;
	context: SchemaPropertyTypeContext;
	description?: string;
	type: Type;
}

interface PresentationOptions {
	catalogOption?: boolean;
	description?: string;
}

const SchemaPropertyType = ({ catalogOption, context, description, type }: SchemaPropertyTypeProps) => {
	const semantic = getTypeSemantic(getPropertyValueType(type));
	if (context === 'details') {
		const physicalType = toCompactPhysicalType(type);
		const semanticLabel = semantic == null ? null : toSemanticLabel(type, context);
		const title = semanticLabel == null ? physicalType : `${physicalType}\n${semanticLabel}`;

		return (
			<span className='schema-property-type schema-property-type--details' title={title}>
				<span className='schema-property-type__details-physical'>{physicalType}</span>
				{semanticLabel != null && (
					<span className='schema-property-type__details-semantic'>{semanticLabel}</span>
				)}
			</span>
		);
	}

	const presentation = getSchemaPropertyTypePresentation(type, context, {
		catalogOption,
		description,
	});
	const title =
		presentation.metadata == null ? presentation.primary : `${presentation.primary} · ${presentation.metadata}`;

	return (
		<span
			className={`schema-property-type schema-property-type--${context}`}
			title={context === 'menu' ? undefined : title}
		>
			<span className='schema-property-type__primary'>{presentation.primary}</span>
			{presentation.metadata != null && (
				<span
					className={`schema-property-type__metadata${
						semantic == null ? '' : ' schema-property-type__metadata--physical'
					}`}
				>
					<span className='schema-property-type__separator' aria-hidden='true'>
						{' · '}
					</span>
					<span className={semantic == null ? '' : 'schema-property-type__metadata-physical'}>
						{presentation.metadata}
					</span>
				</span>
			)}
		</span>
	);
};

const getSchemaPropertyTypePresentation = (
	type: Type,
	context: Exclude<SchemaPropertyTypeContext, 'details'>,
	options: PresentationOptions = {},
): SchemaPropertyTypePresentation => {
	const semantic = getTypeSemantic(getPropertyValueType(type));
	if (semantic == null) {
		let primary: string;
		if (context === 'trigger' || options.catalogOption) {
			primary = type.kind;
		} else {
			primary = toProfileSchemaPhysicalType(type);
		}

		return {
			metadata: context === 'menu' ? options.description : undefined,
			primary,
		};
	}

	const semanticLabel = toSemanticLabel(type, context);
	if (context === 'grid') {
		return {
			metadata: toProfileSchemaSemanticPhysicalType(type),
			primary: withPropertyStructure(type, semanticLabel),
		};
	}

	const usePhysicalTypeFamily =
		context === 'trigger' ||
		(context === 'menu' && options.catalogOption && (semantic === 'country' || semantic === 'duration'));

	return {
		metadata: usePhysicalTypeFamily ? toPhysicalTypeFamily(type) : toProfileSchemaPhysicalType(type),
		primary: semanticLabel,
	};
};

const toCompactPhysicalType = (type: Type): string => {
	switch (type.kind) {
		case 'array':
			return `array of ${toCompactPhysicalType(type.elementType)}`;
		case 'map':
			return `map of ${toCompactPhysicalType(type.elementType)}`;
		case 'int':
			return type.unsigned ? 'unsigned int' : 'int';
		default:
			return type.kind;
	}
};

const toPhysicalTypeFamily = (type: Type): string => {
	const valueType = getPropertyValueType(type);
	return valueType.kind === 'int' && valueType.unsigned ? 'unsigned int' : valueType.kind;
};

const toProfileSchemaPhysicalType = (type: Type): string => {
	if (type.kind === 'array' || type.kind === 'map') {
		return `${type.kind} of ${toProfileSchemaPhysicalType(type.elementType)}`;
	}

	const normalizedType =
		type.kind === 'int' && type.unsigned && type.minimum == null ? { ...type, minimum: 0 } : type;
	return toKrenalisStringType(normalizedType, undefined, ' · ');
};

const toProfileSchemaSemanticPhysicalType = (type: Type): string => {
	if (type.kind === 'array' || type.kind === 'map') {
		return toProfileSchemaSemanticPhysicalType(type.elementType);
	}
	return toProfileSchemaPhysicalType(type);
};

const withPropertyStructure = (type: Type, label: string): string => {
	if (type.kind === 'array' || type.kind === 'map') {
		return `${type.kind} of ${withPropertyStructure(type.elementType, label)}`;
	}
	return label;
};

const toSemanticLabel = (type: Type, context: SchemaPropertyTypeContext): string => {
	const valueType = getPropertyValueType(type);
	const semantic = getTypeSemantic(valueType);
	switch (semantic) {
		case 'email':
			return 'email';
		case 'phone':
			return 'phone number';
		case 'url':
			return 'URL';
		case 'country': {
			const letters = valueType?.kind === 'string' && valueType.format === 'alpha-2' ? 2 : 3;
			if (context === 'details') {
				return `country (${letters}-letter)`;
			}
			if (context === 'grid') {
				return `country — ${letters}-letter ISO code`;
			}
			return 'country';
		}
		case 'money':
			return context === 'grid' && valueType?.kind === 'decimal' && valueType.currency != null
				? `money — ${valueType.currency}`
				: 'money';
		case 'percentage':
			return 'percentage';
		case 'measurement':
			return context === 'grid' && valueType != null && 'unit' in valueType
				? `measurement — ${valueType.unit}`
				: 'measurement';
		case 'duration': {
			if (context === 'grid') {
				const unit = DURATION_UNIT_OPTIONS.find(
					(option) => valueType != null && 'unit' in valueType && option.value === valueType.unit,
				);
				if (unit != null) {
					return `duration — ${unit.symbol}`;
				}
			}
			return 'duration';
		}
		default:
			throw new Error(`unknown semantic ${semantic satisfies never}`);
	}
};

export { getSchemaPropertyTypePresentation, SchemaPropertyType };
