import React from 'react';
import './AccountConnectionProperties.css';
import NotFound from '../../NotFound/NotFound';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import call from '../../../utils/call';
import { transformationFuncExample } from '../../../utils/docs/transformationFuncExample';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react';
import Editor from '@monaco-editor/react';

export default class AccountConnection extends React.Component {

    constructor(props) {
        super(props);
        this.toast = React.createRef();
        this.connectionID = Number(String(window.location).split('/').at(-2));
        this.state = {
            'connection': {},
            'connectionProperties': [],
            'schemaProperties': [],
            'status': null,
            'transformationFunc': '',
            'notFound': false,
        };
    }

    async componentDidMount() {
        let err;

        // get the connection.
        let connection;
        [connection, err] = await call('/admin/connections/get', this.connectionID);
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        if (connection == null) {
            this.setState({notFound: true});
            return;
        }
        this.setState({ connection: connection, transformationFunc: connection.TransformationFunc });

        // get the connection properties.
        let c;
        [c, err] = await call('/admin/connectors-properties', { Connector: this.connectionID });
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        let connectionProperties = [];
        for (let p of c.Properties) connectionProperties.push(p.Name);
        this.setState({ connectionProperties: connectionProperties });

        // get the user schema properties.
        let schemaProperties;
        [schemaProperties, err] = await call('/admin/user-schema-properties');
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        this.setState({ schemaProperties: schemaProperties });
    }

    handleSaving = async (e) => {
        let [, err] = await call('/admin/transformations/update', { Connector: this.connectionID, Transformation: this.state.transformationFunc });
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        this.setState({ status: { variant: 'success', icon: 'check2-circle', text: 'Your transformation function has been saved successfully' } });
        this.toast.current.toast();
    }

    render() {
        if (this.state.notFound) {
            return <NotFound />
        } else {
            return (
                <div className='AccountConnectionProperties'>
                    <Breadcrumbs breadcrumbs={[{ Name: 'Your connections', Link: '/admin/account/connections' }, { Name: `${this.state.connection.Name}'s configuration` }]} />
                    <div className='content'>
                        <Toast reactRef={this.toast} status={this.state.status} />
                        <div className='title'>
                            {this.state.connection.LogoURL !== '' && <img className='littleLogo' src={this.state.connection.LogoURL} alt={`${this.state.connection.Name}'s logo`} />}
                            <div className='text'>(obsolete page) Map {this.state.connection.Name}'s properties to your golden record</div>
                        </div>
                        <div className='properties connectionProperties'>
                            <div className='title'>Connection properties</div>
                            {this.state.connectionProperties.map((p, index) => {
                                return <div key={index} className='property'>{p}</div>
                            })}
                        </div>
                        <div className="editor">
                            <div className='editorWrapper'>
                                <Editor
                                    onChange={(value) => { this.setState({ transformationFunc: value }) }}
                                    defaultLanguage='go'
                                    value={this.state.transformationFunc}
                                    theme='vs-dark'
                                />
                            </div>
                            <SlButton className='saveButton' disabled variant='primary' size='large' onClick={this.handleSaving}>
                                <SlIcon slot='prefix' name='save' />
                                Save (obsolete)
                            </SlButton>
                            <div className='documentation'>
                                <p>A transformation function which can be used with the default schema:</p>
                                <pre><code class='transformationFunc'>{transformationFuncExample}</code></pre>
                            </div>
                        </div>
                        <div className='properties schemaProperties'>
                            <div className='title'>Golden record properties</div>
                            {this.state.schemaProperties.map((p, index) => {
                                return <div key={index} className='property'>{p}</div>
                            })}
                        </div>
                        
                    </div>
                </div>
            )
        }
    }
}
