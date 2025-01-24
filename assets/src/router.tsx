import React from 'react';
import { UI_BASE_PATH } from './constants/paths';
import App from './components/routes/App/App';
import AppLayout from './components/routes/AppLayout/AppLayout';
import Login from './components/routes/Login/Login';
import ConnectorsList from './components/routes/ConnectorsList/ConnectorsList';
import ConnectorSettings from './components/routes/ConnectorSettings/ConnectorSettings';
import ConnectionsMap from './components/routes/ConnectionsMap/ConnectionsMap';
import ConnectionsList from './components/routes/ConnectionsList/ConnectionsList';
import ConnectionWrapper from './components/routes/ConnectionWrapper/ConnectionWrapper';
import RootError from './components/routes/RootError/RootError';
import { Users } from './components/routes/Users/Users';
import SchemaGrid from './components/routes/SchemaGrid/SchemaGrid';
import OAuth from './components/routes/OAuth/OAuth';
import NotFound from './components/routes/NotFound/NotFound';
import ConnectionOverview from './components/routes/ConnectionOverview/ConnectionOverview';
import ConnectionEvents from './components/routes/ConnectionEvents/ConnectionEvents';
import ConnectionActions from './components/routes/ConnectionActions/ConnectionActions';
import ActionWrapper from './components/routes/ActionWrapper/ActionWrapper';
import ConnectionSettings from './components/routes/ConnectionSettings/ConnectionSettings';
import { ConnectionIdentities } from './components/routes/ConnectionIdentities/ConnectionIdentities';
import IdentityResolutionSettings from './components/routes/IdentityResolutionSettings/IdentityResolutionSettings';
import { createBrowserRouter } from 'react-router-dom';
import DataWarehouse from './components/routes/DataWarehouse/DataWarehouse';
import GeneralSettings from './components/routes/GeneralSettings/GeneralSettings';
import Settings from './components/routes/Settings/Settings';
import Members from './components/routes/Members/Members';
import Member from './components/routes/Member/Member';
import Organization from './components/routes/Organization/Organization';
import Workspaces from './components/routes/Workspaces/Workspaces';
import SignUp from './components/routes/SignUp/SignUp';
import { FileConnector } from './components/routes/FileConnector/FileConnector';
import { Schema } from './components/routes/Schema/Schema';
import { SchemaEditWrapper } from './components/routes/SchemaEdit/SchemaEditWrapper';
import { WorkspaceCreate } from './components/routes/WorkspaceCreate/WorkspaceCreate';
import { WorkspacesWrapper } from './components/routes/WorkspacesWrapper/WorkspacesWrapper';
import { APIKeys } from './components/routes/APIKeys/APIKeys';

const router = createBrowserRouter([
	{
		path: UI_BASE_PATH,
		element: <App />,
		errorElement: <RootError />,
		children: [
			{ path: '', element: <Login /> },
			{ path: 'sign-up/:token', element: <SignUp /> },
			{
				path: 'workspaces',
				element: <WorkspacesWrapper />,
				children: [
					{
						path: '',
						element: <Workspaces />,
					},
					{
						path: 'add',
						element: <WorkspaceCreate />,
					},
				],
			},
			{
				element: <AppLayout />,
				children: [
					{ path: 'connectors/:name', element: <ConnectorSettings /> },
					{ path: 'connectors/file/:name', element: <FileConnector /> },
					{ path: 'connectors', element: <ConnectorsList /> },
					{ path: 'connections/sources', element: <ConnectionsList /> },
					{ path: 'connections/destinations', element: <ConnectionsList /> },
					{
						path: 'connections/:id',
						element: <ConnectionWrapper />,
						children: [
							{
								path: 'actions',
								element: <ConnectionActions />,
								children: [
									{ path: 'edit/:action', element: <ActionWrapper /> },
									{ path: 'add/event/:eventType', element: <ActionWrapper /> },
									{ path: 'add/event', element: <ActionWrapper /> },
									{ path: 'add/:actionTarget', element: <ActionWrapper /> },
								],
							},
							{ path: 'overview', element: <ConnectionOverview /> },
							{ path: 'events', element: <ConnectionEvents /> },
							{ path: 'settings', element: <ConnectionSettings /> },
							{ path: 'identities', element: <ConnectionIdentities /> },
						],
					},
					{ path: 'connections', element: <ConnectionsMap /> },
					{ path: 'oauth/authorize', element: <OAuth /> },
					{ path: 'users', element: <Users /> },
					{
						path: 'schema',
						element: <Schema />,
						children: [
							{
								path: '',
								element: <SchemaGrid />,
								children: [
									{
										path: 'edit',
										element: <SchemaEditWrapper />,
									},
								],
							},
						],
					},
					{
						path: 'settings',
						element: <Settings />,
						children: [
							{
								path: 'general',
								element: <GeneralSettings />,
							},
							{
								path: 'identity-resolution',
								element: <IdentityResolutionSettings />,
							},
							{
								path: 'data-warehouse',
								element: <DataWarehouse />,
							},
						],
					},
					{
						path: 'organization',
						element: <Organization />,
						children: [
							{
								path: 'members/current',
								element: <Member />,
							},
							{
								path: 'members',
								element: <Members />,
							},
							{
								path: 'api-keys',
								element: <APIKeys />,
							},
						],
					},
					{ path: '*', element: <NotFound /> },
				],
			},
		],
	},
]);

export default router;
