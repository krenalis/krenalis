import React, { Component } from 'react';
import ConnectorEntry from '../../components/ConnectorEntry/ConnectorEntry';
import './ConnectorsList.css';

export default class ConnectorsList extends Component {

	constructor(props) {
		super(props);
		this.state = {
			connectors: [],
		}
	}

	componentDidMount = async () => {
		let connectors;
		try {
			let res = await fetch('/admin/connectors/find');
			connectors = await res.json();
		} catch (err) {
			console.error(`error while calling 'connectors/find': ${err.message}`);
		}
		this.setState({ connectors: connectors });
	}

	installConnector = async (e, id, url) => {
		document.cookie = `install-connector=${id};path=/`;
		window.location = url;
	}

    render() {
        return (
			<div className="ConnectorsList">
				<div class='content'>
					<div className="title">Select a connector to add</div>
					<div className="connectors">
						 {this.state.connectors.map((c) => {
							return <ConnectorEntry key={c.ID} id={c.ID} name={c.Name} logoUrl={c.LogoURL} onClick={(e) => {this.installConnector(e, c.ID, c.OauthURL)}} />
						 })}
					</div>
				</div>
			</div>
        )
    }
}
