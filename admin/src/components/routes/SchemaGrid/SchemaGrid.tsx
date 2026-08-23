import React, { useCallback, useContext, useEffect, useRef, useState } from 'react';
import './SchemaGrid.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { Outlet, useLocation } from 'react-router-dom';
import Grid from '../../base/Grid/Grid';
import AppContext from '../../../context/AppContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { PropertyDetailsPanel } from './PropertyDetailsPanel';
import { useSchemaGrid } from './useSchemaGrid';

const schemaGridColumns = 'minmax(170px, 0.65fr) minmax(220px, 0.85fr) minmax(300px, 1.75fr) minmax(160px, 0.55fr)';
const schemaGridNestedRowsIndentation = { base: 34, step: 20 };

interface SchemaGridOutletContext {
	selectedPropertyPath: string | null;
}

const SchemaGrid = () => {
	const [isSearchOpen, setIsSearchOpen] = useState(false);
	const [search, setSearch] = useState('');
	const [selectedPropertyPath, setSelectedPropertyPath] = useState<string | null>(null);
	const { redirect } = useContext(AppContext);
	const { schema, isLoadingSchema, latestAlterError, isAltering } = useContext(SchemaContext);
	const gridRef = useRef<any>();
	const searchRef = useRef<any>();
	const location = useLocation();
	const isEditing = location.pathname.endsWith('/schema/edit');
	const onSelectProperty = useCallback((path: string) => setSelectedPropertyPath(path), []);

	const { columns, objectCount, propertyCount, rows, selectedProperty } = useSchemaGrid(
		schema,
		isLoadingSchema,
		search,
		selectedPropertyPath,
		onSelectProperty,
	);
	const [lastSelectedProperty, setLastSelectedProperty] = useState(selectedProperty);
	const isDetailsPanelOpen = selectedProperty != null && !isEditing;
	const detailsPanelProperty = isEditing ? null : selectedProperty || lastSelectedProperty;

	useEffect(() => {
		if (!isLoadingSchema && selectedPropertyPath != null && selectedProperty == null) {
			setSelectedPropertyPath(null);
		}
	}, [isLoadingSchema, selectedProperty, selectedPropertyPath]);

	useEffect(() => {
		if (selectedProperty != null) {
			setLastSelectedProperty(selectedProperty);
		}
	}, [selectedProperty]);

	const onEditClick = () => {
		redirect('profile-unification/schema/edit');
	};

	const onExpandClick = () => {
		gridRef.current?.expand();
	};

	const onCollapseClick = () => {
		gridRef.current?.collapse();
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
					<div className='schema-grid__summary'>
						<span>
							<SlIcon name='table' />
							{propertyCount} {propertyCount === 1 ? 'property' : 'properties'}
						</span>
						<span>
							<SlIcon name='box' />
							{objectCount} {objectCount === 1 ? 'object' : 'objects'}
						</span>
					</div>
				</div>
				<SlButton
					className='schema-grid__alter-button'
					variant='primary'
					onClick={isAltering ? null : onEditClick}
					disabled={isAltering}
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
			<div className={`schema-grid__workspace${isDetailsPanelOpen ? ' schema-grid__workspace--with-panel' : ''}`}>
				<div className='schema-grid__card'>
					<div className='schema-grid__toolbar'>
						<div className='schema-grid__expansion-buttons'>
							<SlTooltip className='schema-grid__toolbar-tooltip' content='Expand all properties' hoist>
								<SlButton
									className='schema-grid__expand-all-button schema-grid__toolbar-icon-button'
									size='small'
									aria-label='Expand all properties'
									onClick={onExpandClick}
								>
									<SlIcon name='chevron-expand' />
								</SlButton>
							</SlTooltip>
							<SlTooltip className='schema-grid__toolbar-tooltip' content='Collapse all properties' hoist>
								<SlButton
									className='schema-grid__collapse-all-button schema-grid__toolbar-icon-button'
									size='small'
									aria-label='Collapse all properties'
									onClick={onCollapseClick}
								>
									<SlIcon name='chevron-contract' />
								</SlButton>
							</SlTooltip>
						</div>
						<div
							className={`schema-grid__search-control${isSearchOpen ? ' schema-grid__search-control--open' : ''}`}
						>
							{isSearchOpen ? (
								<SlInput
									ref={searchRef}
									className='schema-grid__search'
									size='small'
									placeholder='Search a property...'
									value={search}
									onSlBlur={onSearchBlur}
									onSlInput={(event: any) => setSearch(event.target.value)}
								>
									<SlIcon name='search' slot='prefix' />
								</SlInput>
							) : (
								<SlTooltip className='schema-grid__toolbar-tooltip' content='Search properties' hoist>
									<SlButton
										className='schema-grid__search-button schema-grid__toolbar-icon-button'
										size='small'
										aria-label='Search properties'
										onClick={onSearchClick}
									>
										<SlIcon name='search' />
									</SlButton>
								</SlTooltip>
							)}
						</div>
					</div>
					<Grid
						ref={gridRef}
						columns={columns}
						rows={rows}
						gridColumnsWidths={schemaGridColumns}
						nestedRowsIndentation={schemaGridNestedRowsIndentation}
						isLoading={isLoadingSchema || isAltering}
						loadingText={isAltering ? 'Schema is being altered' : 'Loading schema'}
						noRowsMessage={search === '' ? undefined : 'No properties match your search'}
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
			<Outlet
				context={{ selectedPropertyPath: selectedProperty?.path || null } satisfies SchemaGridOutletContext}
			/>
		</div>
	);
};

export default SchemaGrid;
export type { SchemaGridOutletContext };
