import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { UserEventsResponse, UserIdentitiesResponse, userTraitsResponse } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserTab } from './Users.types';
import { UserEvent, UserIdentity } from '../../../lib/api/types/user';

const useUserDrawer = (id: string, selectedTab: UserTab) => {
	const [traits, setTraits] = useState<Record<string, any>>();
	const [events, setEvents] = useState<UserEvent[]>();
	const [identities, setIdentities] = useState<UserIdentity[]>();
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { api, handleError, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchUserTraits = async () => {
			setIsLoading(true);
			// Fetch the user's traits.
			let traitsResponse: userTraitsResponse;
			try {
				traitsResponse = await api.workspaces.users.traits(id);
			} catch (err) {
				setTimeout(() => setIsLoading(false), 200);
				if (err instanceof NotFoundError) {
					handleError('This user does not exist');
					redirect('users');
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
			setTraits(traitsResponse.traits);
			setTimeout(() => setIsLoading(false), 200);
			return;
		};
		if (id === '') {
			return;
		}
		fetchUserTraits();
	}, [id]);

	useEffect(() => {
		const fetchUserTab = async () => {
			if (selectedTab === 'events') {
				setIsLoading(true);
				// Fetch the user's events.
				let eventsResponse: UserEventsResponse;
				try {
					eventsResponse = await api.workspaces.users.events(id);
				} catch (err) {
					setTimeout(() => setIsLoading(false), 200);
					if (err instanceof NotFoundError) {
						handleError('This user does not exist');
						redirect('users');
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
				// Fetch the user's identities.
				let identitiesResponse: UserIdentitiesResponse;
				try {
					identitiesResponse = await api.workspaces.users.identities(id, 0, 1000);
				} catch (err) {
					setTimeout(() => setIsLoading(false), 200);
					if (err instanceof NotFoundError) {
						handleError('This user does not exist');
						redirect('users');
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
		if (id === '') {
			return;
		}
		fetchUserTab();
	}, [id, selectedTab]);

	return { isLoading, traits, events, identities };
};

export { useUserDrawer };
