import React from 'react';
import './Sidebar.css';
import { NavLink, Navigate } from 'react-router-dom';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react';

export default class Sidebar extends React.Component {
    
    constructor(props) {
        super(props);
        this.state = {
            'isLoggedOut': false,
        };
    }

    handleLogout = () => {
        document.cookie = 'session=; Max-Age=-99999999; Path=/';  
        this.setState({isLoggedOut: true});
    }
    
    render() {
        if (this.state.isLoggedOut) {
            return <Navigate to='/admin' />
        } else {
            return (
                <div className='Sidebar'>
                    <div className='Items'>
                        <div className='Top'>
                            <SlButton variant='text'>
                                <SlIcon name='plugin' />
                                <NavLink to='/admin/connectors'></NavLink>
                            </SlButton>
                            <SlButton variant='text'>
                                <SlIcon name='person-circle' />
                                <NavLink to='/admin/account/connections'></NavLink>
                            </SlButton>
                        </div>
                        <div className='Bottom'>
                            <SlButton variant='text' onClick={this.handleLogout}>
                                <SlIcon name='box-arrow-left' />
                            </SlButton>
                        </div>
                    </div>
                </div>
            )
        }
    }
}
