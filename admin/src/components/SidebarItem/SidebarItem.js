import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import './SidebarItem.css';

// TODO(@Andrea): this item should resize based on the sidebar 'isOpen' state.

export default class SidebarItem extends Component {
    render() {
        return (
            <div className='SidebarItem'>
                <i className='material-symbols-outlined'>{this.props.icon}</i>
                {this.props.href ? <a href={this.props.href}></a> : <Link to={this.props.link}></Link>}
            </div>
        )
    }
}
