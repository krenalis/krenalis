import React, { useContext } from 'react';
import ConnectionContext from '../../../context/ConnectionContext';
import { Snippet } from '../../base/Snippet/Snippet';

const ConnectionSnippet = () => {
	const { connection } = useContext(ConnectionContext);

	return (
		<div className='connection-settings__snippet-copy'>
			<div>Embed the snippet in your website to start sending events:</div>
			<Snippet connectionID={connection.id} />
		</div>
	);
};

export default ConnectionSnippet;
