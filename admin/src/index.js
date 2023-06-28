import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { adminBasePath } from './constants/path';
import ConnectorsList from './components/routes/ConnectorsList/ConnectorsList';
import ConnectorSettings from './components/routes/ConnectorSettings/ConnectorSettings';
import ConnectionsMap from './components/routes/ConnectionsMap/ConnectionsMap';
import ConnectionsList from './components/routes/ConnectionsList/ConnectionsList';
import ConnectionWrapper from './components/routes/ConnectionWrapper/ConnectionWrapper';
import { ConnectionProvider } from './providers/ConnectionProvider';
import UsersWrapper from './components/routes/UsersWrapper/UsersWrapper';
import UsersList from './components/routes/UsersList/UsersList';
import User from './components/routes/User/User';
import Schema from './components/routes/Schema/Schema';
import OAuth from './components/routes/OAuth/OAuth';
import NotFound from './components/routes/NotFound/NotFound';
import ConnectionOverview from './components/routes/ConnectionOverview/ConnectionOverview';
import ConnectionEvents from './components/routes/ConnectionEvents/ConnectionEvents';
import ConnectionActions from './components/routes/ConnectionActions/ConnectionActions';
import ConnectionSettings from './components/routes/ConnectionSettings/ConnectionSettings';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
	<BrowserRouter>
		<Routes>
			<Route path={adminBasePath} element={<App />}>
				<Route path='connectors/:id' element={<ConnectorSettings />} />
				<Route path='connectors' element={<ConnectorsList />} />
				<Route path='connections/sources' element={<ConnectionsList />}></Route>
				<Route path='connections/destinations' element={<ConnectionsList />}></Route>
				<Route element={<ConnectionProvider />}>
					<Route path='connections/:id' element={<ConnectionWrapper />}>
						<Route path='actions' element={<ConnectionActions />} />
						<Route path='overview' element={<ConnectionOverview />} />
						<Route path='events' element={<ConnectionEvents />} />
						<Route path='settings' element={<ConnectionSettings />} />
					</Route>
				</Route>
				<Route path='connections' element={<ConnectionsMap />} />
				<Route path='oauth/authorize' element={<OAuth />} />
				<Route element={<UsersWrapper />}>
					<Route path='users/:id' element={<User />} />
					<Route path='users' element={<UsersList />} />
				</Route>
				<Route path='schema' element={<Schema />} />
				<Route path='*' element={<NotFound />} />
			</Route>
		</Routes>
	</BrowserRouter>
);
