import React, { Component } from 'react'
import './Dialog.css'

export default class Dialog extends Component {
    render() {
        let icon;
        switch (this.props.type) {
            case "question":
                icon = <i class="material-symbols-outlined">question_mark</i>
                break;
            default:
                icon = <i class="material-symbols-outlined">waving_hand</i>
                break;
        }
        return (
            <div className={`Dialog ${this.props.type}`} style={{width: this.props.width, height: this.props.height}} data-isOpen={this.props.isOpen}>
                <div className="close" onClick={this.props.onClose}><i class="material-symbols-outlined">close</i></div>
                <div className="icon">{icon}</div>
                <div className="text">
                    <div className="title">{this.props.title}</div>
                    <div className="description">{this.props.description}</div>
                </div>
                <div className="children">
                    {this.props.children}
                </div>
            </div>
        )
    }
}
