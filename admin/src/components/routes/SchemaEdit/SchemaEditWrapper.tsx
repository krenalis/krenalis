import React, { useContext } from 'react';
import { useOutletContext } from 'react-router-dom';
import Fullscreen from '../../base/Fullscreen/Fullscreen';
import AppContext from '../../../context/AppContext';
import { SchemaEdit } from './SchemaEdit';
import { SchemaContext } from '../../../context/SchemaContext';
import type { SchemaGridOutletContext } from '../SchemaGrid/SchemaGrid';

const SchemaEditWrapper = () => {
	const { redirect } = useContext(AppContext);
	const { setIsLoadingSchema } = useContext(SchemaContext);
	const { selectedPropertyPath } = useOutletContext<SchemaGridOutletContext>();

	const onClose = () => {
		redirect('profile-unification/schema');
		setIsLoadingSchema(true);
	};

	return (
		<Fullscreen className='schema-edit-fullscreen' isLoading={false} onClose={onClose}>
			<SchemaEdit initialPropertyKey={selectedPropertyPath} />
		</Fullscreen>
	);
};

export { SchemaEditWrapper };
