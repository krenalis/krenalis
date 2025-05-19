import React from 'react';
import './ConnectionIdentities.css';
import { useConnectionIdentities } from './useConnectionIdentities';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import Grid from '../../base/Grid/Grid';
import IconWrapper from '../../base/IconWrapper/IconWrapper';

const ConnectionIdentities = () => {
	const { isLoading, identityProperties, identitiesRows } = useConnectionIdentities();

	return (
		<div className={`connection-identities${isLoading ? ' connection-identities--loading' : ''}`}>
			{isLoading ? (
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			) : identitiesRows.length === 0 ? (
				<div className='connection-identities__no-identity'>
					<IconWrapper name='person-exclamation' size={40} />
					<div className='connection-identities__no-identity-description'>
						No identity has been imported yet
					</div>
				</div>
			) : (
				<>
					<div className='connection-identities__title'>
						These are the last {identitiesRows.length} identities imported by the connection
					</div>
					<Grid columns={identityProperties} rows={identitiesRows} />
				</>
			)}
		</div>
	);
};

export { ConnectionIdentities };
