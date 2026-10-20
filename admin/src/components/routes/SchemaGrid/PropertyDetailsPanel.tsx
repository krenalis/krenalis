import React, { ReactNode } from 'react';
import './PropertyDetailsPanel.css';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import {
	DURATION_UNIT_OPTIONS,
	getPropertyValueType,
	getTypeSemantics,
	isSuitableAsIdentifier,
	toKrenalisStringType,
	toSemanticsLabel,
	UNIT_OF_MEASURE_OPTIONS,
} from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import { Property } from '../../../lib/api/types/types';
import TransformedConnection from '../../../lib/core/connection';
import { PropertyPanelLayout } from '../Schema/PropertyPanelLayout';
import {
	SchemaPropertyIdentifierLabel,
	SchemaPropertyIdentifierValue,
	SchemaPropertyPrimarySourceLabel,
} from '../Schema/SchemaPropertyGrid';

interface PropertyDetailsPanelProps {
	identifierPosition?: number;
	onClose: () => void;
	primarySource: TransformedConnection | null;
	property: Property;
}

interface PropertyDetailProps {
	children: ReactNode;
	className?: string;
	label: ReactNode;
}

const PropertyDetailsPanel = ({ identifierPosition, onClose, primarySource, property }: PropertyDetailsPanelProps) => {
	const semanticsDetail = getSemanticsDetail(property);
	const physicalType = toKrenalisStringType(property.type);
	const valueType = getPropertyValueType(property.type);
	const semantics = getTypeSemantics(valueType);
	let semanticsLabel = semantics == null ? null : toSemanticsLabel(semantics);
	if (semantics === 'country') {
		const letters = valueType.kind === 'string' && valueType.format === 'alpha-2' ? 2 : 3;
		semanticsLabel += ` (${letters}-letter)`;
	}

	return (
		<PropertyPanelLayout className='property-details-panel' closeLabel='Close property details' onClose={onClose}>
			<div className='property-details-panel__details'>
				<div className='property-details-panel__section'>
					<PropertyDetail label='Name'>{property.name}</PropertyDetail>
					<PropertyDetail label='Type'>
						<span
							className='property-details-panel__type'
							title={semanticsLabel == null ? physicalType : `${physicalType}\n${semanticsLabel}`}
						>
							<span className='property-details-panel__physical-type'>{physicalType}</span>
							{semanticsLabel != null && (
								<span className='property-details-panel__semantics'>{semanticsLabel}</span>
							)}
						</span>
					</PropertyDetail>
					{semanticsDetail != null && (
						<PropertyDetail label={semanticsDetail.label}>
							{semanticsDetail.value != null && semanticsDetail.value !== '' ? (
								semanticsDetail.value
							) : (
								<span className='property-details-panel__empty-value'>—</span>
							)}
						</PropertyDetail>
					)}
				</div>
				<div className='property-details-panel__section property-details-panel__section--metadata'>
					<PropertyDetail label='Display name'>
						{property.displayName || <span className='property-details-panel__empty-value'>—</span>}
					</PropertyDetail>
					<PropertyDetail label='Description' className='property-details-panel__description'>
						{property.description || <span className='property-details-panel__empty-value'>—</span>}
					</PropertyDetail>
					{isSuitableAsIdentifier(property.type) && (
						<PropertyDetail label={<SchemaPropertyIdentifierLabel />}>
							<SchemaPropertyIdentifierValue position={identifierPosition} />
						</PropertyDetail>
					)}
					<PropertyDetail
						label={<SchemaPropertyPrimarySourceLabel primarySourceName={primarySource?.name ?? null} />}
					>
						{primarySource == null ? (
							<span className='property-details-panel__empty-value'>—</span>
						) : (
							<span className='property-details-panel__primary-source'>
								<LittleLogo code={primarySource.connector.code} path={CONNECTORS_ASSETS_PATH} />
								{primarySource.name}
							</span>
						)}
					</PropertyDetail>
				</div>
			</div>
		</PropertyPanelLayout>
	);
};

const getSemanticsDetail = (property: Property): { label: string; value?: string } | null => {
	const type = getPropertyValueType(property.type);
	if (type?.kind === 'decimal' && type.semantics === 'money') {
		return { label: 'Currency', value: type.currency };
	}
	if (type != null && 'semantics' in type && type.semantics === 'measurement') {
		const option = UNIT_OF_MEASURE_OPTIONS.find((candidate) => candidate.value === type.unit);
		return { label: 'Unit', value: option == null ? undefined : `${option.label} · ${option.value}` };
	}
	if (type != null && 'semantics' in type && type.semantics === 'duration') {
		const option = DURATION_UNIT_OPTIONS.find((candidate) => candidate.value === type.unit);
		return { label: 'Unit', value: option == null ? undefined : `${option.label} · ${option.symbol}` };
	}
	return null;
};

const PropertyDetail = ({ children, className, label }: PropertyDetailProps) => {
	return (
		<div className='property-details-panel__detail'>
			<div className='property-details-panel__label'>{label}</div>
			<div className={`property-details-panel__value${className == null ? '' : ` ${className}`}`}>{children}</div>
		</div>
	);
};

export { PropertyDetailsPanel };
