import React from 'react';
import './SchemaPropertyType.css';
import Type, { Semantic } from '../../../lib/api/types/types';
import { DURATION_UNIT_OPTIONS, getPropertyValueType, toKrenalisStringType } from '../../helpers/types';

type SchemaPropertyTypeContext = 'menu' | 'trigger' | 'grid' | 'details';

interface SchemaPropertyTypePresentation {
	metadata?: string;
	primary: string;
}

interface SchemaPropertyTypeProps {
	catalogOption?: boolean;
	context: SchemaPropertyTypeContext;
	description?: string;
	semantic?: Semantic;
	type: Type;
}

interface PresentationOptions {
	catalogOption?: boolean;
	description?: string;
}

const SchemaPropertyType = ({ catalogOption, context, description, semantic, type }: SchemaPropertyTypeProps) => {
	if (context === 'details') {
		const physicalType = toCompactPhysicalType(type);
		const semanticLabel = semantic == null ? null : toSemanticLabel(semantic, context);
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

	const presentation = getSchemaPropertyTypePresentation(type, semantic, context, {
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
			<span
				className={`schema-property-type__primary${
					semantic == null ? ' schema-property-type__primary--technical' : ''
				}`}
			>
				{presentation.primary}
			</span>
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
	semantic: Semantic | undefined,
	context: Exclude<SchemaPropertyTypeContext, 'details'>,
	options: PresentationOptions = {},
): SchemaPropertyTypePresentation => {
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

	const usePhysicalTypeFamily =
		context === 'trigger' ||
		(context === 'menu' && options.catalogOption && (semantic.kind === 'country' || semantic.kind === 'duration'));

	return {
		metadata: usePhysicalTypeFamily ? toPhysicalTypeFamily(type) : toProfileSchemaPhysicalType(type),
		primary: toSemanticLabel(semantic, context),
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
		return `${type.kind}(${toProfileSchemaPhysicalType(type.elementType)})`;
	}

	const normalizedType =
		type.kind === 'int' && type.unsigned && type.minimum == null ? { ...type, minimum: 0 } : type;
	return toKrenalisStringType(normalizedType, undefined, ' · ');
};

const toSemanticLabel = (semantic: Semantic, context: SchemaPropertyTypeContext): string => {
	switch (semantic.kind) {
		case 'email':
			return 'email';
		case 'phone':
			return 'phone number';
		case 'url':
			return 'URL';
		case 'country': {
			const letters = semantic.format === 'iso_3166_1_alpha_2' ? 2 : 3;
			if (context === 'details') {
				return `country (${letters}-letter)`;
			}
			if (context === 'grid') {
				return `country — ${letters}-letter ISO code`;
			}
			return 'country';
		}
		case 'datetime':
			return 'formatted date and time';
		case 'money':
			return context === 'grid' && semantic.currency != null ? `money — ${semantic.currency}` : 'money';
		case 'percentage':
			return 'percentage';
		case 'measurement':
			return context === 'grid' ? `measurement — ${semantic.unit}` : 'measurement';
		case 'duration': {
			if (context === 'grid') {
				const unit = DURATION_UNIT_OPTIONS.find((option) => option.value === semantic.unit);
				if (unit != null) {
					return `duration — ${unit.symbol}`;
				}
			}
			return 'duration';
		}
		default:
			throw new Error(`unknown semantic kind ${semantic satisfies never}`);
	}
};

export { getSchemaPropertyTypePresentation, SchemaPropertyType };
