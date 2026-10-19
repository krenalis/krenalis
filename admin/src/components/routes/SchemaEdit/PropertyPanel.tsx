import React from 'react';
import './PropertyPanel.css';
import SlAnimation from '@shoelace-style/shoelace/dist/react/animation/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { PrimarySources } from '../../../lib/api/types/workspace';
import { PropertyPanelLayout } from '../Schema/PropertyPanelLayout';
import { PropertyForm } from './PropertyForm';
import {
	PropertyChangeStatus,
	PropertyFieldChanges,
	PropertyParent,
	PropertyStatusBadge,
	PropertyToEdit,
} from './useSchemaEdit';

interface PropertyPanelProps {
	animateActions: boolean;
	dirty: boolean;
	fieldChanges?: PropertyFieldChanges;
	identifierPosition?: number;
	property: PropertyToEdit | null;
	parents: PropertyParent[];
	primarySources: PrimarySources;
	status?: PropertyChangeStatus;
	onClose: () => void;
	onActionsAnimationFinish: () => void;
	onDirtyChange: (dirty: boolean) => void;
	onRemove: (property: PropertyToEdit) => void;
	onSave: (property: PropertyToEdit, primarySource: string | null) => void;
}

const PropertyPanel = ({
	animateActions,
	dirty,
	fieldChanges,
	identifierPosition,
	property,
	parents,
	primarySources,
	status,
	onClose,
	onActionsAnimationFinish,
	onDirtyChange,
	onRemove,
	onSave,
}: PropertyPanelProps) => {
	const [valid, setValid] = React.useState(true);
	const isNew = property != null && property.key == null;
	const showActions = isNew || dirty;
	const formID = 'schema-edit-property-form';
	let actions: React.ReactNode = null;
	if (showActions) {
		actions = (
			<SlAnimation
				name='shake'
				duration={1000}
				playbackRate={1.2}
				iterations={1}
				play={animateActions}
				onSlFinish={onActionsAnimationFinish}
			>
				<div className='property-panel__form-actions'>
					<SlButton className='property-panel__cancel' size='small' onClick={onClose}>
						Cancel
					</SlButton>
					<SlButton
						className='property-panel__save'
						disabled={!valid}
						form={formID}
						size='small'
						type='submit'
						variant='primary'
					>
						Confirm
					</SlButton>
				</div>
			</SlAnimation>
		);
	} else if (property != null && !isNew) {
		actions = (
			<SlTooltip className='schema-edit__toolbar-tooltip' content='Delete this property from the schema' hoist>
				<SlButton
					className='property-panel__remove property-panel__delete-button schema-edit__toolbar-icon-button'
					size='small'
					aria-label='Delete this property from the schema'
					onClick={() => onRemove(property)}
				>
					<SlIcon name='trash' />
				</SlButton>
			</SlTooltip>
		);
	}

	return (
		<>
			{property == null ? (
				<aside className='property-panel property-panel--empty' />
			) : (
				<PropertyPanelLayout
					actions={actions}
					actionsAfterContent={showActions}
					title={isNew ? 'New property' : 'Property'}
					titleAdornment={status != null && <PropertyStatusBadge status={status} />}
				>
					<PropertyForm
						key={property.key ?? '__new__'}
						fieldChanges={fieldChanges}
						formID={formID}
						identifierPosition={identifierPosition}
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

export { PropertyPanel };
