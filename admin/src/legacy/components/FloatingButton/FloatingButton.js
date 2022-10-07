import React, { Component } from 'react'
import './FloatingButton.css'

export default class FloatingButton extends Component {
    render() {
        return (
            <div className="FloatingButton" onClick={this.props.onClick}>
                <i className="material-symbols-outlined">{this.props.icon}</i>
            </div>
        )
    }
}
