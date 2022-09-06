import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import App from './App';
import Login from './routes/Login/Login'
import Private from './routes/Private/Private'
import Dashboard from './routes/Dashboard/Dashboard'

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path='/admin/' element={<App />} >
        <Route index element={<Login />} />
        <Route element={<Private />} >
          <Route path='dashboard' element={<Dashboard />} />
        </Route>
      </Route>
    </Routes>
  </BrowserRouter>
);
