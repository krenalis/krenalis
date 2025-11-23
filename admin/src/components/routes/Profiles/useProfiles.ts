import { useEffect, useContext, useState } from 'react';
import AppContext from '../../../context/AppContext';
import { UI_BASE_PATH } from '../../../constants/paths';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ProfileProperty } from './Profiles.types';
import { ObjectType } from '../../../lib/api/types/types';
import { FindProfilesResponse, ResponseProfile } from '../../../lib/api/types/responses';
import { flattenSchema } from '../../../lib/core/action';
import { PROFILES_PROPERTIES_KEY } from '../../../constants/storage';

const DEFAULT_PROFILE_LIMIT = 1000;

const useProfiles = () => {
	const [profiles, setProfiles] = useState<ResponseProfile[]>([]);
	const [profilesTotal, setProfilesTotal] = useState<number>(0);
	const [profilesProperties, setProfilesProperties] = useState<ProfileProperty[]>([]);
	const [profileIDList, setProfileIDList] = useState<string[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { api, handleError, redirect, selectedWorkspace, warehouse } = useContext(AppContext);

	useEffect(() => {
		if (warehouse == null) {
			// a workspace without a connected data warehouse cannot show
			// warehouse profiles.
			redirect('settings');
			handleError('Please connect to a data warehouse before proceeding');
			return;
		}
		// on mount, fetch the first page of profiles.
		fetchProfiles();
	}, [selectedWorkspace]);

	const fetchProfiles = async (): Promise<string[]> => {
		setIsLoading(true);

		// fetch the profile schema.
		let schema: ObjectType;
		try {
			schema = await api.workspaces.profileSchema();
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
				handleError(err);
			}, 300);
			return;
		}

		// check if previous profiles properties are already saved in the storage.
		const storageProperties = localStorage.getItem(PROFILES_PROPERTIES_KEY);
		let preferences: ProfileProperty[] = [];
		if (storageProperties != null) {
			try {
				preferences = JSON.parse(storageProperties);
			} catch (err) {
				// the value of the properties in the storage is corrupted.
				localStorage.removeItem(PROFILES_PROPERTIES_KEY);
			}
		}

		const flatSchema = flattenSchema(schema);
		const paths = Object.keys(flatSchema);

		// compute the properties to show in the table columns and in
		// the “Toggle columns” menu, and those that should be requested
		// to the server.
		const toShow: ProfileProperty[] = [];
		const toFetch: string[] = [];
		for (const path of paths) {
			const isFirstLevel = !path.includes('.');
			if (isFirstLevel) {
				// fetch all the profiles properties by passing all the
				// first level properties to the server.
				toFetch.push(path);
			}

			let isParent = false;
			const depth = path.split('.').length;
			for (const p of paths) {
				const isSameProperty = p === path;
				if (isSameProperty) {
					continue;
				}
				const isChildren = p.includes('.');
				if (isChildren) {
					const parts = p.split('.');
					const prefix = parts.slice(0, depth).join('.');
					if (prefix === path) {
						isParent = true;
						continue;
					}
				}
			}

			if (isParent) {
				// show only flattened subproperties instead of full
				// parent properties (e.g. `obj.prop.prop2` instead of
				// `obj`).
				continue;
			}

			// check if there is a preference for the property.
			const preference = preferences.find((prop) => prop.name === path);

			let isTypeChanged = false;
			if (preference != null) {
				isTypeChanged = preference.type !== flatSchema[path].type;
			}

			toShow.push({
				name: path,
				isUsed: preference != null && !isTypeChanged ? preference.isUsed : true,
				type: flatSchema[path].type,
			});
		}
		setProfilesProperties(toShow);

		// update the value of the properties in the storage.
		localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(toShow));

		// fetch the profiles.
		let res: FindProfilesResponse;
		try {
			res = await api.workspaces.profiles.find(toFetch, null, '', true, 0, DEFAULT_PROFILE_LIMIT);
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
				if (err instanceof NotFoundError) {
					redirect(UI_BASE_PATH);
					handleError('The workspace does not exist anymore');
					return;
				}
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'PropertyNotExist':
							// one of the properties has been concurrently
							// removed from the profile schema. Try again.
							fetchProfiles();
							return;
						case 'DataWarehouseFailed':
							handleError('An error occurred with the data warehouse');
							return;
					}
				}
				handleError(err);
			}, 300);
			return;
		}

		const profiles = res.profiles;
		const total = res.total;

		setProfiles(profiles);
		setProfilesTotal(total);

		// compute the list of profiles MPIDs needed for navigating between profiles.
		const mpids: string[] = [];
		for (const profile of profiles) {
			mpids.push(profile.mpid);
		}
		setProfileIDList(mpids);

		setTimeout(() => {
			setIsLoading(false);
		}, 300);

		return mpids;
	};

	return {
		profiles: profiles,
		profilesTotal: profilesTotal,
		profilesProperties: profilesProperties,
		isLoading,
		profileIDList: profileIDList,
		fetchProfiles: fetchProfiles,
	};
};

export { useProfiles };
