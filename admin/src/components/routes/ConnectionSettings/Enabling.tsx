import React, { useContext } from 'react';
import { AppContext } from '../../../context/providers/AppProvider';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import TransformedConnection from '../../../lib/helpers/transformedConnection';

interface EnablingProps {
	connection: TransformedConnection;
}

const Enabling = ({ connection: c }: EnablingProps) => {
	const { api, showError, setAreConnectionsStale } = useContext(AppContext);

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
		setAreConnectionsStale(true);
	};

	return (
		<SlSwitch onSlChange={onSwitchChange} checked={c.enabled}>
			The connection is {c.enabled ? 'enabled' : 'disabled'}
		</SlSwitch>
	);
};

export default Enabling;
