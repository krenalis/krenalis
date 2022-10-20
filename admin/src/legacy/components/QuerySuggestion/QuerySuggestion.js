import React from 'react'
import './QuerySuggestion.css'

export default class QueryExample extends React.Component {
    render() {
        return (
            <div className="example">
                <div className="description">{this.props.description}</div>
                <div className="btn secondary" onClick={() => { this.props.onClick(this.props.query) }}>Apply</div>
            </div>
        )
    }
}
