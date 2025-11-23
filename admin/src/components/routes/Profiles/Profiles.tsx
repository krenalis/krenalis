import React, { useContext, useLayoutEffect } from 'react';
import './Profiles.css';
import AppContext from '../../../context/AppContext';
import profilesContext from '../../../context/ProfilesContext';
import { ProfilesList } from './ProfilesList';

import { useProfiles } from './useProfiles';

const Profiles = () => {
	const { setTitle } = useContext(AppContext);

	const { profiles, profilesTotal, profilesProperties, isLoading, profileIDList, fetchProfiles } = useProfiles();

	useLayoutEffect(() => {
		setTitle('Profiles');
	}, []);

	return (
		<profilesContext.Provider
			value={{
				profiles: profiles,
				profilesTotal: profilesTotal,
				profilesProperties: profilesProperties,
				isLoading,
				profileIDList: profileIDList,
				fetchProfiles: fetchProfiles,
			}}
		>
			<ProfilesList />
		</profilesContext.Provider>
	);
};

export { Profiles };
