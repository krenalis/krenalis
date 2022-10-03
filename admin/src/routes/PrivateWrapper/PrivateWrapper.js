import React, { Component } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from '../../components/Sidebar/Sidebar'
import './PrivateWrapper.css'

export default class PrivateWrapper extends Component {
    render() {
        return (
            <div>
                <Sidebar />
                <Outlet />
            </div>
        )
    }
}
