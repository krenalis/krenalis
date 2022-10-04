import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import App from './App';
import Login from './routes/Login/Login'
import PrivateWrapper from './routes/PrivateWrapper/PrivateWrapper'
import Dashboard from './routes/Dashboard/Dashboard'
import ConnectorsList from './routes/ConnectorsList/ConnectorsList'
import ConnectorConfirmation from './routes/ConnectorConfirmation/ConnectorConfirmation'
import AccountConnectors from './routes/AccountConnectors/AccountConnectors';
import SchemaSettings from './routes/SchemaSettings/SchemaSettings';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path='/admin/' element={<App />} >
        <Route index element={<Login />} />
        <Route element={<PrivateWrapper />} >
          <Route path='dashboard' element={<Dashboard />} />
          <Route path='connectors/confirmation/:id' element={<ConnectorConfirmation />} />
          <Route path='connectors' element={<ConnectorsList />} />
          <Route path='account/connectors' element={<AccountConnectors />} />
          <Route path='schema' element={<SchemaSettings />} />
        </Route>
      </Route>
    </Routes>
  </BrowserRouter>
);
