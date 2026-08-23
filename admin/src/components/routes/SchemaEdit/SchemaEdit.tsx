import React, { useContext, useRef, useState } from 'react';
import './SchemaEdit.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SortableGrid from '../../base/Grid/SortableGrid';
import SyntaxHighlight from '../../base/SyntaxHighlight/SyntaxHighlight';
import { FullscreenContext } from '../../../context/FullscreenContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { TypeKind } from '../../../lib/api/types/types';
import { EditableProperty, newPropertyToEdit } from './SchemaEdit.helpers';
import { PropertyPanel } from './PropertyPanel';
import { PropertyToEdit, PropertyToRemove, useSchemaEdit } from './useSchemaEdit';

const schemaEditGridColumns =
	'minmax(160px, 0.65fr) minmax(210px, 0.85fr) minmax(240px, 1.5fr) minmax(160px, 0.65fr) 90px';
const schemaEditNestedRowsIndentation = { base: 34, step: 20 };

interface SchemaEditProps {
	initialPropertyKey?: string | null;
}

const SchemaEdit = ({ initialPropertyKey }: SchemaEditProps) => {
	const [propertyToEdit, setPropertyToEdit] = useState<PropertyToEdit | null>(null);
	const [propertyToRemove, setPropertyToRemove] = useState<PropertyToRemove | null>(null);
	const [pendingPropertyToEdit, setPendingPropertyToEdit] = useState<PropertyToEdit | null>(null);
	const [propertyDraftDirty, setPropertyDraftDirty] = useState(false);
	const [isSearchOpen, setIsSearchOpen] = useState(false);
	const [search, setSearch] = useState('');
	const [showOnlyChanged, setShowOnlyChanged] = useState(false);
	const searchRef = useRef<any>();

	const { schema } = useContext(SchemaContext);
	const { closeFullscreen } = useContext(FullscreenContext);

	const onAddClick = (parentKey: string, indentation: number, root: string) => {
		if (propertyToEdit != null && propertyToEdit.key == null) {
			return;
		}
		const property = newPropertyToEdit(parentKey, indentation, root);
		if (propertyDraftDirty) {
			setPendingPropertyToEdit(property);
			return;
		}
		setPropertyToEdit(property);
	};

	const onEditClick = (propertyKey: string, property: EditableProperty) => {
		if (propertyToEdit?.key === propertyKey) {
			return;
		}
		const nextProperty = { key: propertyKey, ...property };
		if (propertyDraftDirty) {
			setPendingPropertyToEdit(nextProperty);
			return;
		}
		setPropertyToEdit(nextProperty);
	};

	const onRemoveClick = (propertyKey: string, propertyName: string, typeKind: TypeKind) => {
		setPropertyToRemove({ key: propertyKey, name: propertyName, type: typeKind });
	};

	const {
		rows,
		columns,
		changeCount,
		objectCount,
		propertyCount,
		propertyParents,
		propertyStatuses,
		primarySources,
		queries,
		hasSchemaChanges,
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
	} = useSchemaEdit(
		schema,
		onEditClick,
		closeFullscreen,
		propertyToEdit?.key,
		search,
		showOnlyChanged,
		initialPropertyKey,
	);
	const isAddingProperty = propertyToEdit != null && propertyToEdit.key == null;

	const onSaveProperty = (property: PropertyToEdit, primarySource: string | null) => {
		if (property.key == null) {
			onAddProperty(property, primarySource);
			setPropertyToEdit(null);
			setPropertyDraftDirty(false);
			return;
		}
		onEditProperty(property, primarySource);
		setPropertyToEdit(structuredClone(property));
		setPropertyDraftDirty(false);
	};

	const onConfirmRemove = () => {
		onRemoveProperty(propertyToRemove.key);
		if (propertyToEdit?.key === propertyToRemove.key) {
			setPropertyToEdit(null);
			setPropertyDraftDirty(false);
		}
		setPropertyToRemove(null);
	};

	const onDiscardPropertyDraft = () => {
		setPropertyToEdit(pendingPropertyToEdit);
		setPendingPropertyToEdit(null);
		setPropertyDraftDirty(false);
	};

	const onExpandClick = () => {
		sortableGridRef.current?.expand();
	};

	const onCollapseClick = () => {
		sortableGridRef.current?.collapse();
	};

	const onCancelEdit = () => {
		closeFullscreen();
	};

	const onSearchBlur = (event: any) => {
		if (event.target.value === '') {
			setIsSearchOpen(false);
		}
	};

	const onSearchClick = () => {
		setIsSearchOpen(true);
		requestAnimationFrame(() => searchRef.current?.focus());
	};

	return (
		<div className='schema-edit'>
			<div className='schema-edit__header'>
				<div>
					<h1 className='schema-edit__header-title'>Edit profile schema</h1>
					<div className='schema-edit__header-description'>Update the structure of unified profiles.</div>
				</div>
				<div className='schema-edit__header-buttons'>
					<SlButton className='schema-edit__header-cancel-button' onClick={onCancelEdit}>
						Cancel
					</SlButton>
					<SlButton
						className='schema-edit__header-apply-button'
						variant='primary'
						onClick={onApplyChanges}
						disabled={!hasSchemaChanges || propertyDraftDirty || isAddingProperty}
					>
						Review and apply changes...
					</SlButton>
				</div>
			</div>
			<div className='schema-edit__overview'>
				<div className='schema-edit__overview-main'>
					<div className='schema-edit__summary'>
						<span>
							<SlIcon name='table' />
							{propertyCount} {propertyCount === 1 ? 'property' : 'properties'}
						</span>
						<span>
							<SlIcon name='box' />
							{objectCount} {objectCount === 1 ? 'object' : 'objects'}
						</span>
						<div
							className={`schema-edit__change-count${changeCount === 0 ? ' schema-edit__change-count--empty' : ''}`}
						>
							<span />
							{changeCount === 0
								? 'No pending changes'
								: `${changeCount} pending ${changeCount === 1 ? 'change' : 'changes'}`}
						</div>
					</div>
					<SlButton
						variant='text'
						className='schema-edit__add-property'
						onClick={() => onAddClick('', 0, '')}
					>
						<SlIcon name='plus-lg' slot='prefix' />
						Add a new property
					</SlButton>
				</div>
			</div>
			<div className='schema-edit__workspace'>
				<div className='schema-edit__schema-panel'>
					<div className='schema-edit__toolbar'>
						<div className='schema-edit__expansion-buttons'>
							<SlTooltip className='schema-edit__toolbar-tooltip' content='Expand all properties' hoist>
								<SlButton
									className='schema-edit__expand-all-button schema-edit__expansion-button schema-edit__toolbar-icon-button'
									size='small'
									aria-label='Expand all properties'
									onClick={onExpandClick}
								>
									<SlIcon name='chevron-expand' />
								</SlButton>
							</SlTooltip>
							<SlTooltip className='schema-edit__toolbar-tooltip' content='Collapse all properties' hoist>
								<SlButton
									className='schema-edit__collapse-all-button schema-edit__expansion-button schema-edit__toolbar-icon-button'
									size='small'
									aria-label='Collapse all properties'
									onClick={onCollapseClick}
								>
									<SlIcon name='chevron-contract' />
								</SlButton>
							</SlTooltip>
						</div>
						<div className='schema-edit__toolbar-controls'>
							<div
								className={`schema-edit__search-control${isSearchOpen ? ' schema-edit__search-control--open' : ''}`}
							>
								{isSearchOpen ? (
									<SlInput
										ref={searchRef}
										className='schema-edit__search'
										size='small'
										placeholder='Search a property...'
										value={search}
										onSlBlur={onSearchBlur}
										onSlInput={(event: any) => setSearch(event.target.value)}
									>
										<SlIcon name='search' slot='prefix' />
									</SlInput>
								) : (
									<SlTooltip
										className='schema-edit__toolbar-tooltip'
										content='Search properties'
										hoist
									>
										<SlButton
											className='schema-edit__search-button schema-edit__toolbar-icon-button'
											size='small'
											aria-label='Search properties'
											onClick={onSearchClick}
										>
											<SlIcon name='search' />
										</SlButton>
									</SlTooltip>
								)}
							</div>
							<SlTooltip className='schema-edit__toolbar-tooltip' content='Filter properties' hoist>
								<SlDropdown
									className='schema-edit__filter'
									placement='bottom-end'
									distance={8}
									stayOpenOnSelect
									hoist
								>
									<SlButton
										className='schema-edit__filter-button schema-edit__toolbar-icon-button'
										slot='trigger'
										size='small'
										aria-label='Filter properties'
									>
										<SlIcon name='filter' />
										<span
											className={`schema-edit__filter-dot${showOnlyChanged ? ' schema-edit__filter-dot--active' : ''}`}
										/>
									</SlButton>
									<SlMenu className='schema-edit__filter-menu'>
										<SlSwitch
											className='schema-edit__show-changed'
											size='small'
											checked={showOnlyChanged}
											onSlChange={(event: any) => setShowOnlyChanged(event.target.checked)}
										>
											Show only changed properties
										</SlSwitch>
									</SlMenu>
								</SlDropdown>
							</SlTooltip>
						</div>
					</div>
					<SortableGrid
						rows={rows}
						columns={columns}
						gridColumnsWidths={schemaEditGridColumns}
						nestedRowsIndentation={schemaEditNestedRowsIndentation}
						onSortRow={onSortRow}
						ref={sortableGridRef}
					/>
				</div>
				<PropertyPanel
					dirty={propertyDraftDirty}
					property={propertyToEdit}
					parents={propertyParents}
					primarySources={primarySources}
					status={propertyStatuses[propertyToEdit?.key]}
					onClose={() => {
						setPropertyToEdit(null);
						setPropertyDraftDirty(false);
					}}
					onDirtyChange={setPropertyDraftDirty}
					onRemove={(property) => onRemoveClick(property.key, property.name, property.type.kind)}
					onSave={onSaveProperty}
				/>
			</div>
			<SlDialog
				open={isQueriesLoading || queries != null}
				label='Review changes'
				onSlAfterHide={onCancelChanges}
				className={`schema-edit__queries${isQueriesLoading ? ' schema-edit__queries--loading' : ''}`}
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
					/>
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
							<div className='schema-edit__queries-buttons' slot='footer'>
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
				onClose={() => setPropertyToRemove(null)}
				title='Delete property?'
				actions={
					<>
						<SlButton onClick={() => setPropertyToRemove(null)}>Cancel</SlButton>
						<SlButton
							variant='danger'
							className='schema-edit__confirm-remove-property'
							onClick={onConfirmRemove}
						>
							Delete property
						</SlButton>
					</>
				}
			>
				<p>
					The property <b>“{propertyToRemove?.name}”</b> will be deleted when you apply the schema changes.
					{propertyToRemove?.type === 'object' ? ' Its nested properties will also be deleted.' : ''}
				</p>
			</AlertDialog>
			<AlertDialog
				variant='warning'
				isOpen={pendingPropertyToEdit != null}
				onClose={() => setPendingPropertyToEdit(null)}
				title='Discard unsaved changes?'
				actions={
					<>
						<SlButton onClick={() => setPendingPropertyToEdit(null)}>Keep editing</SlButton>
						<SlButton variant='primary' onClick={onDiscardPropertyDraft}>
							Discard changes
						</SlButton>
					</>
				}
			>
				<p>
					{propertyToEdit?.key == null
						? 'The new property has unsaved changes. They will be lost.'
						: `The unsaved changes to “${propertyToEdit?.name}” will be lost.`}
				</p>
			</AlertDialog>
		</div>
	);
};

export { SchemaEdit };
