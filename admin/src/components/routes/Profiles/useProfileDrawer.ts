import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { ProfileEventsResponse, IdentitiesResponse, profileAttributesResponse } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ProfileTab } from './Profiles.types';
import { ProfileEvent, Identity, ProfileAttributes } from '../../../lib/api/types/profile';

const profileAttributesCache = new Map<string, ProfileAttributes>();
const profileAttributesCacheLimit = 200;

interface ProfileResource<T> {
	key: string;
	value: T;
}

const useProfileDrawer = (
	kpid: string,
	selectedTab: ProfileTab,
	datasetVersion: string | undefined,
	profilesQueryKey: string,
	onDatasetStale: () => void,
) => {
	const [attributesResource, setAttributesResource] = useState<ProfileResource<ProfileAttributes>>();
	const [eventsResource, setEventsResource] = useState<ProfileResource<ProfileEvent[]>>();
	const [identitiesResource, setIdentitiesResource] = useState<ProfileResource<Identity[]>>();
	const [attributesLoadingKey, setAttributesLoadingKey] = useState('');
	const [eventsLoadingKey, setEventsLoadingKey] = useState('');
	const [identitiesLoadingKey, setIdentitiesLoadingKey] = useState('');

	const { api, handleError, selectedWorkspace } = useContext(AppContext);
	const attributesKey =
		kpid === '' || datasetVersion == null
			? ''
			: `${selectedWorkspace}:${datasetVersion}:${profilesQueryKey}:${kpid}`;
	const auxiliaryKey = kpid === '' || datasetVersion == null ? '' : `${selectedWorkspace}:${datasetVersion}:${kpid}`;
	const attributes =
		attributesResource?.key === attributesKey
			? attributesResource.value
			: attributesKey === ''
				? undefined
				: profileAttributesCache.get(attributesKey);
	const events = selectedTab === 'events' && eventsResource?.key === auxiliaryKey ? eventsResource.value : undefined;
	const identities =
		selectedTab === 'identities' && identitiesResource?.key === auxiliaryKey ? identitiesResource.value : undefined;
	const isAttributesLoading = attributesKey !== '' && attributesLoadingKey === attributesKey;
	const isEventsLoading = selectedTab === 'events' && eventsLoadingKey === auxiliaryKey;
	const isIdentitiesLoading = selectedTab === 'identities' && identitiesLoadingKey === auxiliaryKey;

	useEffect(() => {
		if (attributesKey === '' || datasetVersion == null) {
			return;
		}

		const cached = profileAttributesCache.get(attributesKey);
		if (cached != null) {
			setAttributesLoadingKey('');
			return;
		}

		const controller = new AbortController();
		setAttributesLoadingKey(attributesKey);
		void api.workspaces.profiles
			.attributes(kpid, datasetVersion, controller.signal)
			.then((response: profileAttributesResponse) => {
				if (controller.signal.aborted) {
					return;
				}
				if (response.datasetVersion !== datasetVersion) {
					onDatasetStale();
					return;
				}
				if (
					!profileAttributesCache.has(attributesKey) &&
					profileAttributesCache.size >= profileAttributesCacheLimit
				) {
					const oldestCacheKey = profileAttributesCache.keys().next().value;
					if (oldestCacheKey != null) {
						profileAttributesCache.delete(oldestCacheKey);
					}
				}
				profileAttributesCache.set(attributesKey, response.attributes);
				setAttributesResource({ key: attributesKey, value: response.attributes });
			})
			.catch((error) => {
				if (error?.name === 'AbortError') {
					return;
				}
				if (error instanceof UnprocessableError && error.code === 'ProfileDatasetVersionMismatch') {
					onDatasetStale();
					return;
				}
				if (error instanceof NotFoundError) {
					handleError('This profile does not exist');
					return;
				}
				handleError(error);
			})
			.finally(() => {
				if (!controller.signal.aborted) {
					setAttributesLoadingKey((key) => (key === attributesKey ? '' : key));
				}
			});

		return () => controller.abort();
	}, [api, attributesKey, datasetVersion, handleError, kpid, onDatasetStale]);

	useEffect(() => {
		if (kpid === '' || datasetVersion == null || (selectedTab !== 'events' && selectedTab !== 'identities')) {
			return;
		}

		// Events and identities are live auxiliary data in V1. Passing the
		// expected dataset version verifies that the profile generation remains
		// current while they are read; it does not make them snapshots.
		const controller = new AbortController();
		if (selectedTab === 'events') {
			setEventsLoadingKey(auxiliaryKey);
			void api.workspaces.profiles
				.events(kpid, datasetVersion, controller.signal)
				.then((response: ProfileEventsResponse) => {
					if (!controller.signal.aborted) {
						setEventsResource({ key: auxiliaryKey, value: response.events });
					}
				})
				.catch((error) => handleAuxiliaryError(error, controller.signal, handleError, onDatasetStale))
				.finally(() => {
					if (!controller.signal.aborted) {
						setEventsLoadingKey((key) => (key === auxiliaryKey ? '' : key));
					}
				});
		} else {
			setIdentitiesLoadingKey(auxiliaryKey);
			void api.workspaces.profiles
				.identities(kpid, 0, 1000, datasetVersion, controller.signal)
				.then((response: IdentitiesResponse) => {
					if (!controller.signal.aborted) {
						setIdentitiesResource({ key: auxiliaryKey, value: response.identities });
					}
				})
				.catch((error) => handleAuxiliaryError(error, controller.signal, handleError, onDatasetStale))
				.finally(() => {
					if (!controller.signal.aborted) {
						setIdentitiesLoadingKey((key) => (key === auxiliaryKey ? '' : key));
					}
				});
		}

		return () => controller.abort();
	}, [api, auxiliaryKey, datasetVersion, handleError, kpid, onDatasetStale, selectedTab]);

	return {
		attributes,
		events,
		identities,
		isAttributesLoading,
		isEventsLoading,
		isIdentitiesLoading,
	};
};

const handleAuxiliaryError = (
	error: any,
	signal: AbortSignal,
	handleError: (error: Error | string) => void,
	onDatasetStale: () => void,
) => {
	if (signal.aborted || error?.name === 'AbortError') {
		return;
	}
	if (error instanceof UnprocessableError && error.code === 'ProfileDatasetVersionMismatch') {
		onDatasetStale();
		return;
	}
	if (error instanceof NotFoundError) {
		handleError('This profile does not exist');
		return;
	}
	handleError(error);
};

export { useProfileDrawer };
