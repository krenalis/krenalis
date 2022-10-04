import React, { Component } from 'react'
import './SchemaSettings.css'
import { DiffEditor } from '@monaco-editor/react';
import StatusMessage from '../../components/StatusMessage/StatusMessage';

export default class SchemaSettings extends Component {

	constructor(props) {
		super(props);
		this.state = {
			schemaName: 'user',
			originalSchema: '',
			newSchema: '',
			statusMessage: null,
		}
	}

	componentDidMount = async () => {
		let schema;
		try {
			let res = await fetch('/admin/schemas/get', {
				method: 'POST',
				body: JSON.stringify({SchemaName: this.state.schemaName})
			});
			schema = await res.json();
		} catch(err) {
			console.error(err);
		}
		this.setState({originalSchema: schema, newSchema: schema});
	}

	handleSchemaNameUpdate = async (e) => {
		this.setState({statusMessage: null});
		let schemaName = e.currentTarget.value;
		let schema;
		try {
			let res = await fetch('/admin/schemas/get', {
				method: 'POST',
				body: JSON.stringify({SchemaName: schemaName})
			});
			schema = await res.json();
		} catch(err) {
			console.error(err);
			this.setState({statusMessage: {type: 'error', text: 'something went wrong, try again later'}});
			return;
		}
		this.setState({schemaName: schemaName, originalSchema: schema, newSchema: schema});
	}

	handleEditorMount = (editor) => {
		const modifiedEditor = editor.getModifiedEditor();
		modifiedEditor.onDidChangeModelContent((_) => {
			this.setState({newSchema: modifiedEditor.getValue()});
		});
	}

	handleSchemaSaving = async () => {
		let res
		this.setState({statusMessage: null})
		try {
			res = await fetch('/admin/schemas/update', {
				method: 'POST',
				body: JSON.stringify({SchemaName: this.state.schemaName, Schema: this.state.newSchema})
			});
		} catch(err) {
			console.error(err);
			this.setState({statusMessage: {type: 'error', text: 'something went wrong, try again later'}});
			return;
		}
		if (res.status !== 200) {
			this.setState({statusMessage: {type: 'error', text: `unexpected status ${res.status} returned from Chichi`}});
			return;
		}
		this.setState({statusMessage: {type:'success', text:'schema successfully saved'}, originalSchema: this.state.newSchema})
	}

	render() {
		return (
			<div className="SchemaSettings">
				<div className="content">
					<div className="title">Your schemas</div>
					{this.state.statusMessage && <StatusMessage onClose={() => {this.setState({statusMessage: null})}} message={this.state.statusMessage} />}
					<div className="bar">
						<select name="schemaname" id="schemaname" value={this.state.schemaName} onChange={(e) => {this.handleSchemaNameUpdate(e)}}>
							<option value="user">User</option>
							<option value="group">Group</option>
							<option value="event">Event</option>
						</select>
						<div className="apply btn" onClick={this.handleSchemaSaving}>Save</div>
					</div>
					<div className="editor-wrapper">
						<DiffEditor
							language="json"
							original={this.state.originalSchema}
							modified={this.state.newSchema}
							value={this.state.newSchema}
							onMount={this.handleEditorMount}
						/>
					</div>
				</div>
			</div>
		)
	}
}
