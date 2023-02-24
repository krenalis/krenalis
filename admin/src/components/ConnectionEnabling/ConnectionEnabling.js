import { useContext } from 'react';
import './ConnectionEnabling.css';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { SlSwitch } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionEnabling = ({ connection: c }) => {
	let { API, showError } = useContext(AppContext);
	let { setConnection } = useContext(ConnectionContext);

	const onSwitchChange = async () => {
		let cn = { ...c };
		let v = !cn.Enabled;
		let [, err] = await API.connections.setStatus(c.ID, v);
		if (err != null) {
			showError(err);
			return;
		}
		cn.Enabled = v;
		setConnection(cn);
	};

	return (
		<div className='ConnectionEnabling'>
			<SlSwitch onSlChange={onSwitchChange} checked={c.Enabled}>
				The connection is {c.Enabled ? 'enabled' : 'disabled'}
			</SlSwitch>
		</div>
	);
};

export default ConnectionEnabling;
