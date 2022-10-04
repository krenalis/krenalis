import React, { Component } from 'react';
import './AccountConnector.css';
import Editor from "@monaco-editor/react";
import StatusMessage from '../../components/StatusMessage/StatusMessage';

const mockProperties = {
    external: ['first_name', 'last_name', 'email', 'phone_number', 'property_1'],
    internal: ['Email', 'FirstName', 'LastName', 'PhoneNumber', 'Age']
}

// TODO(@Andrea): highlight the property when the user is writing about it in
// the current editor line.
export default class AccountConnector extends Component {
  
    constructor(props) {
        super(props);
        this.state = {
            'statusMessage': null,
            'transformationFunc': '',
        }
    }

    async componentDidMount() {
        let connectorID = String(window.location).split('/').pop();
        let transformationFunc;
        try {
            let res = await fetch('/admin/transformations/get', {
                method: 'POST',
                body: JSON.stringify({Connector: Number(connectorID)}),
            });
            transformationFunc = await res.json();
        } catch(err) {
            console.error(err);
            this.setState({statusMessage: {type: 'error', text: 'Something went wrong, try again later'}});
            return
        }
        this.setState({transformationFunc: transformationFunc});
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
        for (let p of mockProperties.external) {
            externalProperties.push(<div className="property">{p}</div>)
        }
        let internalProperties = [];
        internalProperties.push(<div className="title">Golden record</div>)
        for (let p of mockProperties.internal) {
            internalProperties.push(<div className="property">{p}</div>)
        }
        return (
        <div className="AccountConnector">
            <div className="content">
                {this.state.statusMessage && <StatusMessage onClose={() => {this.setState({statusMessage: null})}} message={this.state.statusMessage} />}
                <div className="title">Map connector's properties to your golden record</div>
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
                <div className="btn save" onClick={this.handleSaving}>Save</div>
            </div>
        </div>
        )
    }
}
