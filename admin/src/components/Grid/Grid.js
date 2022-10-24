import React from 'react';
import './Grid.css';

export default class Grid extends React.Component {
    render() {
        return (
            <div className="Grid" style={{gridTemplateColumns: `repeat(${this.props.table.Columns.length}, 1fr)`}}>
                {this.props.table.Columns.map((column) => {
                    return <div className="headCell">{column.Name}</div>
                })}
                {this.props.table.Rows.map((row) => {
                    return row.map((value) => <div className="cell">{value}</div>)
                })}
            </div>
    )
  }
}
