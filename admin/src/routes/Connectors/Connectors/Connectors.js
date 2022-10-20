import React from 'react';
import './Connectors.css';
import call from '../../../utils/call';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import Toast from '../../../components/Toast/Toast';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react';

export default class Connectors extends React.Component {

	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.state = {
			connectors: [],
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

	addSource = async (id, url, e) => {
		e.currentTarget.setAttribute('loading', '');
		document.cookie = `add-source=${id};path=/`;
		window.location = url;
	}

	render() {
		return (
			<div className='Connectors'>
			<Navigation navItems={[{name: 'Add a data source', link:'/admin/connectors', selected: true}]}/>
				<div class='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					<div className='connectors'>
						{this.state.connectors.map((c) => {
							return(
								<Card key={c.ID} name={c.Name} logoURL={c.LogoURL}>
									<SlTooltip content={`Add ${c.Name}`}>
										<SlButton size='medium' variant='primary' onClick={(e) => {this.addSource(c.ID, c.OauthURL, e)}} circle>
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
