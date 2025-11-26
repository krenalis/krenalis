import React, { ReactNode } from 'react';
import ListTile from '../../base/ListTile/ListTile';
import { PipelineType } from '../../../lib/api/types/pipeline';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import TransformedConnection from '../../../lib/core/connection';

interface PipelineTypesDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	pipelineTypes: PipelineType[];
	connection: TransformedConnection;
	connectionLogo: ReactNode;
	onSelectPipelineType: (pipelineType: PipelineType) => void;
}

const PipelineTypesDialog = ({
	isOpen,
	setIsOpen,
	pipelineTypes,
	connection,
	connectionLogo,
	onSelectPipelineType,
}: PipelineTypesDialogProps) => {
	const standardPipelineTypes: ReactNode[] = [];
	const eventPipelineTypes: ReactNode[] = [];
	for (const type of pipelineTypes) {
		let disablingReason = null;
		if (connection.pipelines != null && type.target === 'Event' && connection.isSource) {
			let importEventPipeline = connection.pipelines.findIndex((p) => p.target === 'Event');
			if (importEventPipeline > -1) {
				disablingReason = 'You can add only one pipeline that imports events';
			}
		}

		const tile = (
			<ListTile
				key={type.name}
				icon={connectionLogo}
				name={type.name}
				description={type.description}
				disablingReason={disablingReason}
				disabled={disablingReason != null}
				onClick={() => {
					onSelectPipelineType(type);
				}}
				action={<SlIcon name='chevron-right' />}
			/>
		);
		if (type.target === 'User' || type.target === 'Group') {
			standardPipelineTypes.push(tile);
		} else {
			eventPipelineTypes.push(tile);
		}
	}

	return (
		<SlDialog
			label='Add pipeline'
			className='connection-pipelines__dialog'
			onSlAfterHide={() => setIsOpen(false)}
			open={isOpen}
			style={{ '--width': '600px' } as React.CSSProperties}
		>
			<div className='connection-pipelines__dialog-pipeline-types'>
				{standardPipelineTypes}
				{eventPipelineTypes.length > 0 && (
					<>
						<div className='connection-pipelines__dialog-event-pipeline-types-title'>Events</div>
						{eventPipelineTypes}
					</>
				)}
			</div>
		</SlDialog>
	);
};

export default PipelineTypesDialog;
