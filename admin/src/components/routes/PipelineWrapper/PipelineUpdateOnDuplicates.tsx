import React, { useContext } from 'react';
import PipelineContext from '../../../context/PipelineContext';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';

const PipelineUpdateOnDuplicates = () => {
	const { pipeline, setPipeline } = useContext(PipelineContext);

	const onChangeUpdateOnDuplicates = (e) => {
		const p = { ...pipeline };
		p.updateOnDuplicates = e.currentTarget.checked;
		setPipeline(p);
	};

	return (
		pipeline.exportMode.includes('Update') && (
			<div className='pipeline__update-on-duplicates'>
				<SlCheckbox checked={pipeline.updateOnDuplicates!} onSlChange={onChangeUpdateOnDuplicates}>
					If a single profile in Meergo matches multiple app users, update them anyway
				</SlCheckbox>
			</div>
		)
	);
};

export default PipelineUpdateOnDuplicates;
