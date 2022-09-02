import React, { Component } from 'react'
import AceEditor from 'react-ace';
import { BarChart, YAxis, XAxis, Tooltip, Bar } from 'recharts'
import 'ace-builds/src-noconflict/mode-json';
import 'ace-builds/src-noconflict/theme-github';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { github } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import getChartData from '../../utils/getChartData';
import { suggestionsData } from '../../utils/suggestionsData';
import QuerySuggestion from '../../components/QuerySuggestion/QuerySuggestion';
import StatusMessage from '../../components/StatusMessage/StatusMessage';
import './Dashboard.css'

export default class Dashboard extends Component {

    constructor(props) {
        super(props);
        this.state = {
            jsonQuery: { 'GraphOn': 'PageView', 'GroupBy': 'Month', 'DateRange': 'Past12Months' },
            autoUpdate: false,
        }
        this.aceRef = React.createRef();
    }

    async componentDidMount() {
        let [chartData, error] = await getChartData(this.state.jsonQuery);
        if (error !== null) {
            this.setState({ 'statusMessage': error })
            return;
        }
        this.setState({ 'chartData': chartData });
    }

    runQuery = async () => {
        let query = JSON.parse(this.aceRef.current.editor.getValue());
        let [chartData, error] = await getChartData(query);
        if (error !== null) {
            this.setState({ 'jsonQuery': query, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': query, 'chartData': chartData, 'statusMessage': '' });
    }

    applySuggestion = async (query) => {
        let [chartData, error] = await getChartData(query);
        if (error !== null) {
            this.setState({ 'jsonQuery': query, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': query, 'chartData': chartData, 'statusMessage': '' });
    }

    toggleAutoUpdate = () => {
        this.setState({ 'autoUpdate': !this.state.autoUpdate });
    }

    autoUpdateQuery = async () => {
        if (!this.state.autoUpdate) return;
        let query = JSON.parse(this.aceRef.current.editor.getValue());
        let [chartData, error] = await getChartData(query);
        if (error !== null) {
            this.setState({ 'jsonQuery': query, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': query, 'chartData': chartData, 'statusMessage': '' });
    }

    closeStatusMessage = () => {
        this.setState({ 'statusMessage': '' });
    }

    render() {
        let doc = `{
            "GraphOn": "PageView" | "Click",
            "Filters": [{
                "Column": <a column>,
                "Comparison": "Equal" | "NotEqual",
                "Target": <a value>, 
            }, ...],
            "GroupBy": <a column> | "Day" | "Month",
            "DateRange": "Today" | "Yesterday" 
                        | "Past7Days" | "Past12Months"
        }
        
        columns = "timestamp" | "browser" | "language" 
                    | "referrer" | "session" | "target" | "text" | "title" | "url"
        `;
        let suggestions = [];
        for (let s of suggestionsData) suggestions.push(<QuerySuggestion description={s.description} query={s.query} onClick={this.applySuggestion} />);
        return (
            <div className='Dashboard'>
                <div className='content'>

                    <div className='chart'>
                        <BarChart width={1200} height={250} data={this.state.chartData}>
                            <XAxis dataKey='name' />
                            <YAxis />
                            <Tooltip />
                            <Bar dataKey='p' fill='var(--color-primary)' />
                        </BarChart>
                    </div>

                    {this.state.statusMessage && this.state.statusMessage !== '' && <StatusMessage onClose={this.closeStatusMessage} text={this.state.statusMessage} />}

                    <div className='editor'>

                        <div className="textarea">

                            <div className="controls">
                                <div className="autoapply">
                                    <input type="checkbox" checked={this.state.autoUpdate} onChange={this.toggleAutoUpdate} name="autoapply" id="autoapply" />
                                    <label htmlFor="autoapply">Aggiorna automaticamente</label>
                                </div>
                                <div className="btn" onClick={this.runQuery}>
                                    <i className='material-symbols-outlined'>bolt</i>
                                    Run query
                                </div>
                            </div>

                            <AceEditor
                                ref={this.aceRef}
                                placeholder='Insert your JSON query'
                                mode='json'
                                theme='github'
                                onChange={this.autoUpdateQuery}
                                fontSize={16}
                                showPrintMargin={true}
                                showGutter={true}
                                highlightActiveLine={true}
                                value={JSON.stringify(this.state.jsonQuery, null, 2)}
                                setOptions={{
                                    showLineNumbers: true,
                                    tabSize: 2,
                                    useWorker: false
                                }} />

                        </div>

                        <div className="documentation">
                            <div className="title">Documentazione</div>
                            <SyntaxHighlighter className="documentation" language="json" style={github}>
                                {doc}
                            </SyntaxHighlighter>
                        </div>

                        <div className='suggestions'>
                            <h1 className="title">Suggerimenti</h1>
                            {suggestions}
                        </div>

                    </div>

                </div>
            </div>
        )
    }
}
