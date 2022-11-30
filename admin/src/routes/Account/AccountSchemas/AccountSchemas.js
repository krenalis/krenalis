import React from 'react';
import './AccountSchemas.css';
import Toast from '../../../components/Toast/Toast';
import Navigation from '../../../components/Navigation/Navigation';
import call from '../../../utils/call';
import { SlButton, SlIcon, SlSelect, SlMenuItem } from '@shoelace-style/shoelace/dist/react/index.js';
import { DiffEditor } from '@monaco-editor/react';

export default class AccountSchemas extends React.Component {
	constructor(props) {
		super(props);
		this.toast = React.createRef();
		this.state = {
			newSchema: '',
			originalSchema: '',
			schemaName: 'user',
			status: null,
		};
	}

	componentDidMount = async () => {
		let [schema, err] = await call('/admin/schemas/get', { schemaName: this.state.schemaName });
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ originalSchema: schema, newSchema: schema });
	};

	handleSchemaNameUpdate = async (e) => {
		let schemaName = e.currentTarget.value;
		let [schema, err] = await call('/admin/schemas/get', { schemaName: schemaName });
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({ schemaName: schemaName, originalSchema: schema, newSchema: schema });
	};

	handleEditorMount = (editor) => {
		const modifiedEditor = editor.getModifiedEditor();
		modifiedEditor.onDidChangeModelContent((_) => {
			this.setState({ newSchema: modifiedEditor.getValue() });
		});
	};

	handleSchemaSaving = async () => {
		let [, err] = await call('/admin/schemas/update', {
			SchemaName: this.state.schemaName,
			Schema: this.state.newSchema,
		});
		if (err !== null) {
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			this.toast.current.toast();
			return;
		}
		this.setState({
			status: { variant: 'success', icon: 'check2-circle', text: 'Your schema has been saved successfully' },
			originalSchema: this.state.newSchema,
		});
		this.toast.current.toast();
	};

	render() {
		return (
			<div className='AccountSchemas'>
				<Navigation
					navItems={[
						{ name: 'Your connections map', link: '/admin/account/connections-map', selected: false },
						{ name: 'Your connections', link: '/admin/account/connections', selected: false },
						{ name: 'Your schemas', link: '/admin/account/schemas', selected: true },
					]}
				/>
				<div className='content'>
					<Toast reactRef={this.toast} status={this.state.status} />
					<div className='bar'>
						<SlSelect
							name='schemaSelector'
							value={this.state.schemaName}
							onSlChange={(e) => {
								this.handleSchemaNameUpdate(e);
							}}
						>
							<SlMenuItem value='user'>User</SlMenuItem>
							<SlMenuItem value='group'>Group</SlMenuItem>
							<SlMenuItem value='event'>Event</SlMenuItem>
						</SlSelect>
						<SlButton
							className='saveButton'
							variant='primary'
							size='large'
							onClick={this.handleSchemaSaving}
						>
							<SlIcon slot='prefix' name='save' />
							Save
						</SlButton>
					</div>
					<div className='editorWrapper'>
						<DiffEditor
							theme='vs-light'
							language='json'
							original={this.state.originalSchema}
							modified={this.state.newSchema}
							value={this.state.newSchema}
							onMount={this.handleEditorMount}
						/>
					</div>
				</div>
			</div>
		);
	}
}
