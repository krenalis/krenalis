import { useState, useEffect, useContext, useRef } from 'react';
import AppContext from '../../../context/AppContext';
import { ObjectType } from '../../../lib/api/types/types';
import { LatestAlterUserSchema } from '../../../lib/api/types/workspace';

const useSchema = () => {
	const [isLoadingSchema, setIsLoadingSchema] = useState<boolean>(true);
	const [schema, setSchema] = useState<ObjectType>();
	const [isAltering, setIsAltering] = useState<boolean>(false);
	const [latestAlterError, setLatestAlterError] = useState<string>();

	const { api, handleError, selectedWorkspace } = useContext(AppContext);

	const isAlteringRef = useRef<boolean>();

	useEffect(() => {
		isAlteringRef.current = isAltering;
	}, [isAltering]);

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
			handleSchemaAltering();
		}, 3000);

		handleSchemaAltering();

		return () => {
			clearInterval(intervalID);
		};
	}, []);

	const handleSchemaAltering = async () => {
		let res: LatestAlterUserSchema;
		try {
			res = await api.workspaces.LatestAlterUserSchema();
		} catch (err) {
			handleError(err);
			return;
		}
		const startTime = res.startTime;
		const endTime = res.endTime;
		if (startTime != null && endTime == null) {
			// the schema is being altered.
			setIsAltering(true);
		} else if (isAlteringRef.current && endTime != null) {
			// schema altering is concluded.
			setIsLoadingSchema(true);
			setIsAltering(false);
		}
		setLatestAlterError(res.error);
	};

	return {
		isLoadingSchema,
		setIsLoadingSchema,
		schema,
		isAltering,
		setIsAltering,
		latestAlterError,
	};
};

export { useSchema };
