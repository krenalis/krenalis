import { useState, useEffect, useContext, useRef } from 'react';
import AppContext from '../../../context/AppContext';
import { ObjectType } from '../../../lib/api/types/types';
import { ConsentPurposesResponse } from '../../../lib/api/types/responses';
import { ConsentPurpose, LatestAlterProfileSchema } from '../../../lib/api/types/workspace';

const CONSENT_PURPOSES_REFRESH_INTERVAL = 30000;

interface ConsentPurposesState {
	workspace: string;
	purposes: ConsentPurpose[];
}

const useSchema = () => {
	const [isLoadingSchema, setIsLoadingSchema] = useState<boolean>(true);
	const [schema, setSchema] = useState<ObjectType>();
	const [isAltering, setIsAltering] = useState<boolean>(false);
	const [hasLoadedAlterStatus, setHasLoadedAlterStatus] = useState<boolean>(false);
	const [latestAlterError, setLatestAlterError] = useState<string>();

	const { api, handleError, selectedWorkspace, setWorkspaces } = useContext(AppContext);
	const [consentPurposesState, setConsentPurposesState] = useState<ConsentPurposesState>({
		workspace: selectedWorkspace,
		purposes: [],
	});

	const isAlteringRef = useRef<boolean>();

	useEffect(() => {
		isAlteringRef.current = isAltering;
	}, [isAltering]);

	useEffect(() => {
		if (!hasLoadedAlterStatus || !isLoadingSchema || isAltering) {
			return;
		}
		let active = true;
		let timeoutID: number;
		const fetchSchema = async () => {
			try {
				const schema = await api.workspaces.profileSchema();
				if (!active) {
					return;
				}
				const workspaces = await api.workspaces.list();
				if (!active) {
					return;
				}
				setWorkspaces(workspaces);
				setSchema(schema);
				timeoutID = window.setTimeout(() => setIsLoadingSchema(false), 300);
			} catch (err) {
				if (!active) {
					return;
				}
				handleError(err);
				timeoutID = window.setTimeout(fetchSchema, 3000);
			}
		};
		fetchSchema();
		return () => {
			active = false;
			window.clearTimeout(timeoutID);
		};
	}, [selectedWorkspace, isLoadingSchema, isAltering, hasLoadedAlterStatus]);

	useEffect(() => {
		let active = true;
		let requestSequence = 0;
		let purposesKey: string;
		let reportedError = false;

		const fetchConsentPurposes = async () => {
			const sequence = ++requestSequence;
			let res: ConsentPurposesResponse;
			try {
				res = await api.workspaces.consentPurposes();
			} catch (err) {
				if (!active || sequence !== requestSequence || reportedError) {
					return;
				}
				reportedError = true;
				handleError(err);
				return;
			}
			if (!active || sequence !== requestSequence) {
				return;
			}
			reportedError = false;
			const nextPurposesKey = JSON.stringify(res.purposes);
			if (nextPurposesKey === purposesKey) {
				return;
			}
			purposesKey = nextPurposesKey;
			setConsentPurposesState({ workspace: selectedWorkspace, purposes: res.purposes });
		};

		const handleVisibilityChange = () => {
			if (!document.hidden) {
				fetchConsentPurposes();
			}
		};

		fetchConsentPurposes();
		const intervalID = window.setInterval(handleVisibilityChange, CONSENT_PURPOSES_REFRESH_INTERVAL);
		document.addEventListener('visibilitychange', handleVisibilityChange);

		return () => {
			active = false;
			window.clearInterval(intervalID);
			document.removeEventListener('visibilitychange', handleVisibilityChange);
		};
	}, [selectedWorkspace]);

	useEffect(() => {
		let isPolling = false;
		const pollSchemaAltering = async () => {
			if (isPolling) {
				return;
			}
			isPolling = true;
			try {
				await handleSchemaAltering();
			} finally {
				isPolling = false;
			}
		};
		const intervalID = setInterval(pollSchemaAltering, 3000);

		pollSchemaAltering();

		return () => {
			clearInterval(intervalID);
		};
	}, []);

	const handleSchemaAltering = async () => {
		let res: LatestAlterProfileSchema;
		try {
			res = await api.workspaces.LatestAlterProfileSchema();
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
			// Refresh the schema and workspace before allowing further edits.
			setIsLoadingSchema(true);
			setIsAltering(false);
		}
		setLatestAlterError(res.error);
		setHasLoadedAlterStatus(true);
	};

	return {
		consentPurposes: consentPurposesState.workspace === selectedWorkspace ? consentPurposesState.purposes : [],
		isLoadingSchema,
		setIsLoadingSchema,
		schema,
		isAltering,
		setIsAltering,
		latestAlterError,
	};
};

export { useSchema };
