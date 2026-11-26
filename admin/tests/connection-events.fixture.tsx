// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import ConnectionEvents from '../src/components/routes/ConnectionEvents/ConnectionEvents';
import ConnectionWrapper from '../src/components/routes/ConnectionWrapper/ConnectionWrapper';
import AppContext from '../src/context/AppContext';
import API from '../src/lib/api/api';
import type TransformedConnection from '../src/lib/core/connection';

interface FixtureConnection extends Pick<TransformedConnection, 'id' | 'name' | 'role' | 'status' | 'isSDK'> {
	connector: Pick<TransformedConnection['connector'], 'code'>;
}

const connections = ['A', 'B'].map<FixtureConnection>((id) => ({
	id,
	name: id,
	role: 'Source',
	status: { text: 'Working properly', variant: 'success' },
	connector: { code: 'javascript' },
	isSDK: true,
}));
const context = {
	api: new API(location.origin, 'workspace'),
	connections,
	publicMetadata: { externalAssetsURLs: [] },
	setTitle: () => {},
} as unknown as React.ContextType<typeof AppContext>;

createRoot(document.getElementById('root')!).render(
	<AppContext.Provider value={context}>
		<BrowserRouter>
			<Routes>
				<Route path='/admin/connections/:id' element={<ConnectionWrapper />}>
					<Route path='events' element={<ConnectionEvents />} />
				</Route>
			</Routes>
		</BrowserRouter>
	</AppContext.Provider>,
);
