import React, { useContext, useLayoutEffect } from 'react';
import './Profiles.css';
import AppContext from '../../../context/AppContext';
import ProfilesContext from '../../../context/ProfilesContext';
import { ProfilesList } from './ProfilesList';

import { useProfiles } from './useProfiles';

const Profiles = () => {
	const { setTitle } = useContext(AppContext);
	const profilesContext = useProfiles();

	useLayoutEffect(() => {
		setTitle('Profile Unification / Profiles');
	}, []);

	return (
		<ProfilesContext.Provider value={profilesContext}>
			<ProfilesList />
		</ProfilesContext.Provider>
	);
};

export { Profiles };
