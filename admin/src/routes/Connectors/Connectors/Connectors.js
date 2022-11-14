import React from 'react';
import './Connectors.css';
import call from '../../../utils/call';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import Toast from '../../../components/Toast/Toast';
import { Navigate } from 'react-router-dom';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react';

export default class Connectors extends React.Component {

	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.state = {
			connectors: [],
			goToConnectionAdded: 0,
			status: null,
		};
	}

	componentDidMount = async () => {
		let [connectors, err] = await call('/admin/connectors/find');
		if (err != null) {
			this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
			this.toast.current.toast();
			return;
		}
		this.setState({ connectors: connectors });
	}

	addConnection = async (id, type, oAuthURL, e) => {
		e.currentTarget.setAttribute('loading', '');
		if (oAuthURL === '') {
			let [, err] = await call('/admin/add-connection', {Type: type, Connector: id, Storage: 0});
			if (err != null) {
				this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
				e.currentTarget.removeAttribute('loading');
				this.toast.current.toast();
				return;
			}
			this.setState({goToConnectionAdded: id});
			return;
		}
		// install with Oauth.
		document.cookie = `add-connection=${id};path=/`;
		window.location = oAuthURL;
		return;
	}

	render() {
		if (this.state.goToConnectionAdded !== 0) {
			return <Navigate to={`added/${this.state.goToConnectionAdded}`} />
		} else {
			return (
				<div className='Connectors'>
				<Navigation navItems={[{name: 'Add a connection', link:'/admin/connectors', selected: true}]}/>
					<div class='content'>
						<Toast reactRef={this.toast} status={this.state.status} />
						<div className='connectors'>
							{this.state.connectors.map((c) => {
								return(
									<Card key={c.ID} name={c.Name} logoURL={c.LogoURL} type={c.Type}>
										<SlTooltip content={`Add ${c.Name}`}>
											<SlButton size='medium' variant='primary' onClick={async (e) => {await this.addConnection(c.ID, c.Type, c.OAuthURL, e)}} circle>
												<SlIcon name='plus' />
											</SlButton>
										</SlTooltip>
									</Card>
								)
							})}
						</div>
					</div>
				</div>
			)
		}
	}
}
