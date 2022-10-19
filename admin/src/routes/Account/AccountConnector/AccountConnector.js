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
            'connectorProperties': [],
            'schemaProperties': [],
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

        // get the user schema properties.
        let schemaProperties;
        [schemaProperties, err] = await call('/admin/user-schema-properties')
        if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
        this.setState({schemaProperties: schemaProperties});

        // get the connector properties.
        let cp;
        [cp, err] = await call('/admin/connectors-properties', {Connector: connectorID});
        if (err !== null) {
			this.setState({status: {type: 'error', text: err}});
			return;
		}
        let connectorProperties = [];
        for (let p of cp) connectorProperties.push(p.Name);
        this.setState({connectorProperties: connectorProperties});
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
		let connectorProperties = [];
        connectorProperties.push(<div className="title">Data source</div>)
        for (let p of this.state.connectorProperties) {
            connectorProperties.push(<div className="property">{p}</div>)
        }
        let schemaProperties = [];
        schemaProperties.push(<div className="title">Golden record</div>)
        for (let p of this.state.schemaProperties) {
            schemaProperties.push(<div className="property">{p}</div>)
        }
        return (
        <div className="AccountConnector">
            <div className="content">
                {this.state.statusMessage && <StatusMessage onClose={() => {this.setState({statusMessage: null})}} message={this.state.statusMessage} />}
                <h1>Map data source's properties to the golden record</h1>
                <div className="properties ext">
                    {connectorProperties}
                </div>
                <div className="editor-wrapper">
                    <Editor
                        onChange={this.handleEditorChange}
                        defaultLanguage="go"
                        defaultValue={this.state.transformationFunc}
                    />
                    <Button theme="primary" icon="save" text="Save" onClick={this.handleSaving} />
                </div>
                <div className="properties int">
                    {schemaProperties}
                </div>
            </div>
            <div className="content">
                <h1>Documentation</h1>
                <p>A transformation function which can be used with the default schema is the following:</p>
                <pre><code class="documentationExample">
                {`func(input map[string]any, timestamps map[string]time.Time) (map[string]any, map[string]time.Time, error) {
    out := map[string]any{}
    outTimestamps := map[string]time.Time{}
    if firstName, ok := input["firstname"]; ok {
        out["FirstName"] = firstName
        outTimestamps["FirstName"] = timestamps["firstname"]
    }
    if lastName, ok := input["lastname"]; ok {
        out["LastName"] = lastName
        outTimestamps["LastName"] = timestamps["lastname"]
    }
    if email, ok := input["email"]; ok {
        out["Email"] = email
        outTimestamps["Email"] = timestamps["email"]
    }
    return out, outTimestamps, nil
}`}
                </code></pre>
            </div>
        </div>
        )
    }
}
