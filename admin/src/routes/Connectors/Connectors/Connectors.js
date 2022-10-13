import React, { Component } from 'react';

import './Connectors.css';
import ConnectorItem from '../../../components/ConnectorItem/ConnectorItem';


export default class Connectors extends Component {

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
			<div className="Connectors">
				<div class='content'>
					<h1>Connectors</h1>
					<div>Click on a connector to add a data source from the connector</div>
					<div className="connectors">
						 {this.state.connectors.map((c) => {
							return <ConnectorItem key={c.ID} name={c.Name} logoUrl={c.LogoURL} onClick={(e) => {this.installConnector(e, c.ID, c.OauthURL)}} />
						 })}
					</div>
				</div>
			</div>
        )
    }
}
