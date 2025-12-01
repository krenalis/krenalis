import React, { useState, useContext, useRef, useLayoutEffect } from 'react';
import './ConnectionPipelines.css';
import Flex from '../../base/Flex/Flex';
import PipelinesGrid from './PipelinesGrid';
import ListTile from '../../base/ListTile/ListTile';
import PipelineTypesDialog from './PipelineTypesDialog';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { Outlet } from 'react-router-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { Pipeline, PipelineType } from '../../../lib/api/types/pipeline';
import { LinkedConnections } from '../ConnectionSettings/LinkedConnections';
import { isEventConnection } from '../../../lib/core/connection';
import Section from '../../base/Section/Section';
import { Snippet } from '../../base/Snippet/Snippet';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const ConnectionPipelines = () => {
	const [isPipelineTypesDialogOpen, setIsPipelineTypesDialogOpen] = useState<boolean>(false);
	const [isPipelineOpen, setIsPipelineOpen] = useState<boolean>(false);
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	const newPipelineID = useRef<number>(0);

	useLayoutEffect(() => {
		const isNew = window.location.search.indexOf('new=true') !== -1;
		if (isNew) {
			setIsLoading(true);
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		}
	}, []);

	useLayoutEffect(() => {
		if (!isPipelineOpen) {
			const id = sessionStorage.getItem('newPipelineID');
			if (id && id !== '') {
				newPipelineID.current = Number(id);
				sessionStorage.removeItem('newPipelineID');
			}
		}
	}, [isPipelineOpen]);

	const onSelectPipelineType = (pipelineType: PipelineType) => {
		let name: string;
		if (pipelineType.target === 'Event') {
			if (pipelineType.eventType) {
				name = `event/${pipelineType.eventType}`;
			} else {
				name = 'event';
			}
		} else {
			name = pipelineType.target.toLowerCase();
		}
		const newLocation = `connections/${connection.id}/pipelines/add/${name}`;
		setIsPipelineTypesDialogOpen(false);
		redirect(newLocation);
	};

	const onSelectPipeline = (pipeline: Pipeline) => {
		const newLocation = `connections/${connection.id}/pipelines/edit/${pipeline.id}`;
		redirect(newLocation);
	};

	if (isLoading) {
		return (
			<SlSpinner
				style={
					{
						display: 'block',
						position: 'relative',
						top: '50px',
						margin: 'auto',
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></SlSpinner>
		);
	}

	let linkedConnections = (
		<Section
			title={connection.isSource ? 'Event destinations' : 'Event sources'}
			description={
				<>
					{connection.isSource
						? 'Select which destinations should receive events from this source.'
						: 'Select which sources should send events to this destination.'}
					<br />
					{connection.isSource
						? 'When you link a destination connection here, events from this source will automatically be forwarded to that destination and processed by its pipelines'
						: 'When you link a source connection here, events from that source will automatically be forwarded to this destination and processed by its pipelines'}
				</>
			}
			annotated={true}
			className={
				connection.isSource
					? 'connection-pipelines__linked-destinations'
					: 'connection-pipelines__linked-sources'
			}
		>
			<LinkedConnections connection={connection} />
		</Section>
	);

	return (
		<div
			className={`connection-pipelines${connection.pipelines!.length === 0 ? ' connection-pipelines--no-pipeline' : ''}`}
		>
			{connection.connector.hasSnippet && (
				<Snippet connectorCode={connection.connector.code} connectionID={connection.id} />
			)}
			{/* Linked connections are shown: before the pipelines, in the case of destination pipelines; after the pipelines,
			in the case of source pipelines. This is to better suggest the usability flow. */}
			{connection.isDestination &&
				isEventConnection(
					'Destination',
					connection.connector.type,
					connection.connector.asDestination.targets,
				) &&
				linkedConnections}
			<Section
				className='connection-pipelines__list'
				title='Pipelines'
				description={`Pipelines import events, users, and groups from a website into the workspace's data warehouse using ${connection.name}`}
				annotated={true}
			>
				{connection.pipelines!.length === 0 ? (
					<div className='connection-pipelines__no-pipeline'>
						<div className='connection-pipelines__no-pipeline-pipeline-types'>
							{connection.pipelineTypes.map((pipelineType) => (
								<ListTile
									key={pipelineType.name}
									icon={<LittleLogo code={connection.connector.code} path={CONNECTORS_ASSETS_PATH} />}
									name={pipelineType.name}
									description={pipelineType.description}
									className={`connection-pipelines__pipeline-type connection-pipelines__pipeline-type--${pipelineType.target.toLowerCase()}`}
									action={
										<SlButton
											size='small'
											variant='primary'
											onClick={() => {
												onSelectPipelineType(pipelineType);
											}}
										>
											Add pipeline...
										</SlButton>
									}
								/>
							))}
						</div>
					</div>
				) : (
					<>
						<Flex alignItems={'center'}>
							<SlButton
								variant='text'
								onClick={() => {
									setIsPipelineTypesDialogOpen(true);
								}}
								className='connection-pipelines__add'
							>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new pipeline
							</SlButton>
						</Flex>
						<PipelinesGrid
							newPipelineID={newPipelineID}
							pipelines={connection.pipelines!}
							onSelectPipeline={onSelectPipeline}
						/>
					</>
				)}
			</Section>
			{/* Linked connections are shown: before the pipelines, in the case of destination pipelines; after the pipelines,
			in the case of source pipelines. This is to better suggest the usability flow. */}
			{connection.isSource &&
				isEventConnection('Source', connection.connector.type, connection.connector.asSource.targets) &&
				linkedConnections}
			<PipelineTypesDialog
				isOpen={isPipelineTypesDialogOpen}
				setIsOpen={setIsPipelineTypesDialogOpen}
				pipelineTypes={connection.pipelineTypes!}
				connection={connection}
				connectionLogo={<LittleLogo code={connection.connector.code} path={CONNECTORS_ASSETS_PATH} />}
				onSelectPipelineType={onSelectPipelineType}
			/>
			<Outlet context={{ setIsPipelineOpen }} />
		</div>
	);
};

export default ConnectionPipelines;
