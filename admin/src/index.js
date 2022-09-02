import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import App from './App';
import Home from './routes/Home/Home'
import Dashboard from './routes/Dashboard/Dashboard'

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <BrowserRouter>
    <Routes>
      <Route path='/admin/public/' element={<App />} >
        <Route path='' element={<Home />} />
        <Route path='dashboard' element={<Dashboard />} />
      </Route>
    </Routes>
  </BrowserRouter>
);
