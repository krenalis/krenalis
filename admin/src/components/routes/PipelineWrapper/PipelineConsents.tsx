import React, { useContext, useEffect, useRef, useState, forwardRef } from 'react';
import Section from '../../base/Section/Section';
import PipelineContext from '../../../context/PipelineContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ConsentPurposesOperator } from '../../../lib/api/types/pipeline';
import { getMissingConsentLocation, isOutputPathTransformed } from '../../../lib/core/pipeline';
import { UI_BASE_PATH } from '../../../constants/paths';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import type SlOptionElement from '@shoelace-style/shoelace/dist/components/option/option.js';

interface PurposeError {
	purpose: ConsentPurpose;
	message: string;
	isDeleted: boolean;
}

const PipelineConsents = forwardRef<any>((_, ref) => {
	const {
		pipeline,
		pipelineType,
		setPipeline,
		connection,
		isImport,
		consentPurposes: purposes,
		deletedConsentPurposes,
		consentPropertyPaths,
		selectedOutPaths,
	} = useContext(PipelineContext);

	const [isEnabled, setIsEnabled] = useState((pipeline.requiredConsents?.purposes.length ?? 0) > 0);

	const operatorRef = useRef<any>(null);
	const purposeTagsRef = useRef(new WeakMap<SlOptionElement, HTMLElement>());

	useEffect(() => {
		setIsEnabled((pipeline.requiredConsents?.purposes.length ?? 0) > 0);
	}, [pipeline.id]);

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
		if (!hasPurposes) {
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
	const hasPurposes = purposes.length > 0 || deletedConsentPurposes.length > 0;

	// Collect the selected purposes that have been deleted, or that no longer
	// have the consent location read by the pipeline.
	const purposeErrors: PurposeError[] = [];
	for (const id of selectedPurposeIDs) {
		const purpose = purposes.find((p) => p.id === id);
		if (purpose == null) {
			const deleted = deletedConsentPurposes.find((p) => p.id === id);
			if (deleted != null) {
				purposeErrors.push({
					purpose: deleted,
					message: `Consent purpose "${deleted.name}" no longer exists. Remove it from the selected purposes.`,
					isDeleted: true,
				});
			}
			continue;
		}
		const location = getMissingConsentLocation(purpose, pipelineType, connection);
		if (location != null) {
			purposeErrors.push({
				purpose,
				message: `Consent purpose "${purpose.name}" has no ${location} consent location.`,
				isDeleted: false,
			});
		}
	}

	// Show the tags of these purposes in red. The other tags are the same as
	// the default ones of the select.
	const invalidPurposeIDs = new Set(purposeErrors.map(({ purpose }) => purpose.id));
	const getPurposeTag = (option: SlOptionElement): HTMLElement => {
		// The select renders its tags again when it loses the focus, which
		// happens on the mousedown on a remove button. Reuse the tag of the
		// option and change it only when needed, as a tag replaced or changed
		// at that moment does not receive the click.
		let tag = purposeTagsRef.current.get(option);
		if (tag == null) {
			tag = document.createElement('sl-tag');
			tag.setAttribute('part', 'tag');
			tag.setAttribute('size', 'medium');
			tag.setAttribute('removable', '');
			purposeTagsRef.current.set(option, tag);
		}
		const isInvalid = invalidPurposeIDs.has(option.value);
		const exportParts = `base:${isInvalid ? 'invalid-tag__base' : 'tag__base'}, content:tag__content, remove-button:tag__remove-button, remove-button__base:tag__remove-button__base`;
		if (tag.getAttribute('exportparts') !== exportParts) {
			tag.setAttribute('exportparts', exportParts);
		}
		const label = option.getTextLabel();
		if (tag.textContent !== label) {
			tag.textContent = label;
		}
		return tag;
	};

	// Collect the profile properties holding a required consent that the
	// transformation does not return.
	const untransformedConsentPaths = [...consentPropertyPaths].filter(
		([path]) => !isOutputPathTransformed(path, pipeline, pipelineType, selectedOutPaths),
	);
	const hasTransformationFunction = pipeline.transformation?.function != null;

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
				<SlSwitch checked={isEnabled} onSlChange={onToggle} disabled={!hasPurposes} />
				<div
					className={`pipeline__consents-logical-sentence${
						hasPurposes ? '' : ' pipeline__consents-logical-sentence--disabled'
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
					getTag={getPurposeTag}
					onSlChange={onChangePurposes}
					disabled={!isEnabled || !hasPurposes}
				>
					{purposes.map((p) => {
						// A purpose without the consent location read by the pipeline
						// cannot be selected.
						const location = getMissingConsentLocation(p, pipelineType, connection);
						return (
							<SlOption key={p.id} value={p.id} disabled={location != null}>
								{p.name}
								{location != null && (
									<span slot='suffix' className='pipeline__consents-option-message'>
										{`has no ${location} consent location`}
									</span>
								)}
							</SlOption>
						);
					})}
					{deletedConsentPurposes.map((p) => (
						<SlOption key={p.id} value={p.id} disabled>
							{p.name}
							<span slot='suffix' className='pipeline__consents-option-message'>
								no longer exists
							</span>
						</SlOption>
					))}
				</SlSelect>
				{(purposeErrors.length > 0 || untransformedConsentPaths.length > 0) && (
					<div className='pipeline__consents-errors'>
						{purposeErrors.map(({ purpose, message, isDeleted }) => (
							<div key={purpose.id} className='pipeline__consents-error'>
								<SlIcon name='exclamation-circle' />
								<span>
									{message}
									{!isDeleted && (
										<>
											{' '}
											<a
												href={`${UI_BASE_PATH}settings/consent-management?purpose=${encodeURIComponent(purpose.id)}`}
												target='_blank'
												rel='noopener'
											>
												Configure the purpose
											</a>
										</>
									)}
								</span>
							</div>
						))}
						{untransformedConsentPaths.map(([path, pathPurposes]) => (
							<div key={path} className='pipeline__consents-error'>
								<SlIcon name='exclamation-circle' />
								<span>
									{`Consent for ${pathPurposes.map((p) => `"${p.name}"`).join(', ')} is read from the profile property "${path}": ${
										hasTransformationFunction
											? 'select it as an output property of the transformation function in Full mode'
											: 'map it in the transformation'
									}`}
								</span>
							</div>
						))}
					</div>
				)}
			</div>
		</Section>
	);
});

export default PipelineConsents;
