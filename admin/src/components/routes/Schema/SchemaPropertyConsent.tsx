import React, { useContext, useEffect, useId, useRef, useState } from 'react';
import './SchemaPropertyConsent.css';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaPopup from '@awesome.me/webawesome/dist/react/popup/index.js';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import AppContext from '../../../context/AppContext';
import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { formatProfileConsentLocation } from '../../../lib/core/consentPurpose';

const HOVER_DELAY = 300;

interface SchemaPropertyConsentProps {
	purposes: ConsentPurpose[] | undefined;
}

const SchemaPropertyConsent = ({ purposes }: SchemaPropertyConsentProps) => {
	const [isHovered, setIsHovered] = useState(false);
	const [isFocused, setIsFocused] = useState(false);
	const [selectedPurposeID, setSelectedPurposeID] = useState<string>();

	const hoverTimeoutRef = useRef<number>();
	const triggerRef = useRef<HTMLButtonElement>(null);
	const popupID = useId();
	const openPurposeID = useId();

	const { redirect } = useContext(AppContext);

	useEffect(() => () => window.clearTimeout(hoverTimeoutRef.current), []);

	if (purposes == null || purposes.length === 0) {
		return null;
	}

	const isOpen = isHovered || isFocused;
	const hasMultiplePurposes = purposes.length > 1;
	const popupTitle = hasMultiplePurposes ? 'Consent purposes' : 'Consent purpose';
	const selectedPurposeIndex = Math.max(
		purposes.findIndex((purpose) => purpose.id === selectedPurposeID),
		0,
	);
	const selectedPurpose = purposes[selectedPurposeIndex];
	const displayedPurposes = purposes.slice(selectedPurposeIndex, selectedPurposeIndex + 1);

	const selectPurpose = (index: number) => {
		if (index === 0 || index === purposes.length - 1) {
			triggerRef.current?.focus();
		}
		setSelectedPurposeID(purposes[index].id);
	};

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
		redirect(`settings/consent-management?purpose=${encodeURIComponent(selectedPurpose.id)}`);
	};

	return (
		<WaPopup
			className='schema-property-consent'
			active={isOpen}
			placement='top'
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
				ref={triggerRef}
				className='schema-property-consent__trigger'
				type='button'
				slot='anchor'
				aria-controls={popupID}
				aria-expanded={isOpen}
			>
				<WaIcon name='shield-check' aria-hidden='true' />
				<span className='schema-property-consent__trigger-label'>Consent</span>
			</button>
			<div
				className={`schema-property-consent__popup${hasMultiplePurposes ? '' : ' schema-property-consent__popup--without-pagination'}`}
				id={popupID}
				role='region'
				aria-label={popupTitle}
			>
				<WaIcon className='schema-property-consent__popup-icon' name='shield-check' aria-hidden='true' />
				<div className='schema-property-consent__open-purpose-action'>
					<WaTooltip
						className='schema-property-consent__open-purpose-tooltip'
						for={openPurposeID}
						trigger='hover'
					>
						Open consent purpose
					</WaTooltip>
					<WaButton
						id={openPurposeID}
						className='schema-property-consent__open-purpose'
						appearance='plain'
						size='s'
						pill
						onClick={openPurpose}
					>
						<WaIcon name='box-arrow-in-up-right' label='Open consent purpose' />
					</WaButton>
				</div>
				<div
					className='schema-property-consent__popup-content'
					aria-live={hasMultiplePurposes ? 'polite' : undefined}
				>
					<div className='schema-property-consent__popup-title'>{popupTitle}</div>
					{displayedPurposes.map((purpose) => (
						<div className='schema-property-consent__purpose' key={purpose.id}>
							<div className='schema-property-consent__purpose-name'>{purpose.name}</div>
							<div
								className={`schema-property-consent__purpose-field${purpose.eventConsentLocations.length === 0 ? ' schema-property-consent__purpose-field--empty' : ''}`}
								aria-hidden={purpose.eventConsentLocations.length === 0}
							>
								<span>Consent in events:</span>
								<div className='schema-property-consent__event-paths'>
									{purpose.eventConsentLocations.length === 0 ? (
										<code>{'\u00a0'}</code>
									) : (
										purpose.eventConsentLocations.map((location, index) => (
											<code key={index} title={location.purposeCode}>
												{location.purposeCode}
											</code>
										))
									)}
								</div>
							</div>
							<div className='schema-property-consent__purpose-field'>
								<span>Consent in profiles:</span>
								{purpose.profileConsentLocation == null ? (
									'Unknown'
								) : (
									<code>{formatProfileConsentLocation(purpose.profileConsentLocation)}</code>
								)}
							</div>
						</div>
					))}
				</div>
				{hasMultiplePurposes && (
					<div className='schema-property-consent__pagination'>
						<span className='schema-property-consent__pagination-status'>
							{selectedPurposeIndex + 1} of {purposes.length}
						</span>
						<div className='schema-property-consent__pagination-actions'>
							<WaButton
								className='schema-property-consent__pagination-button'
								appearance='plain'
								size='s'
								pill
								title='Previous consent purpose'
								disabled={selectedPurposeIndex === 0}
								onClick={() => selectPurpose(selectedPurposeIndex - 1)}
							>
								<WaIcon name='chevron-left' aria-hidden='true' />
							</WaButton>
							<WaButton
								className='schema-property-consent__pagination-button'
								appearance='plain'
								size='s'
								pill
								title='Next consent purpose'
								disabled={selectedPurposeIndex === purposes.length - 1}
								onClick={() => selectPurpose(selectedPurposeIndex + 1)}
							>
								<WaIcon name='chevron-right' aria-hidden='true' />
							</WaButton>
						</div>
					</div>
				)}
			</div>
		</WaPopup>
	);
};

export { SchemaPropertyConsent };
