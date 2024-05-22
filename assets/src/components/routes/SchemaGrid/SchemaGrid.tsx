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
	const { schema, isLoadingSchema } = useContext(SchemaContext);

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
			<Toolbar className='schema-grid__toolbar'>
				<div className='schema-grid__expansion-buttons'>
					<SlButton onClick={onExpandClick}>
						<SlIcon name='arrows-expand' slot='prefix' />
						Expand all
					</SlButton>
					<SlButton onClick={onCollapseClick}>
						<SlIcon name='arrows-collapse' slot='prefix' />
						Collapse all
					</SlButton>
				</div>
				<SlButton className='schema-grid__edit-button' variant='primary' onClick={onEditClick}>
					Change...
				</SlButton>
			</Toolbar>
			<Grid ref={gridRef} columns={columns} rows={rows} isLoading={isLoadingSchema} />
			<Outlet />
		</div>
	);
};

export default SchemaGrid;
