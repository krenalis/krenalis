import React, { Component } from 'react'
import { CartesianGrid, AreaChart, PieChart, Pie, Area, Legend, LineChart, Line, BarChart, YAxis, XAxis, Tooltip, Bar } from 'recharts'
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
                    <Bar dataKey='p' fill='var(--color-accent)' />
                </BarChart>;
                break;
            case 'line':
                chart = <LineChart width={this.props.width} height={this.props.height} data={this.props.data}
                    margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip />
                    <Line type="monotone" dataKey="p" stroke="var(--color-accent)" />
                </LineChart>;
                break;
            case 'area':
                chart = <AreaChart width={this.props.width} height={this.props.height} data={this.props.data}
                    margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
                    <defs>
                        <linearGradient id="colorUv" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#8884d8" stopOpacity={0.8} />
                            <stop offset="95%" stopColor="#8884d8" stopOpacity={0} />
                        </linearGradient>
                        <linearGradient id="colorPv" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#82ca9d" stopOpacity={0.8} />
                            <stop offset="95%" stopColor="#82ca9d" stopOpacity={0} />
                        </linearGradient>
                    </defs>
                    <XAxis dataKey="name" />
                    <YAxis />
                    <CartesianGrid strokeDasharray="3 3" />
                    <Tooltip />
                    <Area type="monotone" dataKey="p" stroke="#82ca9d" fillOpacity={1} fill="url(#colorPv)" />
                </AreaChart>
                break;
            case 'pie':
                chart = <PieChart width={this.props.width} height={this.props.height}>
                    <Pie data={this.props.data} dataKey="p" nameKey="name" cx="50%" cy="50%" outerRadius={100} fill="#8884d8" />
                    <Legend />
                </PieChart>
                break;
            default:
                chart = 'Error: Unknown chart type';
        }
        return (
            <div className="Chart">{chart}</div>
        )
    }
}
