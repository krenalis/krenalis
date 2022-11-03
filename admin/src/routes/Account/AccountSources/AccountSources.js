import React from 'react';
import './AccountSources.css';
import Toast from '../../../components/Toast/Toast';
import Navigation from '../../../components/Navigation/Navigation';
import Card from '../../../components/Card/Card';
import call from '../../../utils/call';
import { NavLink } from 'react-router-dom';
import { SlButton, SlIcon, SlDialog, SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react';

export default class AccountSources extends React.Component {

	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.state = {
			'askImportConfirmation': 0,
			'resetCursor': false,
			'sources': [],
			'status': null,
		};
	}

	componentDidMount = async () => {
		let [sources, err] = await call('/admin/data-sources/find');
		if (err !== null) {
			this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
			this.toast.current.toast();
			return;
		}
		this.setState({sources: sources});
	}

	handleResetCursorChange = (e) => {
		let value = e.currentTarget.value;
		if (value === 'true') this.setState({resetCursor: true});
		else if (value === 'false') this.setState({resetCursor: false});
	}

	handleImportConfirmation = async (e) => {
		let button = e.currentTarget;
		button.setAttribute('loading', '');
		let id = this.state.askImportConfirmation;
		let resetCursor = this.state.resetCursor;
		let [, err] = await call('/admin/import-raw-user-data-from-connector', {Connector: id, ResetCursor: resetCursor});
		button.removeAttribute('loading');
		if (err !== null) {
			this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}, askImportConfirmation: 0});
			this.toast.current.toast();
			return;
		}
		this.setState({status: {variant:'success', icon:'check2-circle', text:'Your import has been completed succesfully'}, askImportConfirmation: 0});
		this.toast.current.toast();
	}

	handleDelete = async (id) => {
		let [, err] = await call('/admin/data-sources/delete', [id]);
		if (err !== null) {
			this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
			this.toast.current.toast();
			return;
		}
		let clone = this.state.sources.slice();
		let sources = clone.filter((d) => {
			return d.ID !== id;
		});
		this.setState({sources: sources});
	}

	handleSettings = async (id) => {
		let [settingsUI, err] = await call('/admin/connectors/settings-ui', id);
		if (err !== null) {
			this.setState({status: {variant:'danger', icon:'exclamation-octagon', text:err}});
			this.toast.current.toast();
			return;
		}
		console.log(settingsUI);
	}

	render() {
		return (
			<div className='AccountSources'>
				<Navigation navItems={[{name: 'Your data sources', link:'/admin/account/sources', selected: true}, {name: 'Your schemas', link:'/admin/account/schemas', selected: false}]}/>
				<div class='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					{this.state.sources.length === 0 ?
						<div className='noSource'>
							<sl-icon name='plugin'></sl-icon>
							<div className='title'>No data source</div>
							<div className='description'>Get started by installing a new data source</div>
							<SlButton className='installButton' variant='primary'>
								<SlIcon slot='suffix' name='plus-circle-dotted' />
								Add a new data source
								<NavLink to='/admin/connectors'></NavLink>
							</SlButton>
						</div>
					:							
					<div className='sources'>
						{this.state.sources.map((s) => {
							return(
								<Card key={s.ID} name={s.Name} logoURL={s.LogoURL} type={s.Type}>
									<div className='buttons'>
										<SlButton className='importButton' variant='primary' onClick={() => {this.setState({askImportConfirmation: s.ID})}}>
											<SlIcon slot='suffix' name='cloud-download' />
											Import
										</SlButton>
										<SlButton className='configureButton' variant='neutral'>
											<SlIcon slot='suffix' name='shuffle' />
											Properties
											<NavLink to={`${s.ID}/properties`}></NavLink>
										</SlButton>
										<SlButton className='removeButton' variant='danger' onClick={() => {this.handleDelete(s.ID)}}>
											<SlIcon slot='suffix' name='trash3' />
											Remove
										</SlButton>
										{
											s.Type === 'Database' && 
											<SlButton className='editSQLButton' variant='neutral'>
												<SlIcon slot='suffix' name='filetype-sql' />
												Edit SQL
												<NavLink to={`${s.ID}/sql`}></NavLink>
											</SlButton>
										}
										<SlButton className='settingsButton' variant='neutral'>
											<SlIcon slot='suffix' name='gear' />
											Settings
											<NavLink to={`${s.ID}/settings`}></NavLink>
										</SlButton>
									</div>
								</Card>
							) 
						})}
						<div className='addSourceBox'>
							<sl-icon name='plugin'></sl-icon>
							<div className='text'>Add a new data source</div>
							<NavLink to='/admin/connectors'></NavLink>
						</div>
					</div>
					}
				</div>
				<SlDialog open={this.state.askImportConfirmation !== 0} style={{ '--width': '600px' }}>
					<div className='dialogTitle'>Where do you want your import to start?</div>
					<SlSelect placeholder='Select one' value={this.state.resetCursor ? 'true' : 'false'} onSlChange={this.handleResetCursorChange}>
						<SlMenuItem value='true'>Start importing all over again</SlMenuItem>
						<SlMenuItem value='false'>Pick up the import from where it left off</SlMenuItem>
					</SlSelect>
					<div className='buttons'>
						<SlButton variant='neutral' onClick={() => {this.setState({askImportConfirmation: 0})}}>
							<SlIcon slot='suffix' name='x-lg' />
							Cancel
						</SlButton>
						<SlButton variant='primary' onClick={this.handleImportConfirmation}>
							<SlIcon slot='suffix' name='cloud-download' />
							Start import
						</SlButton>
					</div>
				</SlDialog>
			</div>
		)
	}
}
