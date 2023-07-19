import { adminBasePath } from './constants/path';
import App from './App';
import ConnectorsList from './components/routes/ConnectorsList/ConnectorsList';
import ConnectorSettings from './components/routes/ConnectorSettings/ConnectorSettings';
import ConnectionsMap from './components/routes/ConnectionsMap/ConnectionsMap';
import ConnectionsList from './components/routes/ConnectionsList/ConnectionsList';
import ConnectionWrapper from './components/routes/ConnectionWrapper/ConnectionWrapper';
import { ConnectionProvider } from './context/providers/ConnectionProvider';
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
import AnonymousIdentity from './components/routes/AnonymousIdentity/AnonymousIdentity';
import { createBrowserRouter } from 'react-router-dom';

const router = createBrowserRouter([
	{
		path: adminBasePath,
		element: <App />,
		children: [
			{ path: 'connectors/:id', element: <ConnectorSettings /> },
			{ path: 'connectors', element: <ConnectorsList /> },
			{ path: 'connections/sources', element: <ConnectionsList /> },
			{ path: 'connections/destinations', element: <ConnectionsList /> },
			{
				element: <ConnectionProvider />,
				children: [
					{
						path: 'connections/:id',
						element: <ConnectionWrapper />,
						children: [
							{ path: 'actions', element: <ConnectionActions /> },
							{ path: 'overview', element: <ConnectionOverview /> },
							{ path: 'events', element: <ConnectionEvents /> },
							{ path: 'settings', element: <ConnectionSettings /> },
						],
					},
				],
			},
			{ path: 'connections', element: <ConnectionsMap /> },
			{ path: 'oauth/authorize', element: <OAuth /> },
			{
				element: <UsersWrapper />,
				children: [
					{ path: 'users/:id', element: <User /> },
					{ path: 'users', element: <UsersList /> },
				],
			},
			{
				path: 'schema',
				element: <Schema />,
			},
			{
				path: 'anonymous-identity',
				element: <AnonymousIdentity />,
			},
			{ path: '*', element: <NotFound /> },
		],
	},
]);

export default router;
