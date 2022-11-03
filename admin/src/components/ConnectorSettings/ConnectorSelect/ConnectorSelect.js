import React from 'react';
import './ConnectorSelect.css';
import { SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react';

export default class ConnectorSelect extends React.Component {
	
	state = {value: this.props.value};

	onSelectChange = (e) => {
		this.setState({value: e.currentTarget.value});
        this.props.onChange(this.props.name, e.currentTarget.value, e);
    }
	
	render() {
		return (
			<div className='ConnectorSelect'>
				<SlSelect label={this.props.label} value={this.state.value} placeholder={this.props.placeholder} name={this.props.name} onSlChange={this.onSelectChange}>
					{this.props.options.map((opt, i) => {
						return <SlMenuItem value={opt.Value}>{opt.Text}</SlMenuItem>
					})}
				</SlSelect>
			</div>
		)
	}
}
