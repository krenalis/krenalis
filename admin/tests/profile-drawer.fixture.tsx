// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import React, { useState } from 'react';
import ReactDOM from 'react-dom/client';
import '@shoelace-style/shoelace/dist/themes/light.css';
import API from '../src/lib/api/api';
import { profileAttributesResponse } from '../src/lib/api/types/responses';
import Workspace from '../src/lib/api/types/workspace';
import AppContext from '../src/context/AppContext';
import ProfilesContext from '../src/context/ProfilesContext';
import { ProfileDrawer } from '../src/components/routes/Profiles/ProfileDrawer';
import '../src/components/routes/Profiles/Profiles.css';
import './profile-drawer.fixture.css';

declare global {
	interface Window {
		profileDrawerAttributes: profileAttributesResponse['attributes'];
	}
}

const PROFILE_ID = '00000000-0000-4000-8000-000000000001';
const workspace: Workspace = {
	id: 'workspace111',
	name: 'Test workspace',
	profileSchema: {
		schema: { kind: 'object', properties: [] },
		primarySources: {},
		propertyRoles: {
			firstName: 'person.given',
			lastName: 'person.family',
			email: 'contact.mail',
			country: '',
			photo: 'media.avatar',
		},
	},
	resolveIdentitiesOnBatchImport: false,
	identifiers: [],
	warehouseMode: 'Normal',
};
const api = new API('https://example.test', workspace.id);
api.workspaces.profiles.attributes = async () => ({ attributes: window.profileDrawerAttributes });

const appContext = {
	api,
	connections: [],
	workspaces: [workspace],
	selectedWorkspace: workspace.id,
	handleError: (error: Error | string) => {
		throw new Error(String(error));
	},
} as unknown as React.ContextType<typeof AppContext>;
const profilesContext: React.ContextType<typeof ProfilesContext> = {
	profiles: [],
	profilesTotal: 1,
	profilesProperties: [],
	isLoading: false,
	profileIDList: [PROFILE_ID],
	fetchProfiles: async () => [PROFILE_ID],
};

const Fixture = () => {
	const [selectedProfile, setSelectedProfile] = useState(PROFILE_ID);
	return (
		<AppContext.Provider value={appContext}>
			<ProfilesContext.Provider value={profilesContext}>
				<div className='profiles-list__grid-container profile-drawer-fixture'>
					<ProfileDrawer selectedProfile={selectedProfile} setSelectedProfile={setSelectedProfile} />
				</div>
			</ProfilesContext.Provider>
		</AppContext.Provider>
	);
};

ReactDOM.createRoot(document.getElementById('root')!).render(<Fixture />);
