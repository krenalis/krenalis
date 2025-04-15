import React, { useContext, useLayoutEffect } from 'react';
import './Schema.css';
import AppContext from '../../../context/AppContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { Outlet, useLocation } from 'react-router-dom';
import { useSchema } from './useSchema';

const Schema = () => {
	const { setTitle } = useContext(AppContext);

	const { isLoadingSchema, setIsLoadingSchema, schema, isAltering, setIsAltering, latestAlterError } = useSchema();

	const location = useLocation();

	useLayoutEffect(() => {
		setTitle('Customer Model');
	}, [location]);

	return (
		<div className='schema'>
			<div className='route-content'>
				<SchemaContext.Provider
					value={{
						schema,
						isLoadingSchema,
						setIsLoadingSchema,
						latestAlterError,
						isAltering,
						setIsAltering,
					}}
				>
					<Outlet />
				</SchemaContext.Provider>
			</div>
		</div>
	);
};

export { Schema };
