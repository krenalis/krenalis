import React, { forwardRef, useContext, useRef } from 'react';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaButtonElement from '@awesome.me/webawesome/dist/components/button/button.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import { FilterEditor } from '../../base/FilterEditor/FilterEditor';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import { Filter } from '../../../lib/api/types/pipeline';
import { pipelineObjectLabels } from './Pipeline.helpers';

const HIDDEN_KPID_PROPERTIES = ['kpid'];

const PipelineFilters = forwardRef<any>((_, ref) => {
	const { pipeline, setPipeline, pipelineType, connection, isTransformationDisabled, isImport } =
		useContext(PipelineContext);
	const [filterSubject] = pipelineObjectLabels(connection, pipeline, 'plural');
	const filterSubjectTerm = filterSubject.toLowerCase();
	const actionVerb = isImport ? 'import' : pipelineType.target === 'Event' ? 'send' : 'export';
	const isEventBasedUserImport = connection.isEventBased && connection.isSource && pipeline.target === 'User';
	const isAppEventsExport = connection.isApplication && connection.isDestination && pipeline.target === 'Event';
	const isEventImport = connection.isSource && pipeline.target === 'Event';
	const propertiesToHide =
		isEventBasedUserImport || isAppEventsExport || isEventImport ? HIDDEN_KPID_PROPERTIES : undefined;
	const isDisabled = connection.isFileStorage && connection.isSource && isTransformationDisabled;
	const addFilterButtonRef = useRef<WaButtonElement>(null);

	const onChange = (filter: Filter | null) => setPipeline({ ...pipeline, filter });
	const onAddFilter = () =>
		onChange({
			operator: 'and',
			rules: [{ property: '', operator: '', values: [''] }],
		});

	return (
		<Section
			className={`pipeline__filters${isDisabled ? ' pipeline__filters--disabled' : ''}`}
			title='Filters'
			description={
				<>
					<span>{`Choose which ${filterSubjectTerm} to ${actionVerb}. Leave empty to ${actionVerb} all ${filterSubjectTerm}.`}</span>
					<a href='https://www.krenalis.com/docs/ref/admin/filters' target='_blank' rel='noopener'>
						Learn more about filters
					</a>
				</>
			}
			padded={true}
			ref={ref}
			annotated={true}
		>
			{pipeline.filter == null && (
				<WaButton
					ref={addFilterButtonRef}
					className='pipeline__filters-add-condition'
					size='m'
					variant='brand'
					appearance='plain'
					onClick={onAddFilter}
					disabled={isDisabled}
				>
					<WaIcon slot='start' name='plus-circle' />
					Add filter
				</WaButton>
			)}
			<FilterEditor
				disabled={isDisabled}
				emptyFilterFocusRef={addFilterButtonRef}
				filter={pipeline.filter}
				onChange={onChange}
				propertiesToHide={propertiesToHide}
				role={connection.role}
				schema={pipelineType.inputSchema}
				subject={filterSubject}
				target={pipeline.target}
			/>
		</Section>
	);
});

export default PipelineFilters;
