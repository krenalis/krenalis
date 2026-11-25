import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import ConnectionEvents from '../src/components/routes/ConnectionEvents/ConnectionEvents';
import ConnectionWrapper from '../src/components/routes/ConnectionWrapper/ConnectionWrapper';
import AppContext from '../src/context/AppContext';
import API from '../src/lib/api/api';

const connections = ['A', 'B'].map((id) => ({
	id,
	name: id,
	role: 'Source',
	status: 'active',
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
