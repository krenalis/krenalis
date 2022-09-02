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
import { docsData } from '../../utils/docsData';
import { additionalNotesData } from '../../utils/additionalNotesData';
import StatusMessage from '../../components/StatusMessage/StatusMessage';
import Accordion from '../../components/Accordion/Accordion';
import './Dashboard.css'
import FloatingButton from '../../components/FloatingButton/FloatingButton';
import SlideOver from '../../components/SlideOver/SlideOver';

export default class Dashboard extends Component {

    constructor(props) {
        super(props);
        this.state = {
            jsonQuery: {
                Graph: ['Count', 'Pageview'],
                GroupBy: ['Month'],
                DateRange: 'Past12Months',
            },
            autoUpdate: false,
            isSlideOverOpen: false,
        }
        this.aceRef = React.createRef();
    }

    async componentDidMount() {
        let [data, error] = await getChartData(this.state.jsonQuery);
        if (error !== null) {
            this.setState({ 'statusMessage': error })
            return;
        }
        this.setState({ 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery });
    }

    runQuery = async () => {
        let jsonQuery = JSON.parse(this.aceRef.current.editor.getValue());
        let [data, error] = await getChartData(jsonQuery);
        if (error !== null) {
            this.setState({ 'jsonQuery': jsonQuery, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': jsonQuery, 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery, 'statusMessage': '' });
    }

    applySuggestion = async (jsonQuery) => {
        let [data, error] = await getChartData(jsonQuery);
        if (error !== null) {
            this.setState({ 'jsonQuery': jsonQuery, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': jsonQuery, 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery, 'statusMessage': '' });
    }

    toggleAutoUpdate = () => {
        this.setState({ 'autoUpdate': !this.state.autoUpdate });
    }

    autoUpdateQuery = async () => {
        if (!this.state.autoUpdate) return;
        let jsonQuery = JSON.parse(this.aceRef.current.editor.getValue());
        let [data, error] = await getChartData(jsonQuery);
        if (error !== null) {
            this.setState({ 'jsonQuery': jsonQuery, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': jsonQuery, 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery, 'statusMessage': '' });
    }

    closeStatusMessage = () => {
        this.setState({ 'statusMessage': '' });
    }

    toggleSlideOver = () => {
        this.setState({ 'isSlideOverOpen': !this.state.isSlideOverOpen })
    }

    render() {
        let suggestions = [];
        for (let s of suggestionsData) suggestions.push(<QuerySuggestion description={s.description} query={s.jsonQuery} onClick={this.applySuggestion} />);
        return (
            <div className='Dashboard'>
                <div className='content'>

                    <div className='chart'>
                        <BarChart width={1500} height={250} data={this.state.chartData}>
                            <XAxis dataKey='name' />
                            <YAxis />
                            <Tooltip />
                            <Bar dataKey='p' fill='var(--color-primary)' />
                        </BarChart>
                    </div>

                    {this.state.statusMessage && this.state.statusMessage !== '' && <StatusMessage onClose={this.closeStatusMessage} text={this.state.statusMessage} />}

                    <div className='editor'>

                        <div className='textarea'>

                            <div className='controls'>
                                <div className='autoapply'>
                                    <input type='checkbox' checked={this.state.autoUpdate} onChange={this.toggleAutoUpdate} name='autoapply' id='autoapply' />
                                    <label htmlFor='autoapply'>Aggiorna automaticamente</label>
                                </div>
                                <div className='btn' onClick={this.runQuery}>
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

                            <div className="sql-query">
                                <SyntaxHighlighter wrapLongLines={true} language='sql' style={github}>
                                    {this.state.sqlQuery}
                                </SyntaxHighlighter>
                            </div>

                            <div className="columns">{this.state.columns}</div>

                        </div>

                        <Accordion className='documentation' title={'Documentazione'}>
                            <SyntaxHighlighter className='documentation' language='json' style={github}>
                                {docsData}
                            </SyntaxHighlighter>
                        </Accordion>

                        <div className='suggestions'>
                            <h1 className='title'>Suggerimenti</h1>
                            {suggestions}
                        </div>

                    </div>

                    <FloatingButton icon={'speaker_notes'} onClick={this.toggleSlideOver} />

                    <SlideOver onClose={this.toggleSlideOver} title={'Note aggiuntive'} isOpen={this.state.isSlideOverOpen ? true : false}>
                        {additionalNotesData}
                    </SlideOver>

                </div>
            </div>
        )
    }
}
