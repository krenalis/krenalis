import React, { Component } from 'react';
import ConnectorsEntry from '../../components/ConnectorEntry/ConnectorEntry';
import './AccountConnectors.css';

export default class AccountConnectors extends Component {
    
    constructor(props) {
        super(props);
        this.state = {
            'connectors': []
        }
    }

    componentDidMount = async () => {
		let connectors;
		try {
			let res = await fetch('/admin/connectors/findInstalledConnectors');
			connectors = await res.json();
		} catch (err) {
			console.error(`error while calling 'connectors/findInstalledConnectors': ${err}`);
		}
		this.setState({connectors: connectors});
	}
  
	removeConnector = async (e, id) => {
		let res
		try {
			res = await fetch('/admin/connectors/delete', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify([id]),
			});
		} catch (err) {
			console.error(`error while calling 'connectors/delete': ${err}`);
		}
		if (res.status === 200) {
			let clone = this.state.connectors.slice();
			let connectors = clone.filter((c) => {
				return c.ID !== id;
			});
			this.setState({connectors: connectors})
		}

	}

    render() {
        return (
			<div className="AccountConnectors">
				<div class='content'>
					<div className="title">Your connectors</div>
					<div className="connectors">
						{this.state.connectors.length > 0 ? this.state.connectors.map((c) => {
							return <ConnectorsEntry key={c.ID} id={c.ID} name={c.Name} logoUrl={c.LogoURL} onRemove={(e) => {this.removeConnector(e, c.ID)}} />
						}) : `You don't have any connectors installed yet`}
					</div>
				</div>
			</div>
        )
    }

}
