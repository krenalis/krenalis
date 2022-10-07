import React, { Component } from 'react'

import './Button.css'

export default class Button extends Component {
  render() {
    return (
      <div className={`Button ${this.props.theme ? this.props.theme : ''}`} onClick={this.props.onClick}>
        <div className="text">{this.props.text}</div>
        <i className="material-symbols-outlined">{this.props.icon}</i>
      </div>
    )
  }
}
