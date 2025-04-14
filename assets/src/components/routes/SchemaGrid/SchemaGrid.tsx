import React, { useContext, useRef } from 'react';
import './SchemaGrid.css';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { SchemaContext } from '../../../context/SchemaContext';
import { useSchemaGrid } from './useSchemaGrid';
import { Outlet } from 'react-router-dom';
import AppContext from '../../../context/AppContext';
import Toolbar from '../../base/Toolbar/Toolbar';

const SchemaGrid = () => {
	const { redirect } = useContext(AppContext);
	const { schema, isLoadingSchema, latestUpdateError, isUpdating } = useContext(SchemaContext);

	const gridRef = useRef<any>();

	const { columns, rows } = useSchemaGrid(schema, isLoadingSchema);

	const onEditClick = () => {
		redirect('schema/edit');
	};

	const onExpandClick = () => {
		if (gridRef.current) {
			gridRef.current.expand();
		}
	};

	const onCollapseClick = () => {
		if (gridRef.current) {
			gridRef.current.collapse();
		}
	};

	return (
		<div className='schema-grid'>
			{!isUpdating && latestUpdateError && (
				<div className='schema-grid__alter-error'>
					<SlIcon name='exclamation-circle' />
					{latestUpdateError}
				</div>
			)}
			<Toolbar className='schema-grid__toolbar'>
				<div className='schema-grid__expansion-buttons'>
					<SlButton className='schema-grid__expand-all-button' onClick={onExpandClick}>
						<SlIcon name='arrows-expand' slot='prefix' />
						Expand all
					</SlButton>
					<SlButton className='schema-grid__collapse-all-button' onClick={onCollapseClick}>
						<SlIcon name='arrows-collapse' slot='prefix' />
						Collapse all
					</SlButton>
				</div>
				<div className='schema-grid__alter'>
					<SlButton
						className='schema-grid__alter-button'
						variant='primary'
						onClick={isUpdating ? null : onEditClick}
						disabled={isUpdating}
						loading={isUpdating}
					>
						Alter schema...
					</SlButton>
				</div>
			</Toolbar>
			<Grid
				ref={gridRef}
				columns={columns}
				rows={rows}
				isLoading={isLoadingSchema || isUpdating}
				loadingText={isUpdating ? 'Schema is being updated' : 'Loading schema'}
			/>
			<Outlet />
		</div>
	);
};

export default SchemaGrid;
