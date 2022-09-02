import React, { Component } from 'react';
import Logo from '../Logo/Logo';
import SidebarItem from '../SidebarItem/SidebarItem';
import './Sidebar.css';

// TODO(@Andrea): add the 'isOpen' state to toggle when the user wants to
// minimize or see the full-width sidebar.

export default class Sidebar extends Component {
    render() {
        return (
            <div className='Sidebar'>
                <Logo />
                <div className="Items">
                    <div className='Top'>
                        <SidebarItem link='' icon='home' title='' />
                        <SidebarItem href='/admin/public/visualization' icon='dashboard' title='' />
                        <SidebarItem link='dashboard' icon='space_dashboard' title='' />
                    </div>
                    <div className='Bottom'>
                        <SidebarItem link='' icon='settings' title='' />
                        <SidebarItem link='' icon='logout' title='' />
                    </div>
                </div>
            </div>
        )
    }
}
