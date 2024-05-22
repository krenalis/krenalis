import { useState, useEffect, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import { ObjectType } from '../../../lib/api/types/types';

const useSchema = () => {
	const [isLoadingSchema, setIsLoadingSchema] = useState<boolean>(true);
	const [schema, setSchema] = useState<ObjectType>();

	const { api, handleError, warehouse, redirect, selectedWorkspace } = useContext(AppContext);

	useEffect(() => {
		const fetchSchema = async () => {
			let schema: ObjectType;
			try {
				schema = await api.workspaces.userSchema();
			} catch (err) {
				handleError(err);
				return;
			}
			setSchema(schema);
			setTimeout(() => setIsLoadingSchema(false), 300);
		};
		if (warehouse == null) {
			redirect('settings');
			handleError('Please connect to a data warehouse before proceeding');
			return;
		}
		fetchSchema();
	}, [selectedWorkspace, isLoadingSchema]);

	return {
		isLoadingSchema,
		setIsLoadingSchema,
		schema,
	};
};

export { useSchema };
