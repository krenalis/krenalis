import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

import App from './App';
import Login from './routes/Login/Login'
import PrivateWrapper from './routes/PrivateWrapper/PrivateWrapper'
import Connectors from './routes/Connectors/Connectors/Connectors'
import ConnectorsConfirmation from './routes/Connectors/ConnectorsConfirmation/ConnectorsConfirmation'
import AccountConnectors from './routes/Account/AccountConnectors/AccountConnectors';
import AccountConnector from './routes/Account/AccountConnector/AccountConnector';
import ConfigurationsSchema from './routes/Configurations/ConfigurationsSchema/ConfigurationsSchema';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path='/admin/' element={<App />} >
        <Route index element={<Login />} />
        <Route element={<PrivateWrapper />} >
          <Route path='connectors/confirmation/:id' element={<ConnectorsConfirmation />} />
          <Route path='connectors' element={<Connectors />} />
          <Route path='account/connectors/:id' element={<AccountConnector />} />
          <Route path='account/connectors' element={<AccountConnectors />} />
          <Route path='configurations/schema' element={<ConfigurationsSchema />} />
        </Route>
      </Route>
    </Routes>
  </BrowserRouter>
);
