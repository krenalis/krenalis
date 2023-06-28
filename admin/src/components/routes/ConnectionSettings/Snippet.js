import { useContext } from 'react';
import ClipboardInput from '../../common/ClipboardInput/ClipboardInput';
import { ConnectionContext } from '../../../providers/ConnectionProvider';

const Snippet = () => {
	const { connection: c } = useContext(ConnectionContext);

	return (
		<>
			<div>You can use the ID of the connection to setup your source:</div>
			<ClipboardInput value={c.id}></ClipboardInput>
		</>
	);
};

export default Snippet;
