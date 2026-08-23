import React from 'react';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { PrimarySources } from '../../../lib/api/types/workspace';
import { PropertyPanelLayout } from '../Schema/PropertyPanelLayout';
import { PropertyForm } from './PropertyForm';
import { PropertyChangeStatus, PropertyParent, PropertyToEdit } from './useSchemaEdit';

interface PropertyPanelProps {
	dirty: boolean;
	property: PropertyToEdit | null;
	parents: PropertyParent[];
	primarySources: PrimarySources;
	status?: PropertyChangeStatus;
	onClose: () => void;
	onDirtyChange: (dirty: boolean) => void;
	onRemove: (property: PropertyToEdit) => void;
	onSave: (property: PropertyToEdit, primarySource: string | null) => void;
}

const PropertyPanel = ({
	dirty,
	property,
	parents,
	primarySources,
	status,
	onClose,
	onDirtyChange,
	onRemove,
	onSave,
}: PropertyPanelProps) => {
	const [valid, setValid] = React.useState(true);
	const isNew = property != null && property.key == null;
	const showActions = isNew || dirty;
	const formID = 'schema-edit-property-form';

	return (
		<>
			{property == null ? (
				<aside className='property-panel property-panel--empty'>
					<div className='property-panel__empty'>
						<SlIcon name='cursor' />
						<div className='property-panel__empty-title'>Select a property</div>
						<div className='property-panel__empty-description'>
							Choose a property from the schema to view or edit its configuration.
						</div>
					</div>
				</aside>
			) : (
				<PropertyPanelLayout
					actions={
						showActions ? (
							<div className='property-panel__form-actions'>
								<SlButton className='property-panel__cancel' size='small' onClick={onClose}>
									Cancel
								</SlButton>
								<SlButton
									className='property-dialog__save'
									disabled={!valid}
									form={formID}
									size='small'
									type='submit'
									variant='primary'
								>
									Confirm
								</SlButton>
							</div>
						) : (
							!isNew && (
								<SlTooltip
									className='schema-edit__toolbar-tooltip'
									content='Delete this property from the schema'
									hoist
								>
									<SlButton
										className='property-panel__remove property-panel__delete-button schema-edit__toolbar-icon-button'
										size='small'
										aria-label='Delete this property from the schema'
										onClick={() => onRemove(property)}
									>
										<SlIcon name='trash' />
									</SlButton>
								</SlTooltip>
							)
						)
					}
					title={isNew ? 'New property' : 'Property'}
					titleAdornment={status != null && <PropertyStatusBadge status={status} />}
				>
					<PropertyForm
						formID={formID}
						propertyToEdit={property}
						primarySources={primarySources}
						parents={parents}
						showParent={isNew}
						onDirtyChange={onDirtyChange}
						onSave={onSave}
						onValidityChange={setValid}
					/>
				</PropertyPanelLayout>
			)}
		</>
	);
};

const PropertyStatusBadge = ({ status }: { status: PropertyChangeStatus }) => {
	if (status === 'added') {
		return <SlBadge variant='success'>Added</SlBadge>;
	}
	return <SlBadge variant='warning'>Modified</SlBadge>;
};

export { PropertyPanel, PropertyStatusBadge };
