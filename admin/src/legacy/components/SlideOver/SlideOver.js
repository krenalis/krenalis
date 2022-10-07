import React, { Component } from 'react'
import './SlideOver.css'

export default class SlideOver extends Component {
    render() {
        return (
            this.props.isOpen ?
                <div className='SlideOver'>
                    <div className="slideover-content">
                        <div className="head">
                            <div className="title">{this.props.title}</div>
                            <i onClick={this.props.onClose} className="material-symbols-outlined close">close</i>
                        </div>
                        <div className="children">
                            {this.props.children}
                        </div>
                    </div>
                    <div className="slideover-overlay"></div>
                </div> :
                ''
        )
    }
}
