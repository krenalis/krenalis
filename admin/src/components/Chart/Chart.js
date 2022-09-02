import React, { Component } from 'react'
import { CartesianGrid, LineChart, Legend, Line, BarChart, YAxis, XAxis, Tooltip, Bar } from 'recharts'
import './Chart.css'

export default class Chart extends Component {
    render() {
        let chart;
        switch (this.props.type) {
            case 'bar':
                chart = <BarChart width={this.props.width} height={this.props.height} data={this.props.data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey='name' />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey='p' fill='var(--color-primary)' />
                </BarChart>;
                break;
            case 'line':
                chart = <LineChart width={this.props.width} height={this.props.height} data={this.props.data}
                    margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Line type="monotone" dataKey="p" stroke="var(--color-primary)" />
                </LineChart>;
                break;
            default:
                chart = 'Error: Unknown chart type';
        }
        return (
            <div className="Chart">{chart}</div>
        )
    }
}
