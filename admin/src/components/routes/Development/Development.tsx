import React, { useContext, useLayoutEffect } from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import AppContext from '../../../context/AppContext';
import ListTile from '../../base/ListTile/ListTile';
import { Link } from '../../base/Link/Link';
import NotFound from '../NotFound/NotFound';
import './Development.css';

const Development = () => {
	const { selectedWorkspace, setTitle, workspaces } = useContext(AppContext);
	const workspace = workspaces.find((item) => item.id === selectedWorkspace);

	useLayoutEffect(() => {
		setTitle('Development');
	}, [setTitle]);

	if (workspace?.environment !== 'development') return <NotFound />;

	return (
		<div className='development'>
			<div className='route-content'>
				<div className='development__content'>
					<Link path='simulated-world'>
						<ListTile
							className='development__item'
							icon={<WaIcon name='database' />}
							name='Simulated world'
							description='Create simulated applications for development, testing and demos.'
							showHover={true}
							action={<WaIcon name='chevron-right' />}
						/>
					</Link>
				</div>
			</div>
		</div>
	);
};

export { Development };
