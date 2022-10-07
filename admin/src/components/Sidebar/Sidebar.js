import React, { Component } from 'react';
import { Link } from 'react-router-dom';

import './Sidebar.css';
import SidebarItem from '../SidebarItem/SidebarItem';

export default class Sidebar extends Component {
    render() {
        return (
            <div className='Sidebar'>
                <div className='logo'>
                    <div className='image'>C</div>
                    <Link to='connectors'></Link>
                </div>
                <div className="Items">
                    <div className='Top'>
                        <SidebarItem link='connectors' icon='list_alt' title='' />
                        <SidebarItem link='account/connectors' icon='account_circle' title='' />
                        <SidebarItem link='configurations/schema' icon='tune' title='' />
                    </div>
                    <div className='Bottom'>
                        <SidebarItem link='/admin/' icon='logout' title='' /> {/* TODO(@Andrea): return redirect to logout page where the cookie is removed */}
                    </div>
                </div>
            </div>
        )
    }
}
