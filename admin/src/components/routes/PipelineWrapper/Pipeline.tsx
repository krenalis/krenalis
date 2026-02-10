import React, { useState, useContext, useRef } from 'react';
import './Pipeline.css';
import PipelineHeader from './PipelineHeader';
import PipelineTransformation from './PipelineTransformation';
import PipelineFile from './PipelineFile';
import PipelineQuery from './PipelineQuery';
import PipelineFilters from './PipelineFilters';
import PipelineExportMode from './PipelineExportMode';
import PipelineUpdateOnDuplicates from './PipelineUpdateOnDuplicates';
import PipelineMatching from './PipelineMatching';
import PipelineTable from './PipelineTable';
import { usePipeline } from './usePipeline';
import ConnectionContext from '../../../context/ConnectionContext';
import { FullscreenContext } from '../../../context/FullscreenContext';
import PipelineContext from '../../../context/PipelineContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import Section from '../../base/Section/Section';

const Pipeline = ({ pipelineType: providedPipelineType, pipeline: providedPipeline }) => {
	const [showEmptyMatchingError, setShowEmptyMatchingError] = useState<boolean>(false);
	const [isFullscreenTransformationOpen, setIsFullscreenTransformationOpen] = useState<boolean>(false);

	const { connection } = useContext(ConnectionContext);
	const { closeFullscreen } = useContext(FullscreenContext)!;

	const transformationSectionRef = useRef<any>();
	const matchingSectionRef = useRef<any>();

	const handleEmptyMatchingError = () => {
		setShowEmptyMatchingError(true);
		const top = matchingSectionRef.current.getBoundingClientRect().top;
		matchingSectionRef.current.closest('.fullscreen').scrollBy({
			top: top - 130,
			left: 0,
			behavior: 'smooth',
		});
	};

	const onClose = (cb?: (...args: any) => void) => {
		closeFullscreen(cb);
	};

	const {
		isEditing,
		isImport,
		pipeline,
		settings,
		setSettings,
		isLoading,
		pipelineType,
		setPipelineType,
		setPipeline,
		transformationType,
		setTransformationType,
		savePipeline,
		setIsFileChanged,
		setIsFileConnectorLoading,
		isFileConnectorLoading,
		setIsFileConnectorChanged,
		isFileConnectorChanged,
		setIsTableChanged,
		setIsQueryChanged,
		isTransformationHidden,
		isTransformationDisabled,
		selectedInPaths,
		setSelectedInPaths,
		selectedOutPaths,
		setSelectedOutPaths,
		computeAutoSelectedPaths,
		issues,
		setIssues,
		showIssues,
		setShowIssues,
	} = usePipeline(connection, providedPipelineType, providedPipeline);

	if (isLoading) {
		return (
			<div className='pipeline pipeline--loading'>
				<SlSpinner
					style={
						{
							fontSize: '4rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			</div>
		);
	}

	if (pipeline == null || pipelineType == null) return;

	const isFileStorageImport = connection.isFileStorage && connection.isSource;

	return (
		<PipelineContext.Provider
			value={{
				transformationType,
				setTransformationType,
				connection,
				pipeline,
				setPipeline,
				savePipeline,
				settings: settings,
				setSettings: setSettings,
				pipelineType,
				setPipelineType,
				isEditing,
				isImport,
				onClose,
				transformationSectionRef,
				handleEmptyMatchingError,
				showEmptyMatchingError,
				isTransformationHidden,
				isTransformationDisabled,
				isFullscreenTransformationOpen,
				setIsFullscreenTransformationOpen,
				setIsQueryChanged,
				setIsFileChanged,
				setIsFormatLoading: setIsFileConnectorLoading,
				isFormatLoading: isFileConnectorLoading,
				setIsFormatChanged: setIsFileConnectorChanged,
				isFormatChanged: isFileConnectorChanged,
				setIsTableChanged,
				selectedInPaths,
				setSelectedInPaths,
				selectedOutPaths,
				setSelectedOutPaths,
				computeAutoSelectedPaths,
				issues,
				setIssues,
				showIssues,
				setShowIssues,
			}}
		>
			<div className='pipeline'>
				<PipelineHeader />
				<div className='pipeline__body'>
					{pipelineType!.fields.includes('Filter') && !isFileStorageImport && <PipelineFilters />}
					{pipelineType!.fields.includes('Query') && <PipelineQuery />}
					{pipelineType!.fields.includes('File') && <PipelineFile />}
					{pipelineType!.fields.includes('TableName') && <PipelineTable />}
					{(pipelineType!.fields.includes('ExportMode') ||
						pipelineType!.fields.includes('Matching') ||
						pipelineType!.fields.includes('UpdateOnDuplicates')) && (
						<Section
							title='Matching'
							description={
								<>
									<span>
										Configure how profiles are matched to existing records and which action is taken
										when a match is found or not found.
									</span>
									<a
										href='https://www.meergo.com/docs/ref/admin/matching'
										target='_blank'
										rel='noopener'
									>
										Learn more about matching
									</a>
								</>
							}
							padded={true}
							className='pipeline__matching'
							annotated={true}
						>
							{pipelineType!.fields.includes('Matching') && <PipelineMatching ref={matchingSectionRef} />}
							{pipelineType!.fields.includes('ExportMode') && <PipelineExportMode />}
							{pipelineType!.fields.includes('UpdateOnDuplicates') && <PipelineUpdateOnDuplicates />}
						</Section>
					)}
					{pipelineType!.fields.includes('Filter') && isFileStorageImport && !isTransformationHidden && (
						<PipelineFilters ref={transformationSectionRef} />
					)}
					{pipelineType!.fields.includes('Transformation') && !isTransformationHidden && (
						<PipelineTransformation ref={isFileStorageImport ? null : transformationSectionRef} />
					)}
				</div>
			</div>
		</PipelineContext.Provider>
	);
};

export default Pipeline;
