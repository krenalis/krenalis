import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { ProfileEventsResponse, IdentitiesResponse, profileAttributesResponse } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ProfileTab } from './Profiles.types';
import { ProfileEvent, Identity } from '../../../lib/api/types/profile';

const useProfileDrawer = (kpid: string, selectedTab: ProfileTab) => {
	const [attributes, setAttributes] = useState<Record<string, any>>();
	const [events, setEvents] = useState<ProfileEvent[]>();
	const [identities, setIdentities] = useState<Identity[]>();
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { api, handleError, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchProfileAttributes = async () => {
			setIsLoading(true);
			// Fetch the profile's attributes.
			let attributesResponse: profileAttributesResponse;
			try {
				attributesResponse = await api.workspaces.profiles.attributes(kpid);
			} catch (err) {
				setTimeout(() => setIsLoading(false), 200);
				if (err instanceof NotFoundError) {
					handleError('This profile does not exist');
					redirect('profile-unification/profiles');
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'WarehouseError') {
						handleError('An error occurred with the data warehouse');
						return;
					}
				}
				handleError(err);
				return;
			}
			setAttributes(attributesResponse.attributes);
			setTimeout(() => setIsLoading(false), 200);
			return;
		};
		if (kpid === '') {
			return;
		}
		fetchProfileAttributes();
	}, [kpid]);

	useEffect(() => {
		const fetchProfileTab = async () => {
			if (selectedTab === 'events') {
				setIsLoading(true);
				// Fetch the profile's events.
				let eventsResponse: ProfileEventsResponse;
				try {
					eventsResponse = await api.workspaces.profiles.events(kpid);
				} catch (err) {
					setTimeout(() => setIsLoading(false), 200);
					if (err instanceof NotFoundError) {
						handleError('This profile does not exist');
						redirect('profile-unification/profiles');
						return;
					}
					if (err instanceof UnprocessableError) {
						if (err.code === 'WarehouseError') {
							handleError('An error occurred with the data warehouse');
							return;
						}
					}
					handleError(err);
					return;
				}
				setEvents(eventsResponse.events);
				setTimeout(() => setIsLoading(false), 200);
				return;
			}

			if (selectedTab === 'identities') {
				setIsLoading(true);
				// Fetch the profile's identities.
				let identitiesResponse: IdentitiesResponse;
				try {
					identitiesResponse = await api.workspaces.profiles.identities(kpid, 0, 1000);
				} catch (err) {
					setTimeout(() => setIsLoading(false), 200);
					if (err instanceof NotFoundError) {
						handleError('This profile does not exist');
						redirect('profile-unification/profiles');
						return;
					}
					if (err instanceof UnprocessableError) {
						if (err.code === 'WarehouseError') {
							handleError('An error occurred with the data warehouse');
							return;
						}
					}
					handleError(err);
					return;
				}
				setIdentities(identitiesResponse.identities);
				setTimeout(() => setIsLoading(false), 200);
				return;
			}
		};
		if (kpid === '') {
			return;
		}
		fetchProfileTab();
	}, [kpid, selectedTab]);

	return { isLoading, attributes, events, identities };
};

export { useProfileDrawer };
