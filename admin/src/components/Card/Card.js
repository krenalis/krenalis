import React from 'react';
import './Card.css';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

// TODO (@Andrea): implement description as a prop to replace 'lorem ipsum' hardcoded text.
export default class Card extends React.Component {
	render() {
		return (
			<div className='Card'>
				<div className='top'>
					<div className='logo'>
						{this.props.logoURL === '' ? (
							<div class='unknownLogo'>?</div>
						) : (
							<img alt={`${this.props.name}'s logo`} src={this.props.logoURL} />
						)}
					</div>
					<div className='name'>{this.props.name}</div>
					{this.props.type && (
						<SlBadge className='type' variant='neutral'>
							{this.props.type}
						</SlBadge>
					)}
					<div className='description'>Lorem ipsum dolor, sit amet consectetur adipisicing elit</div>
				</div>
				<div className='body'>{this.props.children}</div>
			</div>
		);
	}
}
