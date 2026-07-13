import React, { useContext, useEffect, useState, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
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

	const onChange = (e: any) => {
		const p = structuredClone(pipeline);
		p.requiredConsents = e.target.value;
		setPipeline(p);
	};

	return (
		<Section
			className='pipeline__consents'
			title='Privacy'
			description='Choose which purposes an event must have consent for before it is processed by this pipeline. Leave empty to process every event regardless of consent.'
			padded={true}
			ref={ref}
			annotated={true}
		>
			<SlSelect
				className='pipeline__consents-select'
				multiple
				clearable
				placeholder={purposes.length === 0 ? 'No purposes defined yet' : 'Select the required purposes'}
				value={pipeline.requiredConsents ?? []}
				onSlChange={onChange}
				disabled={purposes.length === 0}
			>
				{purposes.map((p) => (
					<SlOption key={p.id} value={p.id}>
						{p.name}
					</SlOption>
				))}
			</SlSelect>
		</Section>
	);
});

export default PipelineConsents;
