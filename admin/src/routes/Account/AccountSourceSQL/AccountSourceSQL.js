import React from 'react';
import './AccountSourceSQL.css';
import NotFound from '../../NotFound/NotFound';
import Toast from '../../../components/Toast/Toast';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import Grid from '../../../components/Grid/Grid';
import call from '../../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react';
import Editor from '@monaco-editor/react';

const queryMaxSize = 16777215;

export default class AccountSourceSQL extends React.Component {
    
    constructor(props) {
        super(props);
        this.toast = React.createRef();
        this.sourceID = Number(String(window.location).split('/').at(-2));
        this.state = {
            'source': {},
            'status': null,
            'notFound': false,
            'query': '',
            'limit': 20, // TODO(@Andrea): implement as a select
            'table': null
        };
    }
    
    componentDidMount = async () => {
        let [source, err] = await call('/admin/data-sources/get', this.sourceID);
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        if (source == null) {
            this.setState({notFound: true});
            return;
        }
        this.setState({ source: source, query: source.UsersQuery });
    }

    handlePreview = async () => {
        if (this.state.query.length > queryMaxSize) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: 'You query is too long' } });
            this.toast.current.toast();
            return;
        }
        if (!this.state.query.includes(':limit')) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: `your query does not contain the ':limit' placeholder` } });
            this.toast.current.toast();
            return;
        }
        let [table, err] = await call('/admin/data-sources/preview-query', {DataSource: this.state.source.ID, Query: this.state.query, Limit: this.state.limit});
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        if (table.Columns.length === 0) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: 'Your query did not return any columns'} });
            this.toast.current.toast();
            return;
        }
        if (table.Rows.length === 0) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: 'Your query did not return any rows'} });
            this.toast.current.toast();
            return;
        }
        this.setState({table: table});
    }

    saveQuery = async () => {
        if (this.state.query.length > queryMaxSize) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: 'You query is too long' } });
            this.toast.current.toast();
            return;
        }
        if (!this.state.query.includes(':limit')) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: `your query does not contain the ':limit' placeholder` } });
            this.toast.current.toast();
            return;
        }
        let [, err] = await call('/admin/data-sources/set-users-query', {DataSource: this.state.source.ID, Query: this.state.query});
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        this.setState({ status: { variant: 'success', icon: 'check2-circle', text: 'Your query has been successfully saved' } });
        this.toast.current.toast();
    }

    render() {
        if (this.state.notFound) {
            return <NotFound />
        } else {
            return (
                <div className='AccountSourceSQL'>
                    <Breadcrumbs breadcrumbs={[{ Name: 'Your data sources', Link: '/admin/account/sources' }, { Name: `${this.state.source.Name}'s SQL query configuration` }]} />
                    <div className='content'>
                        <Toast reactRef={this.toast} status={this.state.status} />
                        <div className='title'>
                            {this.state.source.LogoURL !== '' && <img className='littleLogo' src={this.state.source.LogoURL} alt={`${this.state.source.Name}'s logo`} />}
                            <div className='text'>Configure your {this.state.source.Name} SQL query</div>
                        </div>
                        <div className='editorWrapper'>
                            <Editor
                                onChange={(value) => {this.setState({ query: value })}}
                                defaultLanguage='sql'
                                value={this.state.query}
                                theme='vs-dark'
                            />
                        </div>
                        <div className="buttons">
                            <SlButton className='previewButton' variant='neutral' size='large' onClick={this.handlePreview}>
                                <SlIcon slot='prefix' name='eye' />
                                Preview
                            </SlButton>
                            <SlButton className='saveButton' variant='primary' size='large' onClick={this.saveQuery}>
                                <SlIcon slot='prefix' name='save' />
                                Save
                            </SlButton>
                        </div>
                    </div>
                    {
                        this.state.table && 
                        <SlDialog label='Users preview' open={true} style={{ '--width': '1200px' }} onSlAfterHide={() => this.setState({table: null})}>
                            <Grid table={this.state.table} />
                        </SlDialog> 
                    }
                </div>
            )
        }
    }
}
