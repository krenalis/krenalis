import React from 'react';
import './ConnectionSQL.css';
import NotFound from '../NotFound/NotFound';
import Toast from '../../components/Toast/Toast';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import Grid from '../../components/Grid/Grid';
import call from '../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const queryMaxSize = 16777215;

export default class ConnectionSQL extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.connectionID = Number(String(window.location).split('/').at(-2));
		this.state = {
			connection: {},
			status: null,
			notFound: false,
			query: '',
			limit: 20, // TODO(@Andrea): implement as a select
			table: null,
		};
	}

	componentDidMount = async () => {
		let [connection, err] = await call('/admin/connections/get', 'POST', this.connectionID);
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		if (connection == null) {
			this.setState({ notFound: true });
			return;
		}
		this.setState({ connection: connection, query: connection.UsersQuery });
	};

	handlePreview = async () => {
		if (this.state.query.length > queryMaxSize) {
			this.setState({
				status: { variant: 'danger', icon: 'exclamation-octagon', text: 'You query is too long' },
			});
			this.toast.current.toast();
			return;
		}
		if (!this.state.query.includes(':limit')) {
			this.setState({
				status: {
					variant: 'danger',
					icon: 'exclamation-octagon',
					text: `your query does not contain the ':limit' placeholder`,
				},
			});
			this.toast.current.toast();
			return;
		}
		let [table, err] = await call('/admin/connections/preview-query', 'POST', {
			Connection: this.state.connection.ID,
			Query: this.state.query,
			Limit: this.state.limit,
		});
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		if (table.Columns.length === 0) {
			this.setState({
				status: {
					variant: 'danger',
					icon: 'exclamation-octagon',
					text: 'Your query did not return any columns',
				},
			});
			this.toast.current.toast();
			return;
		}
		if (table.Rows.length === 0) {
			this.setState({
				status: { variant: 'danger', icon: 'exclamation-octagon', text: 'Your query did not return any rows' },
			});
			this.toast.current.toast();
			return;
		}
		this.setState({ table: table });
	};

	saveQuery = async () => {
		if (this.state.query.length > queryMaxSize) {
			this.setState({
				status: { variant: 'danger', icon: 'exclamation-octagon', text: 'You query is too long' },
			});
			this.toast.current.toast();
			return;
		}
		if (!this.state.query.includes(':limit')) {
			this.setState({
				status: {
					variant: 'danger',
					icon: 'exclamation-octagon',
					text: `your query does not contain the ':limit' placeholder`,
				},
			});
			this.toast.current.toast();
			return;
		}
		let [, err] = await call('/admin/connections/set-users-query', 'POST', {
			Connection: this.state.connection.ID,
			Query: this.state.query,
		});
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({
			status: { variant: 'success', icon: 'check2-circle', text: 'Your query has been successfully saved' },
		});
		this.toast.current.toast();
	};

	render() {
		if (this.state.notFound) {
			return <NotFound />;
		} else {
			return (
				<div className='ConnectionSQL'>
					<Breadcrumbs
						breadcrumbs={[
							{ Name: 'Connections list', Link: '/admin/connections' },
							{ Name: `${this.state.connection.Name} query configuration` },
						]}
					/>
					<div className='routeContent'>
						<Toast reactRef={this.toast} status={this.state.status} />
						<div className='title'>
							{this.state.connection.LogoURL !== '' && (
								<img
									className='littleLogo'
									src={this.state.connection.LogoURL}
									alt={`${this.state.connection.Name}'s logo`}
								/>
							)}
							<div className='text'>Configure your {this.state.connection.Name} query</div>
						</div>
						<div className='editorWrapper'>
							<Editor
								onChange={(value) => {
									this.setState({ query: value });
								}}
								defaultLanguage='sql'
								value={this.state.query}
								theme='vs-primary'
							/>
						</div>
						<div className='buttons'>
							<SlButton
								className='previewButton'
								variant='neutral'
								size='large'
								onClick={this.handlePreview}
							>
								<SlIcon slot='prefix' name='eye' />
								Preview
							</SlButton>
							<SlButton className='saveButton' variant='primary' size='large' onClick={this.saveQuery}>
								<SlIcon slot='prefix' name='save' />
								Save
							</SlButton>
						</div>
					</div>
					{this.state.table && (
						<SlDialog
							label='Users preview'
							open={true}
							style={{ '--width': '1200px' }}
							onSlAfterHide={() => this.setState({ table: null })}
						>
							<Grid table={this.state.table} />
						</SlDialog>
					)}
				</div>
			);
		}
	}
}
