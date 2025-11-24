import React, { useContext, useLayoutEffect } from 'react';
import './Profiles.css';
import AppContext from '../../../context/AppContext';
import ProfilesContext from '../../../context/ProfilesContext';
import { ProfilesList } from './ProfilesList';

import { useProfiles } from './useProfiles';

const Profiles = () => {
	const { setTitle } = useContext(AppContext);

	const { profiles, profilesTotal, profilesProperties, isLoading, profileIDList, fetchProfiles } = useProfiles();

	useLayoutEffect(() => {
		setTitle('Profiles');
	}, []);

	return (
		<ProfilesContext.Provider
			value={{
				profiles,
				profilesTotal,
				profilesProperties,
				isLoading,
				profileIDList,
				fetchProfiles,
			}}
		>
			<ProfilesList />
		</ProfilesContext.Provider>
	);
};

export { Profiles };
