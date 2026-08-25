import React, { useCallback, useContext, useEffect, useRef, useState } from 'react';
import '../Schema/SchemaPropertyGrid.css';
import './SchemaEdit.css';
import { useBeforeUnload, useBlocker } from 'react-router-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SortableGrid from '../../base/Grid/SortableGrid';
import { GridKeyboardHints } from '../../base/Grid/GridKeyboardHints';
import { useDocumentGridKeyboardNavigation } from '../../base/Grid/useDocumentGridKeyboardNavigation';
import SyntaxHighlight from '../../base/SyntaxHighlight/SyntaxHighlight';
import { FullscreenContext } from '../../../context/FullscreenContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { TypeKind } from '../../../lib/api/types/types';
import { EditableProperty, getParentPropertyKey, newPropertyToEdit } from './SchemaEdit.helpers';
import { PropertyPanel } from './PropertyPanel';
import { PropertyToEdit, PropertyToRemove, SelectPropertyOptions, useSchemaEdit } from './useSchemaEdit';
import {
	SchemaPropertyGridSummary,
	SchemaPropertyGridToolbar,
	schemaPropertyGridNestedRowsIndentation,
} from '../Schema/SchemaPropertyGrid';

const schemaEditGridColumns =
	'minmax(160px, 0.65fr) minmax(210px, 0.85fr) minmax(240px, 1.5fr) minmax(160px, 0.65fr) 90px';

interface SchemaEditProps {
	initialPropertyKey?: string | null;
}

