import React from 'react';
import './Navigation.css';
import { NavLink } from 'react-router-dom';
import { SlSelect, SlMenuItem, SlAvatar } from '@shoelace-style/shoelace/dist/react';

export default class Navigation extends React.Component {
	render() {
		return (
			<div className='Navigation'>
				<div className='top'>
					<SlSelect name='workspaceSelector' value='1'>
						<SlMenuItem value='1' selected>
							Mock workspace 1
						</SlMenuItem>
						<SlMenuItem value='2'>Mock workspace 2</SlMenuItem>
					</SlSelect>
					<div className='right'>
						<sl-icon name='bell'></sl-icon>
						<SlAvatar image='data:image/jpeg;base64,/9j/' />
					</div>
				</div>
				<div className='bottom'>
					{this.props.navItems &&
						this.props.navItems.map((i, index) => {
							return (
								<div className={`navItem${i.selected ? ' selected' : ''}`}>
									<div className='name'>{i.name}</div>
									<NavLink to={i.link}></NavLink>
								</div>
							);
						})}
				</div>
			</div>
		);
	}
}
