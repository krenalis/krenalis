import React, { Component } from 'react'
import './Accordion.css'

export default class Accordion extends Component {
    constructor(props) {
        super(props);
        this.state = {
            'isOpen': false,
        }
    }

    render() {
        return (
            <div className="Accordion">
                <div className="title" onClick={() => { this.setState({ 'isOpen': !this.state.isOpen }) }}>
                    {this.props.title}
                    <i className="material-symbols-outlined">{this.state.isOpen ? 'expand_less' : 'expand_more'}</i>
                </div>
                {this.state.isOpen ? <div className="accordion-content">{this.props.children}</div> : ''}
            </div>

        )
    }
}
