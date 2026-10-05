import React, { useContext, useLayoutEffect } from 'react';
import './Schema.css';
import AppContext from '../../../context/AppContext';
import { SchemaContext } from '../../../context/SchemaContext';
import { Outlet, useLocation } from 'react-router-dom';
import { useSchema } from './useSchema';

const Schema = () => {
	const { setTitle } = useContext(AppContext);

	const {
		consentPurposes,
		isLoadingSchema,
		setIsLoadingSchema,
		schema,
		isAltering,
		setIsAltering,
		latestAlterError,
	} = useSchema();

	const location = useLocation();

	useLayoutEffect(() => {
		setTitle('Profile Unification / Schema');
	}, [location]);

	return (
		<div className='schema'>
			<SchemaContext.Provider
				value={{
					schema,
					consentPurposes,
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
	);
};

export { Schema };
