import React, { useContext, useState } from 'react';
import './SchemaEdit.css';
import { SchemaContext } from '../../../context/SchemaContext';
import { PropertyToRemove, PropertyToEdit, useSchemaEdit } from './useSchemaEdit';
import { PropertyDialog } from './PropertyDialog';
import SortableGrid from '../../base/Grid/SortableGrid';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { EditableProperty, newPropertyToEdit } from './SchemaEdit.helpers';
import { TypeKind } from '../../../lib/api/types/types';
import { FullscreenContext } from '../../../context/FullscreenContext';
import SyntaxHighlight from '../../base/SyntaxHighlight/SyntaxHighlight';

const SchemaEdit = () => {
	const [propertyToEdit, setPropertyToEdit] = useState<PropertyToEdit | null>(null);
	const [propertyToRemove, setPropertyToRemove] = useState<PropertyToRemove | null>(null);

	const { schema } = useContext(SchemaContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const onAddClick = (parentKey: string, indentation: number, root: string) => {
		const p = newPropertyToEdit(parentKey, indentation, root);
		setPropertyToEdit(p);
	};

	const onEditClick = (propertyKey: string, property: EditableProperty) => {
		setPropertyToEdit({ key: propertyKey, ...property });
	};

	const onRemoveClick = (propertyKey: string, propertyName: string, typeKind: TypeKind) => {
		setPropertyToRemove({ key: propertyKey, name: propertyName, type: typeKind });
	};

	const {
		rows,
		columns,
		primarySources,
		queries,
		isQueriesLoading,
		isConfirmChangesLoading,
		onAddProperty,
		onEditProperty,
		onRemoveProperty,
		onSortRow,
		onApplyChanges,
		onConfirmChanges,
		onCancelChanges,
		sortableGridRef,
	} = useSchemaEdit(schema, onAddClick, onEditClick, onRemoveClick, closeFullscreen);

	const onCancelRemove = () => {
		setPropertyToRemove(null);
	};

	const onCancelEdit = () => {
		closeFullscreen();
	};

	return (
		<div className='schema-edit'>
			<div className='schema-edit__header'>
				<div className='schema-edit__header-title'>Alter schema</div>
				<div className='schema-edit__header-buttons'>
					<SlButton className='schema-edit__header-cancel-button' onClick={onCancelEdit}>
						Cancel
					</SlButton>
					<SlButton className='schema-edit__header-apply-button' variant='primary' onClick={onApplyChanges}>
						Review and apply changes...
					</SlButton>
				</div>
			</div>
			<div className='schema-edit__content'>
				<SortableGrid rows={rows} columns={columns} onSortRow={onSortRow} ref={sortableGridRef} />
				<SlButton
					variant='primary'
					outline={true}
					className='schema-edit__add-property'
					onClick={() => onAddClick('', 0, '')}
				>
					<SlIcon name='plus-circle' slot='suffix' />
					Add property
				</SlButton>
			</div>
			<PropertyDialog
				propertyToEdit={propertyToEdit}
				setPropertyToEdit={setPropertyToEdit}
				primarySources={primarySources}
				onAddProperty={onAddProperty}
				onEditProperty={onEditProperty}
			/>
			<SlDialog
				open={isQueriesLoading || queries != null}
				label='Review changes'
				onSlAfterHide={onCancelChanges}
				className={`schema-edit__queries ${isQueriesLoading ? ' schema-edit__queries--loading' : ''}`}
			>
				{isQueriesLoading ? (
					<SlSpinner
						style={
							{
								margin: '30px auto 50px auto',
								fontSize: '40px',
								'--track-width': '5px',
							} as React.CSSProperties
						}
					></SlSpinner>
				) : (
					queries != null && (
						<>
							{queries.length > 0 ? (
								<div className='schema-edit__queries-preview'>
									<SyntaxHighlight language='sql'>{queries.join('\n\n')}</SyntaxHighlight>
								</div>
							) : (
								<div className='schema-edit__no-query'>No query for this operation</div>
							)}
							<div className='schema-edit__queries-buttons'>
								<SlButton size='small' onClick={onCancelChanges}>
									Cancel
								</SlButton>
								<SlButton
									className='schema-edit__apply-alter-button'
									size='small'
									variant='danger'
									onClick={onConfirmChanges}
									loading={isConfirmChangesLoading}
								>
									Apply alter schema
								</SlButton>
							</div>
						</>
					)
				)}
			</SlDialog>
			<AlertDialog
				variant='danger'
				isOpen={propertyToRemove != null}
				onClose={onCancelRemove}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={onCancelRemove}>Cancel</SlButton>
						<SlButton
							variant='danger'
							className='schema-edit__confirm-remove-property'
							onClick={() => {
								onRemoveProperty(propertyToRemove.key);
								setPropertyToRemove(null);
							}}
						>
							Remove
						</SlButton>
					</>
				}
			>
				<p>
					The property <b>"{propertyToRemove?.name}"</b> will be deleted when you apply the changes.
					{propertyToRemove?.type === 'object'
						? ' Note that, since the property is an object, all its children properties will be deleted as well.'
						: ''}
				</p>
			</AlertDialog>
		</div>
	);
};

export { SchemaEdit };
