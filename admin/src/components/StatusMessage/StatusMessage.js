import React, { Component } from 'react'

import './StatusMessage.css'

export default class StatusMessage extends Component {
    render() {
        let className, icon;
        switch (this.props.message.type) {
            case 'error':
                className = 'error';
                icon = <i className="material-symbols-outlined">error</i>;    
                break;
            case 'success':
                className = 'success';
                icon = <i className="material-symbols-outlined">check_circle</i>;    
                break;
            case 'warning':
                className = 'warning'
                icon = <i className="material-symbols-outlined">warning</i>;    
                break;
            default:
                className = 'info'
                icon = <i className="material-symbols-outlined">info</i>;    
        }

        return (
            <div className={`StatusMessage ${className}`}>
                <div className="text">
                    {icon}
                    {this.props.message.text}
                </div>
                <div className="action">
                    <i className="material-symbols-outlined close" onClick={this.props.onClose}>close</i>
                </div>
            </div>
        )
    }
}
