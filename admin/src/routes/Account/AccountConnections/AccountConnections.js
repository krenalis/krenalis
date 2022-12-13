import React from 'react';
import './AccountConnections.css';
import Toast from '../../../components/Toast/Toast';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import call from '../../../utils/call';
import { NavLink } from 'react-router-dom';
import { SlButton, SlIcon, SlDialog, SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react/index.js';

export default class AccountConnections extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.state = {
			askImportConfirmation: 0,
			connectionToRemove: 0,
			resetCursor: false,
			connections: [],
			status: null,
			showImports: null,
		};
	}

	componentDidMount = async () => {
		let [connections, err] = await call('/admin/connections/find', 'GET');
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ connections: connections });
	};

	handleResetCursorChange = (e) => {
		let value = e.currentTarget.value;
		if (value === 'true') this.setState({ resetCursor: true });
		else if (value === 'false') this.setState({ resetCursor: false });
	};

	handleImportConfirmation = async (e) => {
		let button = e.currentTarget;
		button.setAttribute('loading', '');
		let id = this.state.askImportConfirmation;
		let resetCursor = this.state.resetCursor;
		let [, err] = await call('/admin/import-raw-user-data-from-connector', 'POST', {
			Connector: id,
			ResetCursor: resetCursor,
		});
		button.removeAttribute('loading');
		if (err !== null) {
			this.setState({
				status: { variant: 'danger', icon: 'exclamation-octagon', text: err },
				askImportConfirmation: 0,
			});
			this.toast.current.toast();
			return;
		}
		this.setState({
			status: { variant: 'primary', icon: 'cloud-download', text: 'Your import has been started' },
			askImportConfirmation: 0,
		});
		this.toast.current.toast();
	};

	handleDelete = async (connection) => {
		this.setState({ connectionToRemove: connection });
	};

	handleDeleteConfirmation = async () => {
		let id = this.state.connectionToRemove.ID;
		let [, err] = await call('/admin/connections/delete', 'POST', [id]);
		if (err !== null) {
			this.setState({
				status: { variant: 'danger', icon: 'exclamation-octagon', text: err },
				connectionToRemove: 0,
			});
			this.toast.current.toast();
			return;
		}
		let clone = this.state.connections.slice();
		let connections = clone.filter((d) => {
			return d.ID !== id;
		});
		this.setState({ connections: connections, connectionToRemove: 0 });
	};

	handleShowImports = async (connection) => {
		let [imports, err] = await call('/admin/connections/imports', 'POST', connection.ID);
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ showImports: { connection: connection, imports: imports } });
	};

	render() {
		return (
			<div className='AccountConnections'>
				<Navigation
					navItems={[
						{ name: 'Your connections map', link: '/admin/account/connections-map', selected: false },
						{ name: 'Your connections', link: '/admin/account/connections', selected: true },
						{ name: 'Your schemas', link: '/admin/account/schemas', selected: false },
					]}
				/>
				<div class='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					{this.state.connections.length === 0 ? (
						<div className='noConnection'>
							<sl-icon name='plugin'></sl-icon>
							<div className='title'>No connection</div>
							<div className='description'>Get started by installing a new connection</div>
							<SlButton className='installButton' variant='primary'>
								<SlIcon slot='suffix' name='plus-circle-dotted' />
								Add a new connection
								<NavLink to='/admin/connectors'></NavLink>
							</SlButton>
						</div>
					) : (
						<div className='connections'>
							{this.state.connections.map((c) => {
								return (
									<Card key={c.ID} name={c.Name} role={c.Role} logoURL={c.LogoURL} type={c.Type}>
										<div className='buttons'>
											{(c.Type === 'App' ||
												c.Type === 'Database' ||
												c.Type === 'EventStream' ||
												c.Type === 'File') &&
												c.Role === 'Source' && (
													<SlButton
														className='importButton'
														variant='primary'
														onClick={() => {
															this.setState({ askImportConfirmation: c.ID });
														}}
													>
														<SlIcon slot='suffix' name='cloud-download' />
														Import
													</SlButton>
												)}
											{(c.Type === 'App' ||
												c.Type === 'Database' ||
												c.Type === 'EventStream' ||
												c.Type === 'File') &&
												c.Role === 'Source' && (
													<SlButton
														className='showImportsButton'
														variant='neutral'
														onClick={() => {
															this.handleShowImports(c);
														}}
													>
														<SlIcon slot='suffix' name='eye' />
														See imports
													</SlButton>
												)}
											<SlButton className='settingsButton' variant='neutral'>
												<SlIcon slot='suffix' name='gear' />
												Settings
												<NavLink to={`${c.ID}/settings`}></NavLink>
											</SlButton>
											{c.Type !== 'Storage' && (
												<SlButton className='configureButton' variant='neutral'>
													<SlIcon slot='suffix' name='shuffle' />
													Properties
													<NavLink to={`${c.ID}/properties`}></NavLink>
												</SlButton>
											)}
											{c.Type === 'Database' && (
												<SlButton className='editSQLButton' variant='neutral'>
													<SlIcon slot='suffix' name='filetype-sql' />
													Edit SQL
													<NavLink to={`${c.ID}/sql`}></NavLink>
												</SlButton>
											)}
											<SlButton
												className='removeButton'
												variant='danger'
												onClick={() => {
													this.handleDelete(c);
												}}
											>
												<SlIcon slot='suffix' name='trash3' />
												Remove
											</SlButton>
										</div>
									</Card>
								);
							})}
							<div className='addConnectionBox'>
								<sl-icon name='plugin'></sl-icon>
								<div className='text'>Add a new connection</div>
								<NavLink to='/admin/connectors'></NavLink>
							</div>
						</div>
					)}
				</div>
				<SlDialog
					open={this.state.askImportConfirmation !== 0}
					style={{ '--width': '600px' }}
					onSlAfterHide={() => this.setState({ askImportConfirmation: 0 })}
				>
					<div className='dialogTitle'>Where do you want your import to start?</div>
					<SlSelect
						placeholder='Select one'
						value={this.state.resetCursor ? 'true' : 'false'}
						onSlChange={this.handleResetCursorChange}
					>
						<SlMenuItem value='true'>Start importing all over again</SlMenuItem>
						<SlMenuItem value='false'>Pick up the import from where it left off</SlMenuItem>
					</SlSelect>
					<div className='buttons'>
						<SlButton
							variant='neutral'
							onClick={() => {
								this.setState({ askImportConfirmation: 0 });
							}}
						>
							<SlIcon slot='suffix' name='x-lg' />
							Cancel
						</SlButton>
						<SlButton variant='primary' onClick={this.handleImportConfirmation}>
							<SlIcon slot='suffix' name='cloud-download' />
							Start import
						</SlButton>
					</div>
				</SlDialog>
				{this.state.showImports && (
					<SlDialog
						className='importsListDialog'
						open={true}
						onSlAfterHide={() => this.setState({ showImports: null })}
						style={{ '--width': '1000px' }}
						label={`${this.state.showImports.connection.Name}'s imports`}
					>
						{this.state.showImports.imports.length > 0 ? (
							<div className='importTable'>
								<div className='row head'>
									<div class='id'>ID</div>
									<div class='startTime'>Start time</div>
									<div class='endTime'>End time</div>
									<div class='error'>Error</div>
								</div>
								{this.state.showImports.imports.map((i) => (
									<div className={`row ${i.Error !== '' ? 'failed' : 'successfull'}`} key={i.ID}>
										<div class='id'>{i.ID}</div>
										<div class='startTime'>{i.StartTime}</div>
										<div class='endTime'>{i.EndTime}</div>
										<div class='error'>{i.Error === '' ? '-' : i.Error}</div>
									</div>
								))}
							</div>
						) : (
							<div className='noImports'>
								No import has been performed from the {this.state.showImports.connection.Name}{' '}
								connection
							</div>
						)}
					</SlDialog>
				)}
				<SlDialog
					className='removeDialog'
					open={this.state.connectionToRemove !== 0}
					style={{ '--width': '600px' }}
					onSlAfterHide={() => this.setState({ connectionToRemove: 0 })}
				>
					<div className='removeQuestion'>
						Are you sure you want to remove <span>{this.state.connectionToRemove.Name}</span>?
					</div>
					<div className='buttons'>
						<SlButton
							variant='neutral'
							onClick={() => {
								this.setState({ connectionToRemove: 0 });
							}}
						>
							<SlIcon slot='suffix' name='x-lg' />
							Cancel
						</SlButton>
						<SlButton variant='danger' onClick={this.handleDeleteConfirmation}>
							<SlIcon slot='suffix' name='trash3' />
							Remove
						</SlButton>
					</div>
				</SlDialog>
			</div>
		);
	}
}
