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
                        <SidebarItem link='dashboard' icon='space_dashboard' title='' />
                        <SidebarItem link='connectors' icon='list_alt' title='' />
                        <SidebarItem link='account/connectors' icon='account_circle' title='' />
                        <SidebarItem link='schema' icon='schema' title='' />
                    </div>
                    <div className='Bottom'>
                        <SidebarItem link='dashboard' icon='settings' title='' />
                        <SidebarItem link='dashboard' icon='logout' title='' />
                    </div>
                </div>
            </div>
        )
    }
}