const SchemaEdit = ({ initialPropertyKey }: SchemaEditProps) => {
	const [propertyToEdit, setPropertyToEdit] = useState<PropertyToEdit | null>(null);
	const [propertyToRemove, setPropertyToRemove] = useState<PropertyToRemove | null>(null);
	const [animatePropertyActions, setAnimatePropertyActions] = useState(false);
	const [isCancelEditPending, setIsCancelEditPending] = useState(false);
	const [isDiscardingAndLeaving, setIsDiscardingAndLeaving] = useState(false);
	const [propertyDraftDirty, setPropertyDraftDirty] = useState(false);
	const [search, setSearch] = useState('');
	const [showOnlyChanged, setShowOnlyChanged] = useState(false);
	const selectedPropertyBeforeAddRef = useRef<PropertyToEdit | null>(null);
	const discardAndLeaveRef = useRef(false);
	const skipNavigationBlockRef = useRef(false);

	const { schema } = useContext(SchemaContext);
	const { closeFullscreen } = useContext(FullscreenContext);
	const hasUnsavedPropertyChanges = propertyToEdit != null && propertyDraftDirty;
	const resetPropertyDraft = (property: PropertyToEdit | null) => {
		setPropertyToEdit(property == null ? null : structuredClone(property));
	};
	const closeAfterApplyingChanges = () => {
		skipNavigationBlockRef.current = true;
		closeFullscreen();
	};

	const onSelectProperty = useCallback(
		(propertyKey: string, property: EditableProperty, options: SelectPropertyOptions = {}) => {
			if (propertyToEdit?.key === propertyKey) {
				return;
			}
			if (hasUnsavedPropertyChanges) {
				if (options.animateActionsIfBlocked !== false) {
					setAnimatePropertyActions(true);
				}
				return;
			}
			const nextProperty = { key: propertyKey, ...property };
			selectedPropertyBeforeAddRef.current = null;
			setAnimatePropertyActions(false);
			setPropertyToEdit(nextProperty);
		},
		[hasUnsavedPropertyChanges, propertyToEdit?.key],
	);

	const onRemoveClick = (propertyKey: string, propertyName: string, typeKind: TypeKind) => {
		setPropertyToRemove({ key: propertyKey, name: propertyName, type: typeKind });
	};

	const {
		rows,
		columns,
		changeCount,
		firstVisibleProperty,
		hasObjects,
		isFiltered,
		isSchemaReady,
		isSelectedPropertyVisible,
		propertyCount,
		visiblePropertyCount,
		propertyParents,
		selectedPropertyFieldChanges,
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
		onSelectProperty,
		closeAfterApplyingChanges,
		propertyToEdit?.key,
		search,
		showOnlyChanged,
		hasUnsavedPropertyChanges,
		initialPropertyKey,
	);
	useEffect(() => {
		if (
			!isSchemaReady ||
			(propertyToEdit != null && propertyToEdit.key == null) ||
			hasUnsavedPropertyChanges ||
			isSelectedPropertyVisible
		) {
			return;
		}
		if (firstVisibleProperty != null) {
			onSelectProperty(firstVisibleProperty.key, firstVisibleProperty, { animateActionsIfBlocked: false });
			return;
		}
		selectedPropertyBeforeAddRef.current = null;
		setPropertyToEdit(null);
	}, [
		firstVisibleProperty,
		hasUnsavedPropertyChanges,
		isSchemaReady,
		isSelectedPropertyVisible,
		onSelectProperty,
		propertyToEdit,
	]);
	const propertyInPanel = propertyToEdit?.key == null || isSelectedPropertyVisible ? propertyToEdit : null;
	const hasPendingChanges = hasUnsavedPropertyChanges || hasSchemaChanges;
	const shouldBlockNavigation = useCallback(
		() => hasPendingChanges && !skipNavigationBlockRef.current,
		[hasPendingChanges],
	);
	const navigationBlocker = useBlocker(shouldBlockNavigation);
	const isGridKeyboardNavigationEnabled = visiblePropertyCount > 0;
	const expansionDisabled = isFiltered || !hasObjects;
	let discardChangesDescription = 'The pending schema changes will be discarded.';
	if (hasUnsavedPropertyChanges) {
		if (hasSchemaChanges) {
			discardChangesDescription =
				'The unsaved property changes and all pending schema changes will be discarded.';
		} else if (propertyToEdit?.key == null) {
			discardChangesDescription = 'The new property will be discarded.';
		} else {
			discardChangesDescription = `The unsaved changes to “${propertyToEdit.name}” will be discarded.`;
		}
	}

	useBeforeUnload(
		useCallback(
			(event) => {
				if (hasPendingChanges) {
					event.preventDefault();
					event.returnValue = '';
				}
			},
			[hasPendingChanges],
		),
	);

	useDocumentGridKeyboardNavigation(sortableGridRef, isGridKeyboardNavigationEnabled);

	useEffect(() => {
		if (!isGridKeyboardNavigationEnabled) {
			return;
		}
		const animationFrame = requestAnimationFrame(() => sortableGridRef.current?.focus());
		return () => cancelAnimationFrame(animationFrame);
	}, [isGridKeyboardNavigationEnabled, propertyCount, sortableGridRef]);

	const onAddClick = () => {
		if (hasUnsavedPropertyChanges) {
			setAnimatePropertyActions(true);
			return;
		}

		const contextualProperty = propertyToEdit?.key == null ? selectedPropertyBeforeAddRef.current : propertyToEdit;
		if (propertyToEdit?.key != null) {
			selectedPropertyBeforeAddRef.current = structuredClone(propertyToEdit);
		}
		let parent = propertyParents[0];
		if (contextualProperty?.key != null) {
			const parentKey =
				contextualProperty.type?.kind === 'object'
					? contextualProperty.key
					: getParentPropertyKey(contextualProperty.key);
			parent = propertyParents.find((candidate) => candidate.key === parentKey) || parent;
		}

		const property = newPropertyToEdit(parent.key, parent.indentation, parent.root);
		setAnimatePropertyActions(false);
		setPropertyToEdit(property);
	};

	const onSaveProperty = (property: PropertyToEdit, primarySource: string | null) => {
		setAnimatePropertyActions(false);
		if (property.key == null) {
			const addedProperty = onAddProperty(property, primarySource);
			selectedPropertyBeforeAddRef.current = null;
			setPropertyToEdit(addedProperty);
			setPropertyDraftDirty(false);
			return;
		}
		onEditProperty(property, primarySource);
		resetPropertyDraft(property);
		setPropertyDraftDirty(false);
	};

	const onCancelProperty = () => {
		setAnimatePropertyActions(false);
		setPropertyDraftDirty(false);
		if (propertyToEdit?.key == null) {
			const previousProperty = selectedPropertyBeforeAddRef.current;
			selectedPropertyBeforeAddRef.current = null;
			resetPropertyDraft(previousProperty);
			return;
		}
		resetPropertyDraft(propertyToEdit);
	};

	const onConfirmRemove = () => {
		const nextProperty = onRemoveProperty(propertyToRemove.key);
		if (propertyToEdit?.key === propertyToRemove.key) {
			selectedPropertyBeforeAddRef.current = null;
			setPropertyToEdit(nextProperty);
			setPropertyDraftDirty(false);
		}
		setPropertyToRemove(null);
	};

	const onExpandClick = () => {
		sortableGridRef.current?.expand();
	};

	const onCollapseClick = () => {
		sortableGridRef.current?.collapse();
	};

	const onCancelEdit = () => {
		if (hasPendingChanges) {
			setIsCancelEditPending(true);
			return;
		}
		closeFullscreen();
	};

	const onDiscardChangesAndLeave = () => {
		discardAndLeaveRef.current = true;
		setAnimatePropertyActions(false);
		setIsDiscardingAndLeaving(true);
		setIsCancelEditPending(false);
		setPropertyToEdit(null);
		setPropertyDraftDirty(false);
		skipNavigationBlockRef.current = true;
	};

	const onKeepEditing = () => {
		discardAndLeaveRef.current = false;
		setIsDiscardingAndLeaving(false);
		setIsCancelEditPending(false);
		skipNavigationBlockRef.current = false;
		if (navigationBlocker.state === 'blocked') {
			navigationBlocker.reset();
		}
	};

	const onDiscardDialogAfterHide = () => {
		if (!discardAndLeaveRef.current) {
			return;
		}
		discardAndLeaveRef.current = false;
		if (navigationBlocker.state === 'blocked') {
			navigationBlocker.proceed();
			return;
		}
		closeFullscreen();
	};

	const onReviewChangesClick = () => {
		if (hasUnsavedPropertyChanges) {
			setAnimatePropertyActions(true);
			return;
		}
		if (propertyToEdit?.key == null) {
			onCancelProperty();
		}
		onApplyChanges();
	};

	const onReviewDialogRequestClose = (event: Event) => {
		if (isConfirmChangesLoading) {
			event.preventDefault();
			return;
		}
		onCancelChanges();
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
						onClick={onReviewChangesClick}
						disabled={!hasSchemaChanges && !hasUnsavedPropertyChanges}
					>
						Review and apply changes...
					</SlButton>
				</div>
			</div>
			<div className='schema-edit__overview'>
				<div className='schema-edit__overview-main'>
					<SchemaPropertyGridSummary view='edit' propertyCount={propertyCount}>
						<div
							className={`schema-edit__change-count${changeCount === 0 ? ' schema-edit__change-count--empty' : ''}`}
						>
							<span />
							{changeCount === 0
								? 'No pending changes'
								: `${changeCount} pending ${changeCount === 1 ? 'change' : 'changes'}`}
						</div>
					</SchemaPropertyGridSummary>
					<SlButton variant='text' className='schema-edit__add-property' onClick={onAddClick}>
						<SlIcon name='plus-lg' slot='prefix' />
						Add a new property
					</SlButton>
				</div>
			</div>
			<div className='grid-keyboard-hints-layout schema-edit__layout'>
				<div className='schema-edit__workspace'>
					<div className='schema-edit__schema-panel grid-keyboard-hints-layout__grid'>
						<SchemaPropertyGridToolbar
							view='edit'
							expansionDisabled={expansionDisabled}
							onCollapse={onCollapseClick}
							onExpand={onExpandClick}
							onSearchChange={setSearch}
							search={search}
						>
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
						</SchemaPropertyGridToolbar>
						<SortableGrid
							rows={rows}
							columns={columns}
							keyboardNavigation={isGridKeyboardNavigationEnabled}
							gridColumnsWidths={schemaEditGridColumns}
							nestedRowsIndentation={schemaPropertyGridNestedRowsIndentation}
							onSortRow={onSortRow}
							reorderDisabled={isFiltered}
							ref={sortableGridRef}
						/>
					</div>
					<PropertyPanel
						animateActions={animatePropertyActions}
						dirty={propertyDraftDirty}
						fieldChanges={propertyInPanel == null ? undefined : selectedPropertyFieldChanges}
						property={propertyInPanel}
						parents={propertyParents}
						primarySources={primarySources}
						status={propertyStatuses[propertyInPanel?.key]}
						onClose={onCancelProperty}
						onActionsAnimationFinish={() => setAnimatePropertyActions(false)}
						onDirtyChange={setPropertyDraftDirty}
						onRemove={(property) => onRemoveClick(property.key, property.name, property.type.kind)}
						onSave={onSaveProperty}
					/>
				</div>
				{propertyCount > 0 && (
					<GridKeyboardHints
						canReorder
						disabled={visiblePropertyCount === 0}
						expansionDisabled={expansionDisabled}
						reorderDisabled={isFiltered}
					/>
				)}
			</div>
			<SlDialog
				open={isQueriesLoading || queries != null}
				label='Review changes'
				onSlRequestClose={onReviewDialogRequestClose}
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
								<SlButton size='small' onClick={onCancelChanges} disabled={isConfirmChangesLoading}>
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
				isOpen={!isDiscardingAndLeaving && (isCancelEditPending || navigationBlocker.state === 'blocked')}
				onClose={onDiscardDialogAfterHide}
				onRequestClose={onKeepEditing}
				title='Discard unsaved changes?'
				actions={
					<>
						<SlButton onClick={onKeepEditing}>Keep editing</SlButton>
						<SlButton variant='primary' onClick={onDiscardChangesAndLeave}>
							Discard and leave
						</SlButton>
					</>
				}
			>
				<p>{discardChangesDescription}</p>
			</AlertDialog>
		</div>
	);
};

export { SchemaEdit };
