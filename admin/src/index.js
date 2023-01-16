import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import Login from './routes/Login/Login';
import NavigationWrapper from './routes/NavigationWrapper/NavigationWrapper';
import ConnectionsWrapper from './routes/ConnectionsWrapper/ConnectionsWrapper';
import ConnectorsList from './routes/ConnectorsList/ConnectorsList';
import ConnectionAdded from './routes/ConnectionAdded/ConnectionAdded';
import ConnectionsMap from './routes/ConnectionsMap/ConnectionsMap';
import Connection from './routes/Connection/Connection';
import UsersWrapper from './routes/UsersWrapper/UsersWrapper';
import UsersList from './routes/UsersList/UsersList';
import NotFound from './routes/NotFound/NotFound';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
	<BrowserRouter>
		<Routes>
			<Route path='/admin/' element={<App />}>
				<Route index element={<Login />} />
				<Route element={<NavigationWrapper />}>
					<Route element={<ConnectionsWrapper />}>
						<Route path='connectors/added/:id' element={<ConnectionAdded />} />
						<Route path='connectors' element={<ConnectorsList />} />
						<Route path='connections/:id' element={<Connection />} />
						<Route path='connections' element={<ConnectionsMap />} />
					</Route>
					<Route element={<UsersWrapper />}>
						<Route path='users' element={<UsersList />} />
					</Route>
					<Route path='*' element={<NotFound />} />
				</Route>
			</Route>
		</Routes>
	</BrowserRouter>
);
