import React, { useContext } from 'react';
import Fullscreen from '../../base/Fullscreen/Fullscreen';
import AppContext from '../../../context/AppContext';
import { SchemaEdit } from './SchemaEdit';
import { SchemaContext } from '../../../context/SchemaContext';

const SchemaEditWrapper = () => {
	const { redirect } = useContext(AppContext);
	const { setIsLoadingSchema } = useContext(SchemaContext);

	const onClose = () => {
		redirect('schema');
		setIsLoadingSchema(true);
	};

	return (
		<Fullscreen isLoading={false} onClose={onClose}>
			<SchemaEdit />
		</Fullscreen>
	);
};

export { SchemaEditWrapper };
