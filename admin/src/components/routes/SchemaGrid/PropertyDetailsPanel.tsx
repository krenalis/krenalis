import React, { ReactNode } from 'react';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { getCompatibleProfilePropertyRoles, getProfilePropertyRole } from '../../helpers/profilePropertyRoles';
import {
	DURATION_UNIT_OPTIONS,
	getPropertyValueType,
	isSuitableAsIdentifier,
	UNIT_OF_MEASURE_OPTIONS,
} from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import { Property } from '../../../lib/api/types/types';
import { ProfilePropertyRoleID } from '../../../lib/api/types/workspace';
import TransformedConnection from '../../../lib/core/connection';
import { PropertyPanelLayout } from '../Schema/PropertyPanelLayout';
import {
	SchemaPropertyIdentifierLabel,
	SchemaPropertyIdentifierValue,
	SchemaPropertyInfoTooltip,
	SchemaPropertyPrimarySourceLabel,
} from '../Schema/SchemaPropertyGrid';
import { SchemaPropertyType } from '../Schema/SchemaPropertyType';

interface PropertyDetailsPanelProps {
	propertyRole: ProfilePropertyRoleID | null;
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

const PropertyDetailsPanel = ({
	propertyRole,
	identifierPosition,
	onClose,
	primarySource,
	property,
}: PropertyDetailsPanelProps) => {
	const semanticDetail = getSemanticDetail(property);

	return (
		<PropertyPanelLayout className='property-details-panel' closeLabel='Close property details' onClose={onClose}>
			<div className='property-details-panel__details'>
				<div className='property-details-panel__section'>
					<PropertyDetail label='Name'>{property.name}</PropertyDetail>
					<PropertyDetail label='Type'>
						<SchemaPropertyType context='details' type={property.type} />
					</PropertyDetail>
					{semanticDetail != null && (
						<PropertyDetail label={semanticDetail.label}>
							{semanticDetail.value || <span className='property-details-panel__empty-value'>—</span>}
						</PropertyDetail>
					)}
				</div>
				<div className='property-details-panel__section property-details-panel__section--metadata'>
					{getCompatibleProfilePropertyRoles(property).length > 0 && (
						<PropertyDetail
							label={
								<span className='schema-property-grid__label-content'>
									Role
									<SchemaPropertyInfoTooltip
										content='Specifies which profile concept this property represents.'
										label='About roles'
									/>
								</span>
							}
						>
							{propertyRole == null ? (
								<span className='property-details-panel__empty-value'>—</span>
							) : (
								getProfilePropertyRole(propertyRole).label
							)}
						</PropertyDetail>
					)}
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

const getSemanticDetail = (property: Property): { label: string; value?: string } | null => {
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
