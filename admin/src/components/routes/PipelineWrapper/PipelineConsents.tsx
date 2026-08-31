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
	const { pipeline, pipelineType, setPipeline } = useContext(PipelineContext);

	const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
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
			} catch (err) {
				handleError(err);
			}
		};
		fetchPurposes();
	}, []);

	const setEnabled = (enabled: boolean) => {
		const p = structuredClone(pipeline);
		setIsEnabled(enabled);
		if (enabled) {
			p.requiredConsents = { operator: 'and', purposes: [] };
		} else {
			p.requiredConsents = null;
		}
		setPipeline(p);
	};

	const onToggle = (e: any) => {
		setEnabled(e.target.checked);
	};

	const onSentenceClick = (e: React.MouseEvent) => {
		if (purposes.length === 0) {
			return;
		}
		if ((e.target as HTMLElement).closest('sl-select')) {
			return;
		}
		setEnabled(!isEnabled);
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

	const selectedPurposeIDs = pipeline.requiredConsents?.purposes ?? [];

	const isEventTarget = pipelineType.target === 'Event';
	const subject = isEventTarget ? 'An event' : 'A profile';
	const subjects = isEventTarget ? 'events' : 'profiles';

	return (
		<Section
			className='pipeline__consents'
			title='Privacy'
			description={`Choose whether this pipeline should require consent for specific purposes before processing ${subjects}.`}
			padded={true}
			ref={ref}
			annotated={true}
		>
			<div className='pipeline__consents-toggle'>
				<SlCheckbox checked={isEnabled} onSlChange={onToggle} disabled={purposes.length === 0} />
				<div
					className={`pipeline__consents-logical-sentence${
						purposes.length === 0 ? ' pipeline__consents-logical-sentence--disabled' : ''
					}`}
					onClick={onSentenceClick}
				>
					{`${subject} must have consent for`}
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
					value={selectedPurposeIDs}
					onSlChange={onChangePurposes}
					disabled={!isEnabled || purposes.length === 0}
				>
					{purposes.map((p) => (
						<SlOption key={p.id} value={p.id}>
							{p.name}
						</SlOption>
					))}
				</SlSelect>
			</div>
		</Section>
	);
});

export default PipelineConsents;
