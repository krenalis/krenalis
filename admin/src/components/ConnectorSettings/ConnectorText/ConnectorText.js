import React from 'react';
import './ConnectorText.css';

export default class ConnectorText extends React.Component {
    render() {
        return (
            <div className="ConnectorText">
                <div className="label">{this.props.label}</div>
                <div className="value">{this.props.value}</div>
            </div>
        )
    }
}
