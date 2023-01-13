import React from 'react';
import './Breadcrumbs.css';
import { SlBreadcrumb, SlBreadcrumbItem } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';

const Breadcrumbs = ({ breadcrumbs, onAccent }) => {
	return (
		<SlBreadcrumb className={`Breadcrumbs${onAccent ? ' onAccent' : ''}`}>
			{breadcrumbs.map((b) => (
				<SlBreadcrumbItem>
					{b.Name}
					{b.Link && <NavLink to={b.Link}></NavLink>}
				</SlBreadcrumbItem>
			))}
		</SlBreadcrumb>
	);
};

export default Breadcrumbs;
