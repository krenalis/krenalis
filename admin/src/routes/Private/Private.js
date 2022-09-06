import React, { Component } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from '../../components/Sidebar/Sidebar'
import './Private.css'

export default class Private extends Component {
    render() {
        return (
            <div>
                <Sidebar />
                <Outlet />
            </div>
        )
    }
}
