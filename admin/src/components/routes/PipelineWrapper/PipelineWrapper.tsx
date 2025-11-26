import React, { useState, useEffect, useContext } from 'react';
import Pipeline from './Pipeline';
import Fullscreen from '../../base/Fullscreen/Fullscreen';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { useParams, useLocation, useOutletContext } from 'react-router-dom';
import { Pipeline as PipelineInterface, PipelineType } from '../../../lib/api/types/pipeline';

const PipelineWrapper = () => {
	const [selectedPipelineType, setSelectedPipelineType] = useState<PipelineType>();
	const [selectedPipeline, setSelectedPipeline] = useState<PipelineInterface>();

	const params = useParams();
	const location = useLocation();

	const { setIsPipelineOpen } = useOutletContext<any>();
	const { setIsLoadingConnections, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useEffect(() => {
		setIsPipelineOpen(true);
	}, []);

	useEffect(() => {
		const splitted = location.pathname.split('/');
		const instructionsStartIndex = splitted.findIndex((fragment) => fragment === 'pipelines') + 1;
		const instructions = splitted.slice(instructionsStartIndex);
		const isEditing = instructions[0] === 'edit';
		if (isEditing) {
			const pipeline = connection.pipelines!.find((p) => String(p.id) === params.pipeline);
			setSelectedPipeline(pipeline);
			return;
		} else {
			let pipelineType: PipelineType | undefined;
			const isEvent = instructions.includes('event');
			if (isEvent) {
				if (instructions.length === 3) {
					const eventType = instructions[instructions.length - 1];
					pipelineType = connection.pipelineTypes!.find((p) => p.eventType === eventType);
				} else {
					pipelineType = connection.pipelineTypes!.find((p) => p.target === 'Event' && p.eventType === null);
				}
			} else {
				const target = instructions[instructions.length - 1];
				const capitalized = target.charAt(0).toUpperCase() + target.slice(1);
				pipelineType = connection.pipelineTypes!.find((p) => p.target === capitalized);
			}
			if (pipelineType == null) {
				console.error(`Pipeline type for instructions ${instructions} does not exist anymore`);
				return;
			}
			setSelectedPipelineType(pipelineType);
			return;
		}
	}, [params, location]);

	const onClose = () => {
		setIsLoadingConnections(true);
		redirect(`connections/${connection.id}/pipelines`);
		setIsPipelineOpen(false);
	};

	const isLoading = selectedPipelineType == null && selectedPipeline == null;
	return (
		<Fullscreen onClose={onClose} isLoading={isLoading} className='fullscreen-pipeline'>
			<Pipeline pipelineType={selectedPipelineType} pipeline={selectedPipeline} />
		</Fullscreen>
	);
};

export default PipelineWrapper;
