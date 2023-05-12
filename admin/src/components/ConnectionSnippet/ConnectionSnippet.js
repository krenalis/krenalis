import { useContext } from 'react';
import './ConnectionSnippet.css';
import ClipboardInput from '../ClipboardInput/ClipboardInput';
import { ConnectionContext } from '../../context/ConnectionContext';

const ConnectionSnippet = () => {
	let { connection: c } = useContext(ConnectionContext);

	return (
		<div className='connectionSnippet'>
			<div>You can use the ID of the connection to setup your source:</div>
			<ClipboardInput value={c.ID}></ClipboardInput>
		</div>
	);
};

export default ConnectionSnippet;
