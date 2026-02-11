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
		setIsPipelineTypesDialogOpen(false);
		const newLocation = `connections/${connection.id}/pipelines/add/${name}`;
		setTimeout(() => {
			redirect(newLocation);
		}, 150);
	};

	const onSelectPipeline = (pipeline: Pipeline) => {
		const newLocation = `connections/${connection.id}/pipelines/edit/${pipeline.id}`;
		redirect(newLocation);
	};

	const joinWithAnd = (items: string[], emptyValue = ''): string => {
		if (items.length === 0) {
			return emptyValue;
		}
		if (items.length === 1) {
			return items[0];
		}
		if (items.length === 2) {
			return `${items[0]} and ${items[1]}`;
		}
		return `${items.slice(0, -1).join(', ')}, and ${items[items.length - 1]}`;
	};
	const joinObjects = (objects: string[]): string => joinWithAnd(objects, 'records');
	const joinActions = (actions: string[]): string => joinWithAnd(actions);

	const usersTerm = connection.connector.terms.users?.trim()
		? connection.connector.terms.users.toLowerCase()
		: 'users';
	const userTerm = connection.connector.terms.user?.trim() ? connection.connector.terms.user.toLowerCase() : 'user';
	const connectionLabelForCopy =
		connection.connector.code === 'javascript' ? 'a website' : connection.connector.label;
	const supportedTargets = new Set(connection.pipelineTypes.map((type) => type.target));
	const supportsUser = supportedTargets.has('User');
	const supportsEvent = supportedTargets.has('Event');
	const supportsGroup = supportedTargets.has('Group');

	const batchObjects: string[] = [];
	if (supportsUser) {
		batchObjects.push(usersTerm);
	}
	if (supportsGroup) {
		batchObjects.push('groups');
	}
	const objectText = joinObjects(batchObjects);

	let pipelinesDescription: string;
	if (connection.isSource) {
		const sourceActions: string[] = [];
		if (supportsEvent) {
			sourceActions.push('collect events');
		}
		if (supportsUser) {
			sourceActions.push(`import ${usersTerm}`);
		}
		if (supportsGroup) {
			sourceActions.push('import groups');
		}
		pipelinesDescription = `Use pipelines to ${joinActions(sourceActions)} from ${connectionLabelForCopy} into your workspace.`;
	} else if (supportsUser && supportsEvent) {
		pipelinesDescription = `Use pipelines to export profiles and send events to ${connectionLabelForCopy}.`;
	} else if (supportsEvent) {
		pipelinesDescription = `Use pipelines to send events from your workspace to ${connectionLabelForCopy}.`;
	} else {
		pipelinesDescription = `Use pipelines to export ${objectText} from your workspace to ${connectionLabelForCopy}.`;
	}

	const pipelinesDocsBaseURL = 'https://www.meergo.com/docs';
	const shouldShowPipelineDocsLinks = !new Set(['dummy', 'ui-sample']).has(connection.connector.code.toLowerCase());
	const buildPipelinesDocsURL = (target: 'users' | 'events'): string => {
		const roleSegment = connection.isSource ? 'sources' : 'destinations';
		return `${pipelinesDocsBaseURL}/ref/admin/pipelines/${roleSegment}-${connection.connector.code}-${target}`;
	};
	const pipelinesDocLinks: Array<{ label: string; href: string }> = [];
	if (shouldShowPipelineDocsLinks) {
		if (connection.isSource) {
			if (supportsEvent) {
				pipelinesDocLinks.push({
					label: `Collect events from ${connectionLabelForCopy}`,
					href: buildPipelinesDocsURL('events'),
				});
			}
			if (supportsUser) {
				pipelinesDocLinks.push({
					label: `Import ${usersTerm} from ${connectionLabelForCopy}`,
					href: buildPipelinesDocsURL('users'),
				});
			}
		} else {
			if (supportsEvent) {
				pipelinesDocLinks.push({
					label: `Activate events in ${connectionLabelForCopy}`,
					href: buildPipelinesDocsURL('events'),
				});
			}
			if (supportsUser) {
				pipelinesDocLinks.push({
					label:
						userTerm === 'customer'
							? `Activate customer profiles in ${connectionLabelForCopy}`
							: `Activate profiles in ${connectionLabelForCopy}`,
					href: buildPipelinesDocsURL('users'),
				});
			}
		}
	}
	const displayedPipelineDocLinks = pipelinesDocLinks.slice(0, 2);

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
						? 'Choose which destinations should receive events from this source.'
						: 'Choose which sources should send events to this destination.'}
					<br />
					<br />
					{connection.isSource
						? 'A source pipeline is not required - incoming events are automatically forwarded and processed by the destination pipelines.'
						: "No source pipeline is required - events are automatically forwarded from the linked sources and processed by this destination's pipelines."}
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
				description={
					<>
						<span>{pipelinesDescription}</span>
						{displayedPipelineDocLinks.map((link) => (
							<a key={link.label} href={link.href} target='_blank' rel='noopener'>
								{link.label}
							</a>
						))}
					</>
				}
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
