import './ConnectionEnabling.css';
import call from '../../utils/call';
import { SlSwitch } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionEnabling = ({ connection: c, onConnectionChange, onError }) => {
	const onSwitchChange = async () => {
		let cn = { ...c };
		let v = !cn.Enabled;
		let [, err] = await call(`/api/connections/${c.ID}/status`, 'POST', { Enabled: v });
		if (err != null) {
			onError(err);
			return;
		}
		cn.Enabled = v;
		onConnectionChange(cn);
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
