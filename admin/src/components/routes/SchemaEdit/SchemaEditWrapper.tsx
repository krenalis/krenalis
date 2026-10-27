import React, { useContext, useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import Fullscreen from '../../base/Fullscreen/Fullscreen';
import AppContext from '../../../context/AppContext';
import { SchemaEdit } from './SchemaEdit';
import { SchemaContext } from '../../../context/SchemaContext';
import type { SchemaGridOutletContext } from '../SchemaGrid/SchemaGrid';

const SchemaEditWrapper = () => {
	const { redirect } = useContext(AppContext);
	const { isLoadingSchema, setIsLoadingSchema, isAltering } = useContext(SchemaContext);
	const { selectedPropertyPath } = useOutletContext<SchemaGridOutletContext>();
	const [hasOpenedEditor, setHasOpenedEditor] = useState(false);
	const isLoading = !hasOpenedEditor && (isLoadingSchema || isAltering);

	useEffect(() => {
		if (!isLoading) {
			setHasOpenedEditor(true);
		}
	}, [isLoading]);

	const onClose = () => {
		redirect('profile-unification/schema');
		setIsLoadingSchema(true);
	};

	return (
		<Fullscreen className='schema-edit-fullscreen' isLoading={isLoading} onClose={onClose}>
			<SchemaEdit initialPropertyKey={selectedPropertyPath} />
		</Fullscreen>
	);
};

export { SchemaEditWrapper };
