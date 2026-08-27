import React, { useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import '../Schema/SchemaPropertyGrid.css';
import './SchemaGrid.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Outlet, useLocation } from 'react-router-dom';
import Grid from '../../base/Grid/Grid';
import { GridRef } from '../../base/Grid/Grid.types';
import AppContext from '../../../context/AppContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { PropertyDetailsPanel } from './PropertyDetailsPanel';
import { useSchemaGrid } from './useSchemaGrid';
import { GridKeyboardHints } from '../../base/Grid/GridKeyboardHints';
import { useDocumentGridKeyboardNavigation } from '../../base/Grid/useDocumentGridKeyboardNavigation';
import {
	SchemaPropertyGridSummary,
	SchemaPropertyGridToolbar,
	schemaPropertyGridNestedRowsIndentation,
} from '../Schema/SchemaPropertyGrid';

const schemaGridColumns = 'minmax(170px, 0.65fr) minmax(220px, 0.85fr) minmax(300px, 1.75fr) minmax(160px, 0.55fr)';

interface SchemaGridOutletContext {
	selectedPropertyPath: string | null;
}

const SchemaGrid = () => {
	const [search, setSearch] = useState('');
	const [selectedPropertyPath, setSelectedPropertyPath] = useState<string | null>(null);
	const { redirect } = useContext(AppContext);
	const { schema, isLoadingSchema, latestAlterError, isAltering } = useContext(SchemaContext);
	const gridRef = useRef<GridRef>(null);
	const location = useLocation();
	const isEditing = location.pathname.endsWith('/schema/edit');
	const isSearchActive = search.trim() !== '';
	const onSelectProperty = useCallback((path: string) => setSelectedPropertyPath(path), []);

	const {
		columns,
		firstVisiblePropertyPath,
		isSelectedPropertyVisible,
		objectCount,
		propertyCount,
		rows,
		selectedProperty,
		visiblePropertyCount,
	} = useSchemaGrid(schema, isLoadingSchema, search, selectedPropertyPath, onSelectProperty);
	const visibleSelectedProperty = isSelectedPropertyVisible ? selectedProperty : null;
	const [lastSelectedProperty, setLastSelectedProperty] = useState(visibleSelectedProperty);
	const isDetailsPanelOpen = visibleSelectedProperty != null && !isEditing;
	const detailsPanelProperty = isEditing ? null : visibleSelectedProperty || lastSelectedProperty;
	const gridInteractionsDisabled = isLoadingSchema || isAltering;
	const isGridKeyboardNavigationEnabled = !isEditing && !gridInteractionsDisabled && visiblePropertyCount > 0;
	const expansionDisabled = gridInteractionsDisabled || isSearchActive || objectCount === 0;

	useDocumentGridKeyboardNavigation(gridRef, isGridKeyboardNavigationEnabled);

	useEffect(() => {
		if (!isGridKeyboardNavigationEnabled) {
			return;
		}
		const animationFrame = requestAnimationFrame(() => {
			if (document.activeElement == null || document.activeElement === document.body) {
				gridRef.current?.focus();
			}
		});
		return () => cancelAnimationFrame(animationFrame);
	}, [isGridKeyboardNavigationEnabled, propertyCount]);

	useLayoutEffect(() => {
		if (isLoadingSchema || selectedPropertyPath == null) {
			return;
		}
		if (selectedProperty == null) {
			setSelectedPropertyPath(null);
			return;
		}
		if (!isSelectedPropertyVisible) {
			setSelectedPropertyPath(firstVisiblePropertyPath);
		}
	}, [firstVisiblePropertyPath, isLoadingSchema, isSelectedPropertyVisible, selectedProperty, selectedPropertyPath]);

	useEffect(() => {
		if (visibleSelectedProperty != null) {
			setLastSelectedProperty(visibleSelectedProperty);
		}
	}, [visibleSelectedProperty]);

	const onEditClick = () => {
		redirect('profile-unification/schema/edit');
	};

	const onExpandClick = () => {
		gridRef.current?.expand();
	};

	const onCollapseClick = () => {
		gridRef.current?.collapse();
	};

	const onDetailsPanelTransitionEnd = (event: React.TransitionEvent<HTMLDivElement>) => {
		if (event.propertyName === 'opacity' && !isDetailsPanelOpen) {
			setLastSelectedProperty(null);
		}
	};

	return (
		<div className='schema-grid'>
			<div className='schema-grid__page-header'>
				<div>
					<h1>Profile schema</h1>
					<div className='schema-grid__page-description'>
						Explore the canonical profile schema used for identity resolution and unification.
					</div>
					<SchemaPropertyGridSummary
						view='readOnly'
						objectCount={objectCount}
						propertyCount={propertyCount}
					/>
				</div>
				<SlButton
					className='schema-grid__alter-button'
					variant='primary'
					onClick={gridInteractionsDisabled ? null : onEditClick}
					disabled={gridInteractionsDisabled}
					loading={isAltering}
				>
					Edit schema
				</SlButton>
			</div>
			{!isAltering && latestAlterError && (
				<div className='schema-grid__alter-error'>
					<SlIcon name='exclamation-circle' />
					{latestAlterError}
				</div>
			)}
			<div
				className={`grid-keyboard-hints-layout schema-grid__layout${isDetailsPanelOpen ? ' schema-grid__layout--with-panel' : ''}`}
			>
				<div
					className={`schema-grid__workspace${isDetailsPanelOpen ? ' schema-grid__workspace--with-panel' : ''}`}
				>
					<div className='schema-grid__card grid-keyboard-hints-layout__grid'>
						<SchemaPropertyGridToolbar
							view='readOnly'
							expansionDisabled={expansionDisabled}
							onCollapse={onCollapseClick}
							onExpand={onExpandClick}
							onSearchChange={setSearch}
							search={search}
						/>
						<Grid
							ref={gridRef}
							columns={columns}
							rows={rows}
							keyboardNavigation={isGridKeyboardNavigationEnabled}
							gridColumnsWidths={schemaGridColumns}
							nestedRowsIndentation={schemaPropertyGridNestedRowsIndentation}
							isLoading={isLoadingSchema || isAltering}
							loadingText={isAltering ? 'Schema is being altered' : 'Loading schema'}
							noRowsIcon='search'
							noRowsMessage={isSearchActive ? 'No properties match your search' : undefined}
						/>
					</div>
					{detailsPanelProperty != null && (
						<div className='schema-grid__details-panel' onTransitionEnd={onDetailsPanelTransitionEnd}>
							<PropertyDetailsPanel
								property={detailsPanelProperty.property}
								primarySource={detailsPanelProperty.primarySource}
								onClose={() => setSelectedPropertyPath(null)}
							/>
						</div>
					)}
				</div>
				{propertyCount > 0 && (
					<GridKeyboardHints
						disabled={gridInteractionsDisabled || visiblePropertyCount === 0}
						expansionDisabled={expansionDisabled}
					/>
				)}
			</div>
			<Outlet
				context={
					{
						selectedPropertyPath,
					} satisfies SchemaGridOutletContext
				}
			/>
		</div>
	);
};

export default SchemaGrid;
export type { SchemaGridOutletContext };
