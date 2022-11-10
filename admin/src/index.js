import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import Login from './routes/Login/Login';
import PrivateWrapper from './routes/PrivateWrapper/PrivateWrapper';
import Connectors from './routes/Connectors/Connectors/Connectors';
import ConnectorsConnectionAdded from './routes/Connectors/ConnectorsConnectionAdded/ConnectorsConnectionAdded';
import AccountConnections from './routes/Account/AccountConnections/AccountConnections';
import AccountConnectionProperties from './routes/Account/AccountConnectionProperties/AccountConnectionProperties';
import AccountConnectionSQL from './routes/Account/AccountConnectionSQL/AccountConnectionSQL';
import AccountConnectionSettings from './routes/Account/AccountConnectionSettings/AccountConnectionSettings';
import AccountSchemas from './routes/Account/AccountSchemas/AccountSchemas';
import NotFound from './routes/NotFound/NotFound';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path='/admin/' element={<App />} >
        <Route index element={<Login />} />
        <Route element={<PrivateWrapper />} >
          <Route path='connectors/added/:id' element={<ConnectorsConnectionAdded />} />
          <Route path='connectors' element={<Connectors />} />
          <Route path='account/connections/:id/properties' element={<AccountConnectionProperties />} />
          <Route path='account/connections/:id/sql' element={<AccountConnectionSQL />} />
          <Route path='account/connections/:id/settings' element={<AccountConnectionSettings />} />
          <Route path='account/connections' element={<AccountConnections />} />
          <Route path='account/schemas' element={<AccountSchemas />} />
          <Route path='*' element={<NotFound />} />
        </Route>
      </Route>
    </Routes>
  </BrowserRouter>
);
