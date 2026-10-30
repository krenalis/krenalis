import React from 'react';
import './PropertyPanel.css';
import WaAnimation from '@awesome.me/webawesome/dist/react/animation/index.js';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
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

	const removeButtonID = React.useId();
	const isNew = property != null && property.key == null;
	const showActions = isNew || dirty;
	const formID = 'schema-edit-property-form';
	let actions: React.ReactNode = null;
	if (showActions) {
		actions = (
			<WaAnimation
				name='shake'
				duration={1000}
				playbackRate={1.2}
				iterations={1}
				play={animateActions}
				onWaFinish={onActionsAnimationFinish}
			>
				<div className='property-panel__form-actions'>
					<WaButton appearance='outlined' className='property-panel__cancel' size='s' onClick={onClose}>
						Cancel
					</WaButton>
					<WaButton
						className='property-panel__save'
						disabled={!valid}
						// Web Awesome types form as the owner element, but its
						// setter takes the form id.
						form={formID as unknown as HTMLFormElement}
						size='s'
						type='submit'
						variant='brand'
					>
						Confirm
					</WaButton>
				</div>
			</WaAnimation>
		);
	} else if (property != null && !isNew) {
		actions = (
			<>
				<WaTooltip className='schema-edit__toolbar-tooltip' for={removeButtonID}>
					Delete this property from the schema
				</WaTooltip>
				<WaButton
					id={removeButtonID}
					appearance='outlined'
					className='property-panel__remove property-panel__delete-button schema-edit__toolbar-icon-button'
					size='s'
					aria-label='Delete this property from the schema'
					onClick={() => onRemove(property)}
				>
					<WaIcon name='trash' />
				</WaButton>
			</>
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
