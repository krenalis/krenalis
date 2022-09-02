import React, { Component } from 'react'
import AceEditor from 'react-ace';
import 'ace-builds/src-noconflict/mode-json';
import 'ace-builds/src-noconflict/theme-github';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { github } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import getChartData from '../../utils/getChartData';
import { suggestionsData } from '../../utils/suggestionsData';
import QuerySuggestion from '../../components/QuerySuggestion/QuerySuggestion';
import { docsData } from '../../utils/docsData';
import { additionalNotesData } from '../../utils/additionalNotesData';
import Chart from '../../components/Chart/Chart';
import StatusMessage from '../../components/StatusMessage/StatusMessage';
import Accordion from '../../components/Accordion/Accordion';
import './Dashboard.css'
import FloatingButton from '../../components/FloatingButton/FloatingButton';
import SlideOver from '../../components/SlideOver/SlideOver';

export default class Dashboard extends Component {

    constructor(props) {
        super(props);
        this.state = {
            jsonQuery: suggestionsData[0].jsonQuery,
            autoUpdate: false,
            isSlideOverOpen: false,
            columns: [],
            chartData: [],
            chartType: 'line',
        }
        this.aceRef = React.createRef();
    }

    async componentDidMount() {
        let queryObj = JSON.parse(this.state.jsonQuery);
        let [data, error] = await getChartData(queryObj);
        if (error !== null) {
            this.setState({ 'statusMessage': error })
            return;
        }
        this.setState({ 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery });
    }

    runQuery = async () => {
        let jsonQuery = this.aceRef.current.editor.getValue()
        let queryObj = JSON.parse(jsonQuery);
        let [data, error] = await getChartData(queryObj);
        if (error !== null) {
            this.setState({ 'jsonQuery': jsonQuery, 'statusMessage': error })
            return;
        }
        this.setState({ 'jsonQuery': jsonQuery, 'chartData': data.chartData, 'columns': data.columns, 'sqlQuery': data.sqlQuery, 'statusMessage': '' });
    }

    applySuggestion = async (jsonQuery) => {
        let queryObj = JSON.parse(jsonQuery);
        let [data, error] = await getChartData(queryObj);
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
        let jsonQuery = this.aceRef.current.editor.getValue()
        let queryObj = JSON.parse(jsonQuery);
        let [data, error] = await getChartData(queryObj);
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
        let columnsCells = [];
        for (let c of this.state.columns) columnsCells.push(<div className='head-cell'>{c}</div>)
        for (let entry of this.state.chartData) {
            for (let v in entry) columnsCells.push(<div className='cell'>{entry[v]}</div>)
        }
        let chartOptions = [];
        let chartTypes = ['bar', 'line']
        for (let type of chartTypes) {
            chartOptions.push(<option value={type} selected={type === this.state.chartType ? true : false}>{type}</option>)
        }
        return (
            <div className='Dashboard' >
                <div className='content'>

                    <div className="chart-section">

                        <Chart data={this.state.chartData} type={this.state.chartType} width={1500} height={250} />

                        <select name="chart-type" id="chart-type" onChange={(e) => { this.setState({ 'chartType': e.currentTarget.value }) }}>
                            {chartOptions}
                        </select>

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
                                mode='json'
                                theme='github'
                                onChange={this.autoUpdateQuery}
                                debounceChangePeriod={500}
                                fontSize={16}
                                showPrintMargin={true}
                                showGutter={true}
                                showInvisibles={true}
                                highlightActiveLine={true}
                                value={this.state.jsonQuery}
                                setOptions={{
                                    copyWithEmptySelection: true,
                                    showLineNumbers: true,
                                    useWorker: false,
                                }} />

                            <div className='sql-query'>
                                <SyntaxHighlighter wrapLongLines={true} language='sql' style={github}>
                                    {this.state.sqlQuery}
                                </SyntaxHighlighter>
                            </div>

                            <div className='query-columns' style={{ gridTemplateColumns: `repeat(${this.state.columns.length}, 1fr)` }}>{columnsCells}</div>

                        </div>

                        <div className="side">

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
