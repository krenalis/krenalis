import React from 'react';
import './AccountSource.css';
import NotFound from '../../NotFound/NotFound';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import call from '../../../utils/call';
import { transformationFuncExample } from '../../../utils/docs/transformationFuncExample';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react';
import Editor from '@monaco-editor/react';

export default class AccountSource extends React.Component {

    constructor(props) {
        super(props);
        this.toast = React.createRef();
        this.sourceID = Number(String(window.location).split('/').pop());
        this.state = {
            'source': {},
            'sourceProperties': [],
            'schemaProperties': [],
            'status': null,
            'transformationFunc': '',
            'notFound': false,
        };
    }

    async componentDidMount() {
        let err;

        // get the source.
        let source;
        [source, err] = await call('/admin/data-sources/get', this.sourceID);
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        if (source == null) {
            this.setState({notFound: true});
            return;
        }
        this.setState({ source: source, transformationFunc: source.TransformationFunc });

        // get the source properties.
        let sp;
        [sp, err] = await call('/admin/connectors-properties', { Connector: this.sourceID });
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        let sourceProperties = [];
        for (let p of sp.Properties) sourceProperties.push(p.Name);
        this.setState({ sourceProperties: sourceProperties });

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
        let [, err] = await call('/admin/transformations/update', { Connector: this.sourceID, Transformation: this.state.transformationFunc });
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        this.setState({ status: { variant: 'success', icon: 'check2-circle', text: 'Your transformation function has been saved succesfully' } });
        this.toast.current.toast();
    }

    render() {
        if (this.state.notFound) {
            return <NotFound />
        } else {
            return (
                <div className='AccountSource'>
                    <Breadcrumbs breadcrumbs={[{ Name: 'Your data sources', Link: '/admin/account/sources' }, { Name: `${this.state.source.Name}'s configuration` }]} />
                    <div className='content'>
                        <Toast reactRef={this.toast} status={this.state.status} />
                        <div className='title'>
                            {this.state.source.LogoURL !== '' && <img className='littleLogo' src={this.state.source.LogoURL} alt={`${this.state.source.Name}'s logo`} />}
                            <div className='text'>Map {this.state.source.Name}'s properties to your golden record</div>
                        </div>
                        <div className='properties sourceProperties'>
                            <div className='title'>Source properties</div>
                            {this.state.sourceProperties.map((p, index) => {
                                return <div key={index} className='property'>{p}</div>
                            })}
                        </div>
                        <div className='editorWrapper'>
                            <Editor
                                onChange={(value) => { this.setState({ transformationFunc: value }) }}
                                defaultLanguage='go'
                                value={this.state.transformationFunc}
                                theme='vs-dark'
                            />
                            <SlButton className='saveButton' variant='primary' size='large' onClick={this.handleSaving}>
                                <SlIcon slot='prefix' name='save' />
                                Save
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
