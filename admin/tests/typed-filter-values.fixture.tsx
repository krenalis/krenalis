import React, { useEffect, useMemo, useRef, useState } from 'react';
import ReactDOM from 'react-dom/client';
import JSONbig from 'json-bigint';
import '@shoelace-style/shoelace/dist/themes/light.css';
import '../src/components/routes/PipelineWrapper/Pipeline.css';
import API from '../src/lib/api/api';
import AppContext from '../src/context/AppContext';
import ConnectionContext from '../src/context/ConnectionContext';
import PipelineContext from '../src/context/PipelineContext';
import PipelineFilters from '../src/components/routes/PipelineWrapper/PipelineFilters';
import PipelinesGrid from '../src/components/routes/ConnectionPipelines/PipelinesGrid';
import { ObjectType, Property } from '../src/lib/api/types/types';
import { Filter, Pipeline } from '../src/lib/api/types/pipeline';
import {
	prepareEditorFilter,
	TransformedPipeline,
	TransformedPipelineType,
	transformPipeline,
} from '../src/lib/core/pipeline';
import TransformedConnection from '../src/lib/core/connection';
import { encodeFilter } from '../src/utils/filters';
import useEventListener from '../src/hooks/useEventListener';

const schema: ObjectType = {
	kind: 'object',
	properties: [
		{ name: 'properties', type: { kind: 'json' } },
		{ name: 'account', type: { kind: 'int', bitSize: 64, unsigned: false } },
		{ name: 'unsigned', type: { kind: 'int', bitSize: 64, unsigned: true } },
	].map(
		(p) =>
			({
				prefilled: '',
				createRequired: false,
				updateRequired: false,
				readOptional: true,
				nullable: false,
				displayName: '',
				description: '',
				...p,
			}) as Property,
	),
};
const pipelineType: TransformedPipelineType = {
	name: 'Typed filters',
	target: 'Event',
	description: '',
	eventType: null,
	inputSchema: schema,
	outputSchema: schema,
	inputMatchingSchema: null,
	outputMatchingSchema: null,
	fields: ['Filter'],
};
const api = new API('https://krenalis.test', 'workspace');
const INITIAL_FILTER =
	'{"operator":"and","rules":[' +
	'{"property":"properties.value","operator":"is one of","values":[9007199254740993,"100",true,false,null,""]},' +
	'{"property":"account","operator":"is","values":[9223372036854775807]},' +
	'{"property":"unsigned","operator":"is","values":[18446744073709551615]}]}';
const INITIAL_PIPELINE =
	'{"id":"pipeline","target":"Event","name":"Typed filters","enabled":false,' +
	'"eventType":null,"inSchema":' +
	JSON.stringify(schema) +
	',"outSchema":null,"transformation":null,"filter":' +
	INITIAL_FILTER +
	'}';

interface PreviewControlsProps {
	filter: Filter | null;
	error: string | null;
	onError: (error: string) => void;
}

const PreviewControls = ({ filter, error, onError }: PreviewControlsProps) => {
	const { startListening, stopListening } = useEventListener(
		() => undefined,
		undefined,
		'connection',
		filter,
		null,
		error == null,
	);
	return (
		<>
			<button
				onClick={() => {
					if (error != null) onError(error);
					else startListening();
				}}
			>
				Preview typed filter
			</button>
			<button onClick={stopListening}>Stop preview</button>
		</>
	);
};

