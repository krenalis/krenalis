import React from 'react';
import './AccountConnectionSettings.css';
import NotFound from '../../NotFound/NotFound';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import Toast from '../../../components/Toast/Toast';
import call from '../../../utils/call';
import { renderConnectorComponent } from '../../../components/ConnectorSettings/renderConnectorComponent';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

export default class AccountConnectionSettings extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.connectionID = Number(String(window.location).split('/').at(-2));
		this.state = {
			connection: {},
			settings: { Components: null, Actions: null },
			form: {},
			notFound: false,
			status: null,
		};
	}

	componentDidMount = async () => {
		let err, connection, ui;
		[connection, err] = await call('/admin/connections/get', this.connectionID);
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		if (connection == null) {
			this.setState({ notFound: true });
			return;
		}
		this.setState({ connection: connection });
		[ui, err] = await call('/admin/connectors/ui', this.connectionID);
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		if (ui.Alert != null) alert(`Unexpected alert sent with 'load' event: ${JSON.stringify(ui.Alert)}`);
		let form = {};
		if (ui.Form.Fields != null) {
			for (let f of ui.Form.Fields) {
				form[f.Name] = f.Value;
			}
		}
		this.setState({ settings: ui.Form, form: form });
	};

	onComponentChange = (name, value) => {
		let form = { ...this.state.form };
		form[name] = value;
		this.setState({ form: form });
	};

	onActionClick = async (event) => {
		let [ui, err] = await call('/admin/connectors/ui-event', {
			connection: this.connectionID,
			event: event,
			form: this.state.form,
		});
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		if (ui.Alert != null) {
			this.setState({
				status: { variant: ui.Alert.Variant, icon: 'exclamation-square', text: ui.Alert.Message },
			});
			this.toast.current.toast();
		}
		if (ui.Form != null) {
			let form = {};
			if (ui.Form.Fields != null) {
				for (let f of ui.Form.Fields) {
					form[f.Name] = f.Value;
				}
			}
			this.setState({ settings: ui.Form, form: form });
		}
	};

	render() {
		if (this.state.notFound) {
			return <NotFound />;
		} else {
			return (
				<div className='AccountConnectionSettings'>
					<Breadcrumbs
						breadcrumbs={[
							{ Name: 'Your connections', Link: '/admin/account/connections' },
							{ Name: `${this.state.connection.Name}'s settings` },
						]}
					/>
					<div className='content'>
						<Toast reactRef={this.toast} status={this.state.status} />
						<div className='title'>
							{this.state.connection.LogoURL !== '' && (
								<img
									className='littleLogo'
									src={this.state.connection.LogoURL}
									alt={`${this.state.connection.Name}'s logo`}
								/>
							)}
							<div className='text'>Configure {this.state.connection.Name}</div>
						</div>
						<div className='settings'>
							<div className='components'>
								{this.state.settings.Fields != null &&
									this.state.settings.Fields.map((c, i) =>
										renderConnectorComponent(c, this.onComponentChange)
									)}
							</div>
							<div className='actions'>
								{this.state.settings.Actions != null &&
									this.state.settings.Actions.map((a, i) => (
										<SlButton
											variant={a.Variant}
											onClick={async () => {
												await this.onActionClick(a.Event);
											}}
										>
											{a.Text}
										</SlButton>
									))}
							</div>
						</div>
					</div>
				</div>
			);
		}
	}
}
