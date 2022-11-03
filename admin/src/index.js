import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import Login from './routes/Login/Login';
import PrivateWrapper from './routes/PrivateWrapper/PrivateWrapper';
import Connectors from './routes/Connectors/Connectors/Connectors';
import ConnectorsSourceAdded from './routes/Connectors/ConnectorsSourceAdded/ConnectorsSourceAdded';
import AccountSources from './routes/Account/AccountSources/AccountSources';
import AccountSourceProperties from './routes/Account/AccountSourceProperties/AccountSourceProperties';
import AccountSourceSQL from './routes/Account/AccountSourceSQL/AccountSourceSQL';
import AccountSourceSettings from './routes/Account/AccountSourceSettings/AccountSourceSettings';
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
          <Route path='connectors/added/:id' element={<ConnectorsSourceAdded />} />
          <Route path='connectors' element={<Connectors />} />
          <Route path='account/sources/:id/properties' element={<AccountSourceProperties />} />
          <Route path='account/sources/:id/sql' element={<AccountSourceSQL />} />
          <Route path='account/sources/:id/settings' element={<AccountSourceSettings />} />
          <Route path='account/sources' element={<AccountSources />} />
          <Route path='account/schemas' element={<AccountSchemas />} />
          <Route path='*' element={<NotFound />} />
        </Route>
      </Route>
    </Routes>
  </BrowserRouter>
);
