import React, { Component } from 'react'
import './StatusMessage.css'

export default class StatusMessage extends Component {
    render() {
        return (
            <div className="StatusMessage">
                <div className="text">
                    <i className="material-symbols-outlined error">error</i>
                    {this.props.text}
                </div>
                <div className="action">
                    <i className="material-symbols-outlined close" onClick={this.props.onClose}>close</i>
                </div>
            </div>
        )
    }
}
