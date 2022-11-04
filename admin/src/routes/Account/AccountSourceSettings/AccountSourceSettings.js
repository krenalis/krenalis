import React from 'react';
import './AccountSourceSettings.css';
import NotFound from '../../NotFound/NotFound';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import Toast from '../../../components/Toast/Toast';
import call from '../../../utils/call';
import { renderConnectorComponent } from '../../../components/ConnectorSettings/renderConnectorComponent';
import { SlButton } from '@shoelace-style/shoelace/dist/react';

export default class AccountSourceSettings extends React.Component {
    
    constructor(props) {
        super(props);
        this.toast = React.createRef();
        this.sourceID = Number(String(window.location).split('/').at(-2));
        this.state = {
            source: {},
            settings: {Components:null, Actions:null},
            form:{},
            notFound: false,
        };
    }

    componentDidMount = async () => {
        let err, source, settings;

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
        this.setState({ source: source});

        [settings, err] = await call('/admin/connectors/ui', this.sourceID);
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        let form = {};
        if (settings.Fields != null) {
            for (let f of settings.Fields) { form[f.Name] = f.Value; }
        }
        this.setState({ settings: settings, form: form});
    }

    onComponentChange = (name, value) => {
        let form = { ...this.state.form };
        form[name] = value;
        this.setState({form: form});
    }

    onActionClick = async (event) => {
        let [settings, err] = await call('/admin/connectors/ui-event', {datasource: this.sourceID, event: event, form: this.state.form});
        if (err !== null) {
            this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
            this.toast.current.toast();
            return;
        }
        if (settings == null) {
            this.setState({ status: { variant: 'success', icon: 'check2-circle', text: 'Success' } });
            this.toast.current.toast();
            return;
        }
        let form = {};
        if (settings.Components != null) {
            for (let c of settings.Components) { form[c.Name] = c.Value; }
        }
        this.setState({ settings: settings, form: form});
    }
    
    render() {
        if (this.state.notFound) {
            return <NotFound />
        } else {
            return (
                <div className="AccountSourceSettings">
                    <Breadcrumbs breadcrumbs={[{ Name: 'Your data sources', Link: '/admin/account/sources' }, { Name: `${this.state.source.Name}'s settings` }]} />
                    <div className="content">
                        <Toast reactRef={this.toast} status={this.state.status} />
                        <div className='title'>
                            {this.state.source.LogoURL !== '' && <img className='littleLogo' src={this.state.source.LogoURL} alt={`${this.state.source.Name}'s logo`} />}
                            <div className='text'>Configure {this.state.source.Name}</div>
                        </div>
                        <div className="settings">
                            <div className="components">
                                {this.state.settings.Fields != null && this.state.settings.Fields.map((c, i) => renderConnectorComponent(c, this.onComponentChange))}
                            </div>
                            <div className="actions">
                                {this.state.settings.Actions != null && this.state.settings.Actions.map((a, i) => <SlButton variant={a.Variant} onClick={ async () => { await this.onActionClick(a.Event) }}>{a.Text}</SlButton>)}
                            </div>
                        </div>
                    </div>
                </div>
            )
        }
    }
}
