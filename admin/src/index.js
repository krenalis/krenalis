import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import Login from './components/Login/Login';
import NavigationWrapper from './components/NavigationWrapper/NavigationWrapper';
import ConnectionsWrapper from './components/ConnectionsWrapper/ConnectionsWrapper';
import ConnectorsList from './components/ConnectorsList/ConnectorsList';
import ConnectorSettings from './components/ConnectorSettings/ConnectorSettings';
import ConnectionsMap from './components/ConnectionsMap/ConnectionsMap';
import ConnectionsList from './components/ConnectionsList/ConnectionsList';
import Connection from './components/Connection/Connection';
import UsersWrapper from './components/UsersWrapper/UsersWrapper';
import UsersList from './components/UsersList/UsersList';
import User from './components/User/User';
import SchemaWrapper from './components/SchemaWrapper/SchemaWrapper';
import Schema from './components/Schema/Schema';
import OAuth from './components/OAuth/OAuth';
import NotFound from './components/NotFound/NotFound';
import ConnectionOverview from './components/ConnectionOverview/ConnectionOverview';
import ConnectionEvents from './components/ConnectionEvents/ConnectionEvents';
import ConnectionActions from './components/ConnectionActions/ConnectionActions';
import ConnectionSettings from './components/ConnectionSettings/ConnectionSettings';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
	<BrowserRouter>
		<Routes>
			<Route path='/admin/' element={<App />}>
				<Route index element={<Login />} />
				<Route element={<NavigationWrapper />}>
					<Route element={<ConnectionsWrapper />}>
						<Route path='connectors/:id' element={<ConnectorSettings />} />
						<Route path='connectors' element={<ConnectorsList />} />
						<Route path='connections/sources' element={<ConnectionsList />}></Route>
						<Route path='connections/destinations' element={<ConnectionsList />}></Route>
						<Route path='connections/:id' element={<Connection />}>
							<Route path='actions' element={<ConnectionActions />} />
							<Route path='overview' element={<ConnectionOverview />} />
							<Route path='events' element={<ConnectionEvents />} />
							<Route path='settings' element={<ConnectionSettings />} />
						</Route>
						<Route path='connections' element={<ConnectionsMap />} />
						<Route path='oauth/authorize' element={<OAuth />} />
					</Route>
					<Route element={<UsersWrapper />}>
						<Route path='users/:id' element={<User />} />
						<Route path='users' element={<UsersList />} />
					</Route>
					<Route element={<SchemaWrapper />}>
						<Route path='schema' element={<Schema />} />
					</Route>
					<Route path='*' element={<NotFound />} />
				</Route>
			</Route>
		</Routes>
	</BrowserRouter>
);
