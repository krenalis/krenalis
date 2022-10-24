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
			goToSourceAdded: 0,
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

	addSource = async (id, type, oauthURL, e) => {
		e.currentTarget.setAttribute('loading', '');
		if (oauthURL === '') {
			let [, err] = await call('/admin/add-data-source', {Type: type, Connector: id, Stream: 0});
			if (err != null) {
				this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
				e.currentTarget.removeAttribute('loading');
				this.toast.current.toast();
				return;
			}
			this.setState({goToSourceAdded: id});
			return;
		}
		// install with Oauth.
		document.cookie = `add-source=${id};path=/`;
		window.location = oauthURL;
		return;
	}

	render() {
		if (this.state.goToSourceAdded !== 0) {
			return <Navigate to={`added/${this.state.goToSourceAdded}`} />
		} else {
			return (
				<div className='Connectors'>
				<Navigation navItems={[{name: 'Add a data source', link:'/admin/connectors', selected: true}]}/>
					<div class='content'>
						<Toast reactRef={this.toast} status={this.state.status} />
						<div className='connectors'>
							{this.state.connectors.map((c) => {
								return(
									<Card key={c.ID} name={c.Name} logoURL={c.LogoURL} type={c.Type}>
										<SlTooltip content={`Add ${c.Name}`}>
											<SlButton size='medium' variant='primary' onClick={async (e) => {await this.addSource(c.ID, c.Type, c.OauthURL, e)}} circle>
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
