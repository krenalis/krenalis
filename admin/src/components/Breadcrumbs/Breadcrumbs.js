import React from 'react';
import './Breadcrumbs.css';
import { SlBreadcrumb, SlBreadcrumbItem } from '@shoelace-style/shoelace/dist/react';
import { NavLink } from 'react-router-dom';

export default class Breadcrumbs extends React.Component {
	render() {
		return (
			<SlBreadcrumb className='Breadcrumbs'>
				{this.props.breadcrumbs.map((b) => (
					<SlBreadcrumbItem>
						{b.Name}
						{b.Link && <NavLink to={b.Link}></NavLink>}
					</SlBreadcrumbItem>
				))}
			</SlBreadcrumb>
		);
	}
}
