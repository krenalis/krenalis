import React, { useContext, useEffect, useId, useRef, useState } from 'react';
import './SchemaPropertyConsent.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlPopup from '@shoelace-style/shoelace/dist/react/popup/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { getConsentPurposeEventPath, getConsentPurposeProfilePath } from './SchemaPropertyConsent.helpers';

const HOVER_DELAY = 300;

interface SchemaPropertyConsentProps {
	isJSON: boolean;
	purposes: ConsentPurpose[] | undefined;
}

const SchemaPropertyConsent = ({ isJSON, purposes }: SchemaPropertyConsentProps) => {
	const [isHovered, setIsHovered] = useState(false);
	const [isFocused, setIsFocused] = useState(false);
	const [selectedPurposeID, setSelectedPurposeID] = useState<string>();
	const hoverTimeoutRef = useRef<number>();
	const popupID = useId();
	const { redirect } = useContext(AppContext);

	useEffect(() => () => window.clearTimeout(hoverTimeoutRef.current), []);

	if (purposes == null || purposes.length === 0) {
		return null;
	}

	const isOpen = isHovered || isFocused;
	const popupTitle = isJSON ? 'Consent purposes' : 'Consent purpose';
	const selectedPurposeIndex = isJSON
		? Math.max(
				purposes.findIndex((purpose) => purpose.id === selectedPurposeID),
				0,
			)
		: 0;
	const selectedPurpose = purposes[selectedPurposeIndex];
	const displayedPurposes = isJSON ? purposes.slice(selectedPurposeIndex, selectedPurposeIndex + 1) : purposes;
	const clearHoverTimeout = () => {
		window.clearTimeout(hoverTimeoutRef.current);
		hoverTimeoutRef.current = undefined;
	};
	const handlePointerEnter = (event: React.PointerEvent<HTMLElement>) => {
		const focusedElement = document.activeElement;
		if (focusedElement instanceof HTMLElement) {
			const focusedPopup = focusedElement.closest('.schema-property-consent');
			if (focusedPopup != null && focusedPopup !== event.currentTarget) {
				focusedElement.blur();
			}
		}
		if (!isHovered) {
			clearHoverTimeout();
			hoverTimeoutRef.current = window.setTimeout(() => setIsHovered(true), HOVER_DELAY);
		}
	};
	const handlePointerLeave = () => {
		clearHoverTimeout();
		setIsHovered(false);
	};
	const openPurpose = () => {
		clearHoverTimeout();
		setIsHovered(false);
		setIsFocused(false);
		redirect(`settings/privacy?purpose=${encodeURIComponent(selectedPurpose.id)}`);
	};

	return (
		<SlPopup
			className='schema-property-consent'
			active={isOpen}
			placement='top'
			strategy='fixed'
			distance={8}
			onClick={(event) => event.stopPropagation()}
			arrow
			flip
			shift
			autoSize='vertical'
			autoSizePadding={12}
			hoverBridge
			onPointerEnter={handlePointerEnter}
			onPointerLeave={handlePointerLeave}
			onFocus={() => {
				clearHoverTimeout();
				setIsFocused(true);
			}}
			onBlur={() => setIsFocused(false)}
			onKeyDown={(event) => {
				event.stopPropagation();
				if (event.key === 'Escape' && event.target instanceof HTMLElement) {
					clearHoverTimeout();
					setIsHovered(false);
					event.target.blur();
				}
			}}
		>
			<button
				className='schema-property-consent__trigger'
				type='button'
				slot='anchor'
				aria-controls={popupID}
				aria-expanded={isOpen}
			>
				<SlIcon name='shield-check' aria-hidden='true' />
				<span className='schema-property-consent__trigger-label'>Consent</span>
			</button>
			<div
				className={`schema-property-consent__popup${isJSON ? '' : ' schema-property-consent__popup--boolean'}`}
				id={popupID}
				role='region'
				aria-label={popupTitle}
			>
				<SlIcon className='schema-property-consent__popup-icon' name='shield-check' aria-hidden='true' />
				<div className='schema-property-consent__open-purpose-action'>
					<SlTooltip
						className='schema-property-consent__open-purpose-tooltip'
						content='Open consent purpose'
						hoist
					>
						<SlButton
							className='schema-property-consent__open-purpose'
							variant='text'
							size='small'
							circle
							onClick={openPurpose}
						>
							<SlIcon name='box-arrow-in-up-right' label='Open consent purpose' />
						</SlButton>
					</SlTooltip>
				</div>
				<div className='schema-property-consent__popup-content' aria-live={isJSON ? 'polite' : undefined}>
					<div className='schema-property-consent__popup-title'>{popupTitle}</div>
					{displayedPurposes.map((purpose) => (
						<div className='schema-property-consent__purpose' key={purpose.id}>
							<div className='schema-property-consent__purpose-name'>{purpose.name}</div>
							<div className='schema-property-consent__purpose-field'>
								<span>Code:</span>
								<code>{purpose.code}</code>
							</div>
							<div className='schema-property-consent__purpose-field'>
								<span>Event path:</span>
								<code>{getConsentPurposeEventPath(purpose)}</code>
							</div>
							<div className='schema-property-consent__purpose-field'>
								<span>Profile path:</span>
								<code>{getConsentPurposeProfilePath(purpose)}</code>
							</div>
						</div>
					))}
				</div>
				{isJSON && (
					<div className='schema-property-consent__pagination'>
						<span className='schema-property-consent__pagination-status'>
							{selectedPurposeIndex + 1} of {purposes.length}
						</span>
						<div className='schema-property-consent__pagination-actions'>
							<SlButton
								className='schema-property-consent__pagination-button'
								variant='text'
								size='small'
								circle
								title='Previous consent purpose'
								disabled={selectedPurposeIndex === 0}
								onClick={() => setSelectedPurposeID(purposes[selectedPurposeIndex - 1].id)}
							>
								<SlIcon name='chevron-left' aria-hidden='true' />
							</SlButton>
							<SlButton
								className='schema-property-consent__pagination-button'
								variant='text'
								size='small'
								circle
								title='Next consent purpose'
								disabled={selectedPurposeIndex === purposes.length - 1}
								onClick={() => setSelectedPurposeID(purposes[selectedPurposeIndex + 1].id)}
							>
								<SlIcon name='chevron-right' aria-hidden='true' />
							</SlButton>
						</div>
					</div>
				)}
			</div>
		</SlPopup>
	);
};

export { SchemaPropertyConsent };
