import React, { Component } from 'react'
import './Overlay.css'

export default class Overlay extends Component {
  render() {
    return (
      <div className="Overlay" data-isOpen={this.props.isOpen}></div>
    )
  }
}
