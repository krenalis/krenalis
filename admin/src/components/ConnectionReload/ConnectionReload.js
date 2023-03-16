import { useContext, useState } from 'React';
import './ConnectionReload.css';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import statuses from '../../constants/statuses';

const ConnectionReload = () => {
	let [isLoading, setIsLoading] = useState(false);

	let { API, showStatus, showError } = useContext(AppContext);
	let { connection: c } = useContext(ConnectionContext);

	const reloadConnection = async () => {
		setIsLoading(true);
		let [, err] = await API.connections.reload(c.ID);
		if (err !== null) {
			setIsLoading(false);
			showError(err);
			return;
		}
		setTimeout(() => {
			setIsLoading(false);
			showStatus(statuses.connectionReloaded);
		}, 300);
	};

	return (
		<div className='connectionReload'>
			<p>
				Click to reload the schema and the action types of <b>{c.Name}</b>
			</p>
			<SlButton variant='primary' onClick={reloadConnection} loading={isLoading}>
				<SlIcon slot='prefix' name='arrow-clockwise'></SlIcon>
				Reload
			</SlButton>
		</div>
	);
};

export default ConnectionReload;
