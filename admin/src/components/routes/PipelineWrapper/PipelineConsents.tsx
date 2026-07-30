import React, { useContext, useEffect, useState, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ConsentPurposesOperator } from '../../../lib/api/types/pipeline';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

const PipelineConsents = forwardRef<any>((_, ref) => {
	const { pipeline, setPipeline } = useContext(PipelineContext);

	const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
	const [arePurposesLoaded, setArePurposesLoaded] = useState(false);
	const [isEnabled, setIsEnabled] = useState((pipeline.requiredConsents?.purposes.length ?? 0) > 0);

	const { api, handleError } = useContext(AppContext);

	useEffect(() => {
		setIsEnabled((pipeline.requiredConsents?.purposes.length ?? 0) > 0);
	}, [pipeline.id]);

	useEffect(() => {
		const fetchPurposes = async () => {
			try {
				const res = await api.workspaces.consentPurposes();
				setPurposes(res.purposes);
				setArePurposesLoaded(true);
			} catch (err) {
				handleError(err);
			}
		};
		fetchPurposes();
	}, []);

	const selectedCodes = pipeline.requiredConsents?.purposes ?? [];

	// undefinedCodes are the required codes that are no longer the code of a
	// purpose of the workspace, because the purpose has been deleted or its
	// code has been changed. The pipeline does not process any event until they
	// are removed, so they are listed as options to let the user remove them.
	const undefinedCodes = arePurposesLoaded
		? selectedCodes.filter((code) => !purposes.some((p) => p.code === code))
		: [];

	const onToggle = (e: any) => {
		const p = structuredClone(pipeline);
		setIsEnabled(e.target.checked);
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
			<div className='pipeline__consents-toggle'>
				<SlCheckbox checked={isEnabled} onSlChange={onToggle} disabled={purposes.length === 0} />
				<div className='pipeline__consents-logical-sentence'>
					An event must have consent for
					<SlSelect
						className='pipeline__consents-logical-select'
						size='small'
						value={pipeline.requiredConsents?.operator || 'and'}
						onSlChange={onChangeOperator}
						disabled={!isEnabled}
					>
						<SlOption value='and'>all</SlOption>
						<SlOption value='or'>any</SlOption>
					</SlSelect>
					of the selected purposes to be processed by this pipeline.
				</div>
			</div>
			<div className='pipeline__consents-details'>
				<SlSelect
					className='pipeline__consents-select'
					multiple
					clearable
					placeholder={purposes.length === 0 ? 'No purposes defined yet' : 'Select the required purposes'}
					value={selectedCodes}
					onSlChange={onChangePurposes}
					disabled={!isEnabled || (purposes.length === 0 && undefinedCodes.length === 0)}
				>
					{purposes.map((p) => (
						<SlOption key={p.code} value={p.code}>
							{p.name}
						</SlOption>
					))}
					{undefinedCodes.map((code) => (
						<SlOption key={code} value={code} className='pipeline__consents-undefined-option'>
							{code}
						</SlOption>
					))}
				</SlSelect>
				{undefinedCodes.length > 0 && (
					<div className='pipeline__consents-warning'>
						<SlIcon slot='icon' name='exclamation-triangle' />
						{undefinedCodes.length === 1
							? `The "${undefinedCodes[0]}" purpose no longer exists. This pipeline does not process any event until you remove it or define the purpose again.`
							: `The ${undefinedCodes.map((c) => `"${c}"`).join(', ')} purposes no longer exist. This pipeline does not process any event until you remove them or define the purposes again.`}
					</div>
				)}
			</div>
		</Section>
	);
});

export default PipelineConsents;
