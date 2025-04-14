import { useState, useEffect, useContext, useRef } from 'react';
import AppContext from '../../../context/AppContext';
import { ObjectType } from '../../../lib/api/types/types';
import { LatestUserSchemaUpdate } from '../../../lib/api/types/workspace';

const useSchema = () => {
	const [isLoadingSchema, setIsLoadingSchema] = useState<boolean>(true);
	const [schema, setSchema] = useState<ObjectType>();
	const [isUpdating, setIsUpdating] = useState<boolean>(false);
	const [latestUpdateError, setLatestUpdateError] = useState<string>();

	const { api, handleError, selectedWorkspace } = useContext(AppContext);

	const isUpdatingRef = useRef<boolean>();

	useEffect(() => {
		isUpdatingRef.current = isUpdating;
	}, [isUpdating]);

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
		fetchSchema();
	}, [selectedWorkspace, isLoadingSchema]);

	useEffect(() => {
		const intervalID = setInterval(() => {
			handleSchemaUpdate();
		}, 3000);

		handleSchemaUpdate();

		return () => {
			clearInterval(intervalID);
		};
	}, []);

	const handleSchemaUpdate = async () => {
		let res: LatestUserSchemaUpdate;
		try {
			res = await api.workspaces.latestUserSchemaUpdate();
		} catch (err) {
			handleError(err);
			return;
		}
		const startTime = res.startTime;
		const endTime = res.endTime;
		if (startTime != null && endTime == null) {
			// the schema is being updated.
			setIsUpdating(true);
		} else if (isUpdatingRef.current && endTime != null) {
			// schema update is concluded.
			setIsLoadingSchema(true);
			setIsUpdating(false);
		}
		setLatestUpdateError(res.error);
	};

	return {
		isLoadingSchema,
		setIsLoadingSchema,
		schema,
		isUpdating,
		setIsUpdating,
		latestUpdateError,
	};
};

export { useSchema };
