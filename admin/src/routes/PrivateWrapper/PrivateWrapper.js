import React from 'react'
import { Outlet } from 'react-router-dom'

import './PrivateWrapper.css'
import Sidebar from '../../components/Sidebar/Sidebar'

export default class PrivateWrapper extends React.Component {
    render() {
        return (
            <div className='PrivateWrapper'>
                <Sidebar />
                <Outlet />
            </div>
        )
    }
}
