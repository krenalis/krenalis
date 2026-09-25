import React, { useContext, useEffect, useRef, useState, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ConsentPurposesOperator } from '../../../lib/api/types/pipeline';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';

const PipelineConsents = forwardRef<any>((_, ref) => {
	const { pipeline, pipelineType, setPipeline, connection, isImport } = useContext(PipelineContext);

	const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
	const [isEnabled, setIsEnabled] = useState((pipeline.requiredConsents?.purposes.length ?? 0) > 0);

	const operatorRef = useRef<any>(null);

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

	useEffect(() => {
		const select = operatorRef.current;
		if (select == null || isEnabled) {
			return;
		}

		const onMouseDown = (e: MouseEvent) => {
			e.preventDefault();
			e.stopPropagation();
			select.focus();
		};

		const onKeyDown = (e: KeyboardEvent) => {
			if (e.key === 'Tab' || e.key === 'Escape' || e.altKey || e.ctrlKey || e.metaKey) {
				return;
			}
			e.preventDefault();
			e.stopPropagation();
		};

		// When the consents are disabled, keep the operator readable as part of
		// the sentence, but prevent it from being changed.
		select.addEventListener('mousedown', onMouseDown, true);
		select.addEventListener('keydown', onKeyDown, true);

		return () => {
			// restore select interactions.
			select.removeEventListener('mousedown', onMouseDown, true);
			select.removeEventListener('keydown', onKeyDown, true);
		};
	}, [isEnabled]);

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
	const usersTerm = connection.connector.terms.users.toLowerCase();
	const subjects = isEventTarget ? 'events' : isImport ? usersTerm : 'profiles';
	const actionVerb = isImport ? 'import' : isEventTarget ? 'send' : 'export';
	const actionParticiple = isImport ? 'imported' : isEventTarget ? 'sent' : 'exported';

	return (
		<Section
			className='pipeline__consents'
			title='Consent requirements'
			description={`Define which consent purposes ${subjects} must have before they can be ${actionParticiple}.`}
			padded={true}
			ref={ref}
			annotated={true}
		>
			<div className='pipeline__consents-toggle'>
				<SlSwitch checked={isEnabled} onSlChange={onToggle} disabled={purposes.length === 0} />
				<div
					className={`pipeline__consents-logical-sentence${
						purposes.length === 0 ? ' pipeline__consents-logical-sentence--disabled' : ''
					}`}
					onClick={onSentenceClick}
				>
					{`Only ${actionVerb} ${subjects} ${subjects === 'events' ? 'that have consent for' : 'who have consented to'}`}
					<SlSelect
						ref={operatorRef}
						className={`pipeline__consents-logical-select${
							isEnabled ? '' : ' pipeline__consents-logical-select--readonly'
						}`}
						size='small'
						value={pipeline.requiredConsents?.operator || 'and'}
						onSlChange={onChangeOperator}
					>
						<SlOption value='and'>all</SlOption>
						<SlOption value='or'>any</SlOption>
					</SlSelect>
					of the selected purposes.
				</div>
			</div>
			<div className='pipeline__consents-details'>
				<SlSelect
					className='pipeline__consents-select'
					multiple
					clearable
					placeholder={purposes.length === 0 ? 'No purposes defined yet' : 'Select consent purposes'}
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
