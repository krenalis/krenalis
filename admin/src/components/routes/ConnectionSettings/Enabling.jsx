import { useContext } from 'react';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { SlSwitch } from '@shoelace-style/shoelace/dist/react/index.js';

const Enabling = ({ connection: c }) => {
	const { api, showError } = useContext(AppContext);
	const { isConnectionStale } = useContext(ConnectionContext);

	const onSwitchChange = async () => {
		const cn = { ...c };
		const v = !cn.enabled;
		try {
			await api.connections.setStatus(c.id, v);
		} catch (err) {
			showError(err);
			return;
		}
		cn.enabled = v;
		isConnectionStale(true);
	};

	return (
		<SlSwitch onSlChange={onSwitchChange} checked={c.enabled}>
			The connection is {c.enabled ? 'enabled' : 'disabled'}
		</SlSwitch>
	);
};

export default Enabling;
