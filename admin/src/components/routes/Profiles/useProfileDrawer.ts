import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { ProfileEventsResponse, IdentitiesResponse, profileAttributesResponse } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ObjectType } from '../../../lib/api/types/types';
import { ProfileTab } from './Profiles.types';
import { ProfileEvent, Identity, ProfileAttributes } from '../../../lib/api/types/profile';

interface ProfileResource<T> {
	key: string;
	value: T;
}

const useProfileDrawer = (
	kpid: string,
	selectedTab: ProfileTab,
	schema: ObjectType | undefined,
	profilesQueryKey: string,
	onSchemaNotAligned: () => void,
) => {
	const [attributesResource, setAttributesResource] = useState<ProfileResource<ProfileAttributes>>();
	const [eventsResource, setEventsResource] = useState<ProfileResource<ProfileEvent[]>>();
	const [identitiesResource, setIdentitiesResource] = useState<ProfileResource<Identity[]>>();
	const [attributesLoadingKey, setAttributesLoadingKey] = useState('');
	const [missingProfileKey, setMissingProfileKey] = useState('');
	const [eventsLoadingKey, setEventsLoadingKey] = useState('');
	const [identitiesLoadingKey, setIdentitiesLoadingKey] = useState('');

	const { api, handleError, selectedWorkspace } = useContext(AppContext);
	const attributesKey = kpid === '' || schema == null ? '' : `${selectedWorkspace}:${profilesQueryKey}:${kpid}`;
	const auxiliaryKey = attributesKey;
	const attributes = attributesResource?.key === attributesKey ? attributesResource.value : undefined;
	const events = selectedTab === 'events' && eventsResource?.key === auxiliaryKey ? eventsResource.value : undefined;
	const identities =
		selectedTab === 'identities' && identitiesResource?.key === auxiliaryKey ? identitiesResource.value : undefined;
	const isAttributesLoading = attributesKey !== '' && attributesLoadingKey === attributesKey;
	const isEventsLoading = selectedTab === 'events' && eventsLoadingKey === auxiliaryKey;
	const isIdentitiesLoading = selectedTab === 'identities' && identitiesLoadingKey === auxiliaryKey;

	useEffect(() => {
		setAttributesResource(undefined);
		setMissingProfileKey('');
		if (attributesKey === '' || schema == null) {
			return;
		}

		const controller = new AbortController();
		setAttributesLoadingKey(attributesKey);
		void api.workspaces.profiles
			.attributes(kpid, schema, controller.signal)
			.then((response: profileAttributesResponse) => {
				if (controller.signal.aborted) {
					return;
				}
				setAttributesResource({ key: attributesKey, value: response.attributes });
			})
			.catch((error) => {
				if (controller.signal.aborted || error?.name === 'AbortError') {
					return;
				}
				if (error instanceof UnprocessableError && error.code === 'SchemaNotAligned') {
					onSchemaNotAligned();
					return;
				}
				if (error instanceof NotFoundError) {
					setMissingProfileKey(attributesKey);
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
	}, [api, attributesKey, schema, handleError, kpid, onSchemaNotAligned]);

	useEffect(() => {
		if (kpid === '' || schema == null || (selectedTab !== 'events' && selectedTab !== 'identities')) {
			return;
		}

		const controller = new AbortController();
		if (selectedTab === 'events') {
			setEventsLoadingKey(auxiliaryKey);
			void api.workspaces.profiles
				.events(kpid, controller.signal)
				.then((response: ProfileEventsResponse) => {
					if (!controller.signal.aborted) {
						setEventsResource({ key: auxiliaryKey, value: response.events });
					}
				})
				.catch((error) => handleAuxiliaryError(error, controller.signal, handleError))
				.finally(() => {
					if (!controller.signal.aborted) {
						setEventsLoadingKey((key) => (key === auxiliaryKey ? '' : key));
					}
				});
		} else {
			setIdentitiesLoadingKey(auxiliaryKey);
			void api.workspaces.profiles
				.identities(kpid, 0, 1000, controller.signal)
				.then((response: IdentitiesResponse) => {
					if (!controller.signal.aborted) {
						setIdentitiesResource({ key: auxiliaryKey, value: response.identities });
					}
				})
				.catch((error) => handleAuxiliaryError(error, controller.signal, handleError))
				.finally(() => {
					if (!controller.signal.aborted) {
						setIdentitiesLoadingKey((key) => (key === auxiliaryKey ? '' : key));
					}
				});
		}

		return () => controller.abort();
	}, [api, auxiliaryKey, schema, handleError, kpid, onSchemaNotAligned, selectedTab]);

	return {
		attributes,
		events,
		identities,
		isAttributesLoading,
		isProfileMissing: attributesKey !== '' && missingProfileKey === attributesKey,
		isEventsLoading,
		isIdentitiesLoading,
	};
};

const handleAuxiliaryError = (error: any, signal: AbortSignal, handleError: (error: Error | string) => void) => {
	if (signal.aborted || error?.name === 'AbortError') {
		return;
	}
	if (error instanceof NotFoundError) {
		handleError('This profile does not exist');
		return;
	}
	handleError(error);
};

export { useProfileDrawer };
