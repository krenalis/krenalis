// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

import React, { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Route, Routes, useNavigate } from 'react-router-dom';
import ConnectionEvents from '../src/components/routes/ConnectionEvents/ConnectionEvents';
import ConnectionWrapper from '../src/components/routes/ConnectionWrapper/ConnectionWrapper';
import { UI_BASE_PATH } from '../src/constants/paths';
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
const api = new API(location.origin, 'workspace');

const Fixture = () => {
	const [title, setTitle] = useState<React.ReactNode>('');
	const [error, setError] = useState('');
	const navigate = useNavigate();
	const context = {
		api,
		connections,
		publicMetadata: { externalAssetsURLs: [] },
		setTitle,
		redirect: (path: string) => navigate(UI_BASE_PATH + path),
		handleError: (error: Error | string) => setError(String(error)),
	} as unknown as React.ContextType<typeof AppContext>;
	return (
		<AppContext.Provider value={context}>
			<header>{title}</header>
			{error !== '' && <div role='alert'>{error}</div>}
			<Routes>
				<Route path='/admin/connections/:id' element={<ConnectionWrapper />}>
					<Route path='events' element={<ConnectionEvents />} />
				</Route>
			</Routes>
		</AppContext.Provider>
	);
};

createRoot(document.getElementById('root')!).render(
	<BrowserRouter>
		<Fixture />
	</BrowserRouter>,
);
