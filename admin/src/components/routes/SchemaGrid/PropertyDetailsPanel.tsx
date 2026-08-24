import React, { ReactNode } from 'react';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { toKrenalisStringType } from '../../helpers/types';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import { Property } from '../../../lib/api/types/types';
import TransformedConnection from '../../../lib/core/connection';
import { PropertyPanelLayout } from '../Schema/PropertyPanelLayout';

interface PropertyDetailsPanelProps {
	onClose: () => void;
	primarySource: TransformedConnection | null;
	property: Property;
}

interface PropertyDetailProps {
	children: ReactNode;
	className?: string;
	label: string;
}

const PropertyDetailsPanel = ({ onClose, primarySource, property }: PropertyDetailsPanelProps) => {
	return (
		<PropertyPanelLayout className='property-details-panel' closeLabel='Close property details' onClose={onClose}>
			<div className='property-details-panel__details'>
				<div className='property-details-panel__section'>
					<PropertyDetail label='Name'>{property.name}</PropertyDetail>
					<PropertyDetail label='Type' className='property-details-panel__technical-type'>
						{toKrenalisStringType(property.type)}
					</PropertyDetail>
				</div>
				<div className='property-details-panel__section property-details-panel__section--metadata'>
					<PropertyDetail label='Description' className='property-details-panel__description'>
						{property.description || <span className='property-details-panel__empty-value'>—</span>}
					</PropertyDetail>
					<PropertyDetail label='Primary source'>
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

const PropertyDetail = ({ children, className, label }: PropertyDetailProps) => {
	return (
		<div className='property-details-panel__detail'>
			<div className='property-details-panel__label'>{label}</div>
			<div className={`property-details-panel__value${className == null ? '' : ` ${className}`}`}>{children}</div>
		</div>
	);
};

export { PropertyDetailsPanel };
