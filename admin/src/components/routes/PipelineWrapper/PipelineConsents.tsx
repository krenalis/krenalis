import React, { useContext, useEffect, useState, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ConsentPurposesOperator } from '../../../lib/api/types/pipeline';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

const PipelineConsents = forwardRef<any>((_, ref) => {
	const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);

	const { api, handleError } = useContext(AppContext);
	const { pipeline, setPipeline } = useContext(PipelineContext);

	useEffect(() => {
		const fetchPurposes = async () => {
			try {
				const res = await api.workspaces.consentPurposes();
				setPurposes(res.purposes);
			} catch (err) {
				handleError(err);
			}
		};
		fetchPurposes();
	}, []);

	const isEnabled = !!pipeline.requiredConsents?.operator;

	const onToggle = (e: any) => {
		const p = structuredClone(pipeline);
		if (e.target.checked) {
			p.requiredConsents = { operator: 'and', purposes: [] };
		} else {
			p.requiredConsents = null;
		}
		setPipeline(p);
	};

	const onChangePurposes = (e: any) => {
		const p = structuredClone(pipeline);
		p.requiredConsents = { ...p.requiredConsents, purposes: e.target.value };
		setPipeline(p);
	};

	const onChangeOperator = (e: any) => {
		const p = structuredClone(pipeline);
		p.requiredConsents = { ...p.requiredConsents, operator: e.target.value as ConsentPurposesOperator };
		setPipeline(p);
	};

	return (
		<Section
			className='pipeline__consents'
			title='Privacy'
			description='Choose whether this pipeline should require consent for specific purposes before processing events.'
			padded={true}
			ref={ref}
			annotated={true}
		>
			<SlCheckbox checked={isEnabled} onSlChange={onToggle}>
				Require user consent
			</SlCheckbox>
			{isEnabled && (
				<div className='pipeline__consents-details'>
					<SlSelect
						className='pipeline__consents-select'
						multiple
						clearable
						placeholder={purposes.length === 0 ? 'No purposes defined yet' : 'Select the required purposes'}
						value={pipeline.requiredConsents?.purposes ?? []}
						onSlChange={onChangePurposes}
						disabled={purposes.length === 0}
					>
						{purposes.map((p) => (
							<SlOption key={p.id} value={p.id}>
								{p.name}
							</SlOption>
						))}
					</SlSelect>
					<div className='pipeline__consents-logical-sentence'>
						An event must have consent for
						<SlSelect
							className='pipeline__consents-logical-select'
							size='small'
							value={pipeline.requiredConsents?.operator || 'and'}
							onSlChange={onChangeOperator}
						>
							<SlOption value='and'>all</SlOption>
							<SlOption value='or'>any</SlOption>
						</SlSelect>
						of the selected purposes to be processed by this pipeline.
					</div>
				</div>
			)}
		</Section>
	);
});

export default PipelineConsents;
