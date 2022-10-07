import React, { Component } from 'react';

import './AccountConnector.css';
import StatusMessage from '../../../components/StatusMessage/StatusMessage';
import Button from '../../../components/Button/Button'
import call from '../../../utils/call'

import Editor from "@monaco-editor/react";

export default class AccountConnector extends Component {
  
    constructor(props) {
        super(props);
        this.state = {
            'statusMessage': null,
            'transformationFunc': '',
            'connectorsProperties': ['last_name', 'first_name', 'phone_number', 'email', 'age'],
            'internalProperties': [],
        }
    }

    async componentDidMount() {
        let connectorID = Number(String(window.location).split('/').pop());
        let err;

        // get the transformation function.
        let transformationFunc;
        [transformationFunc, err] = await call('/admin/transformations/get', {Connector: connectorID});
		if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
		this.setState({transformationFunc: transformationFunc});

        // get the properties.
        let properties;
        [properties, err] = await call('/admin/user-schema-properties')
        if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
        this.setState({internalProperties: properties});
    }
  
    // TODO(@Andrea): how to set debounce?
    handleEditorChange = (value, e) => {
        this.setState({transformationFunc: value});
        return;
    }

    handleSaving = async (e) => {
        let connectorID = String(window.location).split('/').pop();
        this.setState({statusMessage: null});
        let res;
        try {
            res = await fetch('/admin/transformations/update', {
                method: 'POST',
                body: JSON.stringify({Connector: Number(connectorID), Transformation: this.state.transformationFunc}),
            });
        } catch(err) {
            console.error(err);
            this.setState({statusMessage: {type: 'error', text: err.message}});
            return
        }
        if (res.status !== 200) {
            this.setState({statusMessage: {type: 'error', text: `Unexpected status ${res.status} returned by Chichi`}});
            return;
        }
        this.setState({statusMessage: {type: 'success', text: 'Your transformation function has been saved succesfully'}});
    }

    render() {
		let externalProperties = [];
        externalProperties.push(<div className="title">Connector</div>)
        for (let p of this.state.connectorsProperties) {
            externalProperties.push(<div className="property">{p}</div>)
        }
        let internalProperties = [];
        internalProperties.push(<div className="title">Golden record</div>)
        for (let p of this.state.internalProperties) {
            internalProperties.push(<div className="property">{p}</div>)
        }
        return (
        <div className="AccountConnector">
            <div className="content">
                {this.state.statusMessage && <StatusMessage onClose={() => {this.setState({statusMessage: null})}} message={this.state.statusMessage} />}
                <h1>Map connector's properties to your golden record</h1>
                <div className="properties ext">
                    {externalProperties}
                </div>
                <div className="editor-wrapper">
                    <Editor
                        onChange={this.handleEditorChange}
                        defaultLanguage="go"
                        defaultValue={this.state.transformationFunc}
                    />
                </div>
                <div className="properties int">
                    {internalProperties}
                </div>
                <Button theme="primary" icon="save" text="Save" onClick={this.handleSaving} />
            </div>
        </div>
        )
    }
}
