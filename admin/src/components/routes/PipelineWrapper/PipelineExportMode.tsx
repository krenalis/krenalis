import React, { useContext, useMemo } from 'react';
import { EXPORT_MODE_OPTIONS, flattenSchema } from '../../../lib/core/pipeline';
import PipelineContext from '../../../context/PipelineContext';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';

const PipelineExportMode = () => {
	const { pipeline, setPipeline, pipelineType, connection } = useContext(PipelineContext);

	const error = useMemo<string>(() => {
		if (pipeline.matching.out === '' || pipeline.exportMode === 'UpdateOnly') {
			return '';
		}
		// If the export mode is "CreateOnly" or "CreateOrUpdate", the
		// out matching property must be present in the destination
		// schema.
		const flatDestinationSchema = flattenSchema(pipelineType.outputSchema);
		const p = flatDestinationSchema[pipeline.matching.out]?.full;
		if (p == null) {
			return `Since "${pipeline.matching.out}" is set as the ${connection.connector.label}'s matching property and it is read-only, users can only be updated, not created. Change the matching property accordingly or select 'Update only'.`;
		}
	}, [pipeline]);

	const onChangeExportMode = (e) => {
		const p = { ...pipeline };
		p.exportMode = e.currentTarget.value;
		setPipeline(p);
	};

	return (
		<div className='pipeline__export-mode'>
			<SlSelect
				className='pipeline__export-mode-select'
				size='medium'
				label='What can be done with users?'
				value={pipeline.exportMode!}
				onSlChange={onChangeExportMode}
			>
				{Object.keys(EXPORT_MODE_OPTIONS).map((k) => (
					<SlOption key={k} value={k}>
						{EXPORT_MODE_OPTIONS[k]}
					</SlOption>
				))}
			</SlSelect>
			{error != '' && <div className='pipeline__export-mode-error'>{error}</div>}
		</div>
	);
};

export default PipelineExportMode;
