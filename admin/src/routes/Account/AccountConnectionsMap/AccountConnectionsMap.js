import React from 'react';
import './AccountConnectionsMap.css';
import Toast from '../../../components/Toast/Toast';
import Navigation from '../../../components/Navigation/Navigation';
import call from '../../../utils/call';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';
import Xarrow from 'react-xarrows';

export default class AccountConnectionsMap extends React.Component {
	constructor(props) {
		super(props);
		this.state = {
			sources: [],
			destinations: [],
		};
	}

	componentDidMount = async () => {
		let [connections, err] = await call('/admin/connections/find');
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		let sources = [];
		let destinations = [];
		for (let c of connections) {
			if (c.Role === 'Source') sources.push(c);
			if (c.Role === 'Destination') destinations.push(c);
		}
		this.setState({ sources: sources, destinations: destinations });
	};

	renderConnections = (cns) => {
		let connections = [];
		for (let c of cns) {
			let connection;
			let typ = c.Type;
			let files = [];
			if (typ === 'Storage') {
				files = this.state[`${c.Role.toLowerCase()}s`].filter((cn) => cn.Storage === c.ID);
			}
			if (typ === 'Storage' && files.length > 0) {
				connection = (
					<div className='connection storage' id={`${c.Role.toLowerCase()}-${c.ID}`}>
						<div className='files'>
							{files.map((f) => {
								return (
									<div className='file' id={`file-${f.ID}`}>
										<div className='littleLogo'>
											<img src={f.LogoURL} alt={`${f.Name}'s logo`} />
										</div>
										{f.Name}
									</div>
								);
							})}
						</div>
						<div className='storage' id={`storage-${c.ID}`}>
							<div className='littleLogo'>
								<img src={c.LogoURL} alt={`${c.Name}'s logo`} />
							</div>
							{c.Name}
						</div>
						{files.map((f) => {
							return (
								<div key={`arrow-${f.ID}`} className='arrow'>
									<Xarrow
										start={`file-${f.ID}`}
										end={`storage-${c.ID}`}
										startAnchor={c.Role === 'Source' ? 'right' : 'left'}
										endAnchor={c.Role === 'Source' ? 'left' : 'right'}
										showHead={false}
										color='#a5b4fc'
										strokeWidth={1}
									/>
								</div>
							);
						})}
					</div>
				);
			} else if (c.Type !== 'File') {
				connection = (
					<div className='connection' id={`${c.Role.toLowerCase()}-${c.ID}`}>
						<div className='littleLogo'>
							<img src={c.LogoURL} alt={`${c.Name}'s logo`} />
						</div>
						{c.Name}
					</div>
				);
			}
			connections.push(connection);
		}
		return connections;
	};

	render() {
		return (
			<div className='AccountConnectionsMap'>
				<Navigation
					navItems={[
						{ name: 'Your connections map', link: '/admin/account/connections-map', selected: true },
						{ name: 'Your connections', link: '/admin/account/connections', selected: false },
						{ name: 'Your schemas', link: '/admin/account/schemas', selected: false },
					]}
				/>
				<div className='content'>
					<div className='buttons'>
						<SlButton className='addSource' variant='primary'>
							<SlIcon slot='suffix' name='plus-circle-dotted' />
							Add a new source
							<NavLink to='/admin/connectors?role=Source'></NavLink>
						</SlButton>
						<SlButton className='addDestination' variant='primary'>
							<SlIcon slot='suffix' name='plus-circle-dotted' />
							Add a new destination
							<NavLink to='/admin/connectors?role=Destination'></NavLink>
						</SlButton>
					</div>
					<div className='map'>
						<div className='sources'>{this.renderConnections(this.state.sources)}</div>
						<div className='main'>
							<div className='centralLogo' id='centralLogo'>
								Chichi
							</div>
						</div>
						<div className='destinations'>{this.renderConnections(this.state.destinations)}</div>
					</div>
				</div>
				<div className='arrows'>
					{this.state.sources.map((s) => {
						if (s.Type !== 'File') {
							return (
								<div key={`arrow-${s.ID}`} className='arrow'>
									<Xarrow
										start={`source-${s.ID}`}
										end='centralLogo'
										startAnchor='right'
										endAnchor='left'
										showHead={false}
										color='#a5b4fc'
										strokeWidth={1}
									/>
								</div>
							);
						}
					})}
					{this.state.destinations.map((d) => {
						if (d.Type !== 'File') {
							return (
								<div key={`arrow-${d.ID}`} className='arrow'>
									<Xarrow
										start='centralLogo'
										end={`destination-${d.ID}`}
										startAnchor='right'
										endAnchor='left'
										showHead={false}
										color='#a5b4fc'
										strokeWidth={1}
									/>
								</div>
							);
						}
					})}
				</div>
			</div>
		);
	}
}