const Harness = () => {
	const [loaded, setLoaded] = useState<Pipeline | null>(null);
	const [pipeline, setPipeline] = useState<TransformedPipeline | undefined>();
	const [error, setError] = useState('');
	const [payload, setPayload] = useState('');
	const [preview, setPreview] = useState('');
	const runPipelineButtonRefs = useRef({});
	const runPipelineDropdownButtonRefs = useRef({});
	const [connection] = useState(
		() =>
			({
				id: 'connection',
				name: 'SDK',
				isSDK: true,
				isSource: true,
				isEventBased: true,
				isDestination: false,
				isApplication: false,
				isDatabase: false,
				isFileStorage: false,
				connector: { type: 'SDK' },
				pipelineTypes: [pipelineType],
				pipelines: [],
			}) as unknown as TransformedConnection,
	);

	useEffect(() => {
		const originalFetch = window.fetch;
		let storedPipeline = INITIAL_PIPELINE;
		window.fetch = async (input, init) => {
			const url = String(input);
			let body = '{"id":"pipeline"}';
			if (url.endsWith('/connections') && init?.method === 'GET')
				body = '{"connections":[{"pipelines":[' + storedPipeline + ']}]}';
			if (url.endsWith('/events/listeners') && init?.method === 'POST') {
				setPreview(String(init.body));
				body = '{"id":"listener"}';
			} else if (init?.method === 'PUT') {
				setPayload(String(init.body));
				storedPipeline = JSONbig.stringify({
					...JSONbig.parse(storedPipeline),
					...JSONbig.parse(String(init.body)),
				});
			}
			if (url.endsWith('/events/listeners/listener')) body = '{"events":[],"omitted":0}';
			return new Response(body, { status: 200, headers: { 'Content-Type': 'application/json' } });
		};
		api.workspaces.connections.find().then((connections) => setLoaded(connections[0].pipelines[0]));
		return () => {
			window.fetch = originalFetch;
		};
	}, []);

	const prepared = useMemo(() => {
		try {
			return { filter: pipeline == null ? null : prepareEditorFilter(pipeline.filter, schema), error: null };
		} catch (error) {
			return { filter: null as Filter, error };
		}
	}, [pipeline]);
	const save = async () => {
		try {
			const filter = prepareEditorFilter(pipeline.filter, schema);
			await api.workspaces.connections.updatePipeline('pipeline', {
				name: pipeline.name,
				filter,
				compression: '',
				format: '',
			});
			setError('');
		} catch (error) {
			setError(error.message);
		}
	};
	return (
		<AppContext.Provider
			value={
				{
					api,
					handleError: (error) => setError(String(error)),
					connectors: [],
					runPipelineButtonRefs,
					runPipelineDropdownButtonRefs,
				} as React.ContextType<typeof AppContext>
			}
		>
			<ConnectionContext.Provider value={{ connection, setConnection: () => undefined }}>
				{loaded != null && pipeline == null && (
					<PipelinesGrid
						newPipelineID={{ current: '' }}
						pipelines={[loaded]}
						onSelectPipeline={(value) => setPipeline(transformPipeline(value, schema, false, schema))}
					/>
				)}
				{pipeline != null && (
					<PipelineContext.Provider
						value={
							{
								pipeline,
								setPipeline,
								pipelineType,
								connection,
								isImport: true,
								isTransformationDisabled: false,
							} as React.ContextType<typeof PipelineContext>
						}
					>
						<PipelineFilters />
						<button onClick={save}>Save typed filter</button>
						<PreviewControls
							filter={prepared.filter}
							error={prepared.error?.message ?? null}
							onError={setError}
						/>
						<button
							onClick={async () => {
								const connections = await api.workspaces.connections.find();
								setLoaded(connections[0].pipelines[0]);
								setPipeline(undefined);
							}}
						>
							Reload from API
						</button>
						<output data-testid='prepared'>
							{prepared.error == null && prepared.filter != null
								? encodeFilter(prepared.filter)
								: 'invalid'}
						</output>
					</PipelineContext.Provider>
				)}
				<output data-testid='payload'>{payload}</output>
				<output data-testid='preview'>{preview}</output>
				<output data-testid='error'>{error}</output>
			</ConnectionContext.Provider>
		</AppContext.Provider>
	);
};
ReactDOM.createRoot(document.getElementById('root')!).render(<Harness />);
