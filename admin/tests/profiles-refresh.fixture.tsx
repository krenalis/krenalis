// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import React, { useCallback, useState } from 'react';
import ReactDOM from 'react-dom/client';
import '@shoelace-style/shoelace/dist/themes/light.css';
import API from '../src/lib/api/api';
import Workspace from '../src/lib/api/types/workspace';
import AppContext from '../src/context/AppContext';
import { Profiles } from '../src/components/routes/Profiles/Profiles';

const workspace: Workspace = {
	id: 'workspace111',
	name: 'Test workspace',
	profileSchema: {
		schema: { kind: 'object', properties: [] },
		propertyRoles: { firstName: '', lastName: '', email: '', country: '', photo: '' },
		primarySources: {},
	},
	resolveIdentitiesOnBatchImport: false,
	identifiers: [],
	warehouseMode: 'Normal',
};
const api = new API('https://example.test', workspace.id);

const Fixture = () => {
	const [error, setError] = useState('');
	const handleError = useCallback((error: Error | string) => setError(String(error)), []);
	const appContext = {
		api,
		connections: [],
		workspaces: [workspace],
		selectedWorkspace: workspace.id,
		warehouse: {},
		handleError,
		setTitle: () => {},
	} as unknown as React.ContextType<typeof AppContext>;

	return (
		<AppContext.Provider value={appContext}>
			<Profiles />
			<p role='alert'>{error}</p>
		</AppContext.Provider>
	);
};

ReactDOM.createRoot(document.getElementById('root')!).render(<Fixture />);
