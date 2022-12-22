import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import Login from './routes/Login/Login';
import PrivateWrapper from './routes/PrivateWrapper/PrivateWrapper';
import ConnectorsList from './routes/ConnectorsList/ConnectorsList';
import ConnectionAdded from './routes/ConnectionAdded/ConnectionAdded';
import ConnectionsList from './routes/ConnectionsList/ConnectionsList';
import ConnectionsMap from './routes/ConnectionsMap/ConnectionsMap';
import ConnectionProperties from './routes/ConnectionProperties/ConnectionProperties';
import ConnectionSQL from './routes/ConnectionSQL/ConnectionSQL';
import ConnectionSettings from './routes/ConnectionSettings/ConnectionSettings';
import Schemas from './routes/Schemas/Schemas';
import UsersList from './routes/UsersList/UsersList';
import NotFound from './routes/NotFound/NotFound';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
	<BrowserRouter>
		<Routes>
			<Route path='/admin/' element={<App />}>
				<Route index element={<Login />} />
				<Route element={<PrivateWrapper />}>
					<Route path='connectors/added/:id' element={<ConnectionAdded />} />
					<Route path='connectors' element={<ConnectorsList />} />
					<Route path='connections/:id/properties' element={<ConnectionProperties />} />
					<Route path='connections/:id/sql' element={<ConnectionSQL />} />
					<Route path='connections/:id/settings' element={<ConnectionSettings />} />
					<Route path='connections-map' element={<ConnectionsMap />} />
					<Route path='connections' element={<ConnectionsList />} />
					<Route path='schemas' element={<Schemas />} />
					<Route path='users' element={<UsersList />} />
					<Route path='*' element={<NotFound />} />
				</Route>
			</Route>
		</Routes>
	</BrowserRouter>
);
