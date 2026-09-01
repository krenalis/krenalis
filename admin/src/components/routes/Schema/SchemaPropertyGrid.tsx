import React, { ReactNode, useEffect, useId, useRef, useState } from 'react';
import './SchemaPropertyGrid.css';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { Property } from '../../../lib/api/types/types';
import { GridNestedRowsIndentation } from '../../base/Grid/Grid.types';

const schemaPropertyGridNestedRowsIndentation: GridNestedRowsIndentation = { base: 34, step: 20 };

const schemaPropertyGridClassNamePrefixes = {
	edit: 'schema-edit',
	readOnly: 'schema-grid',
} as const;

type SchemaPropertyGridView = keyof typeof schemaPropertyGridClassNamePrefixes;
type SchemaPropertyGridClassNamePrefix = (typeof schemaPropertyGridClassNamePrefixes)[SchemaPropertyGridView];

const SchemaPropertyName = ({ property }: { property: Property }) => (
	<div className='schema-property-grid__property-name'>
		<div>{property.name}</div>
		{property.displayName && (
			<div className='schema-property-grid__property-display-name'>{property.displayName}</div>
		)}
	</div>
);

const SchemaPropertyIdentifierBadge = ({ position }: { position: number }) => (
	<SlBadge className='schema-property-grid__identifier' pill variant='neutral'>
		#{position}
	</SlBadge>
);

const SchemaPropertyIdentifierValue = ({ position }: { position?: number }) =>
	position == null ? (
		<>Not an identifier</>
	) : (
		<span className='schema-property-grid__identifier-value'>
			Identifier <SchemaPropertyIdentifierBadge position={position} />
		</span>
	);

// Prevent a surrounding Shoelace label from focusing its associated control.
const preventInfoInteraction = (event: React.SyntheticEvent) => {
	event.preventDefault();
	event.stopPropagation();
};

const SchemaPropertyInfoTooltip = ({ content, label }: { content: string; label: string }) => {
	const descriptionID = useId();

	return (
		<SlTooltip className='schema-property-grid__tooltip' placement='top' trigger='hover' hoist={true}>
			<span id={descriptionID} className='schema-property-grid__tooltip-content' slot='content'>
				{content}
			</span>
			<span
				className='schema-property-grid__info'
				role='img'
				aria-label={label}
				aria-describedby={descriptionID}
				onPointerDownCapture={preventInfoInteraction}
				onClickCapture={preventInfoInteraction}
			>
				<SlIcon name='info-circle' aria-hidden='true' />
			</span>
		</SlTooltip>
	);
};

const SchemaPropertyIdentifierLabel = () => (
	<span className='schema-property-grid__label-content'>
		Identifier
		<SchemaPropertyInfoTooltip
			content={
				'Krenalis checks identifiers in order, starting with #1, to determine whether identities belong to the same profile.\n\n' +
				"To change which properties are identifiers or the order in which they're checked, go to Profile Unification → Rules."
			}
			label='About identifiers'
		/>
	</span>
);

const SchemaPropertyPrimarySourceLabel = ({ primarySourceName }: { primarySourceName?: string | null }) => {
	let content =
		'If this source has a value for this property, its most recent value is used. Otherwise, the most recent value from any other source is used.';
	if (primarySourceName === null) {
		content = 'This property has no primary source, so the most recent value from any source is used.';
	} else if (primarySourceName !== undefined) {
		content = `The ${primarySourceName} connection is the primary source for this property, so its most recent value is used when available. Otherwise, the most recent value from any other source is used.`;
	}

	return (
		<span className='schema-property-grid__label-content'>
			Primary source
			<SchemaPropertyInfoTooltip content={content} label='About primary source' />
		</span>
	);
};

interface SchemaPropertyGridSummaryProps {
	children?: ReactNode;
	objectCount: number;
	propertyCount: number;
	view: SchemaPropertyGridView;
}

const SchemaPropertyGridSummary = ({ children, objectCount, propertyCount, view }: SchemaPropertyGridSummaryProps) => (
	<div className={`${schemaPropertyGridClassNamePrefixes[view]}__summary`}>
		<span>
			<SlIcon name='table' />
			<span>
				{propertyCount} {propertyCount === 1 ? 'property' : 'properties'} ({objectCount}{' '}
				{objectCount === 1 ? 'object' : 'objects'})
			</span>
		</span>
		{children}
	</div>
);

interface SchemaPropertyGridExpansionButtonsProps {
	classNamePrefix: SchemaPropertyGridClassNamePrefix;
	disabled: boolean;
	onCollapse: () => void;
	onExpand: () => void;
}

const SchemaPropertyGridExpansionButtons = ({
	classNamePrefix,
	disabled,
	onCollapse,
	onExpand,
}: SchemaPropertyGridExpansionButtonsProps) => {
	const editButtonClassName = classNamePrefix === 'schema-edit' ? ' schema-edit__expansion-button' : '';
	const buttonClassName = `${classNamePrefix}__toolbar-icon-button${editButtonClassName}`;

	return (
		<div className={`${classNamePrefix}__expansion-buttons`}>
			<SlTooltip className={`${classNamePrefix}__toolbar-tooltip`} content='Expand all properties' hoist>
				<SlButton
					className={`${classNamePrefix}__expand-all-button ${buttonClassName}`}
					size='small'
					aria-label='Expand all properties'
					disabled={disabled}
					onClick={onExpand}
				>
					<SlIcon name='chevron-expand' />
				</SlButton>
			</SlTooltip>
			<SlTooltip className={`${classNamePrefix}__toolbar-tooltip`} content='Collapse all properties' hoist>
				<SlButton
					className={`${classNamePrefix}__collapse-all-button ${buttonClassName}`}
					size='small'
					aria-label='Collapse all properties'
					disabled={disabled}
					onClick={onCollapse}
				>
					<SlIcon name='chevron-contract' />
				</SlButton>
			</SlTooltip>
		</div>
	);
};

interface SchemaPropertySearchProps {
	classNamePrefix: SchemaPropertyGridClassNamePrefix;
	isOpen: boolean;
	onChange: (value: string) => void;
	onOpenChange: (isOpen: boolean) => void;
	searchRef: React.RefObject<any>;
	value: string;
}

const SchemaPropertySearch = ({
	classNamePrefix,
	isOpen,
	onChange,
	onOpenChange,
	searchRef,
	value,
}: SchemaPropertySearchProps) => {
	const searchControlRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		if (!isOpen || value !== '') {
			return;
		}

		const closeWhenOutside = (event: Event) => {
			if (searchControlRef.current != null && !event.composedPath().includes(searchControlRef.current)) {
				onOpenChange(false);
			}
		};
		document.addEventListener('pointerdown', closeWhenOutside);

		return () => {
			document.removeEventListener('pointerdown', closeWhenOutside);
		};
	}, [isOpen, onOpenChange, value]);

	const onClick = () => {
		onOpenChange(true);
	};

	return (
		<div
			ref={searchControlRef}
			className={`${classNamePrefix}__search-control${isOpen ? ` ${classNamePrefix}__search-control--open` : ''}`}
		>
			{isOpen ? (
				<SlInput
					ref={searchRef}
					className={`${classNamePrefix}__search`}
					size='small'
					placeholder='Search a property...'
					clearable
					value={value}
					onSlInput={(event: any) => onChange(event.target.value)}
				>
					<SlIcon name='search' slot='prefix' />
					<SlIcon name='backspace' slot='clear-icon' />
				</SlInput>
			) : (
				<SlTooltip className={`${classNamePrefix}__toolbar-tooltip`} content='Search properties' hoist>
					<SlButton
						className={`${classNamePrefix}__search-button ${classNamePrefix}__toolbar-icon-button`}
						size='small'
						aria-label='Search properties'
						onClick={onClick}
					>
						<SlIcon name='search' />
					</SlButton>
				</SlTooltip>
			)}
		</div>
	);
};

interface SchemaPropertyGridToolbarProps {
	children?: ReactNode;
	expansionDisabled: boolean;
	onCollapse: () => void;
	onExpand: () => void;
	onSearchChange: (value: string) => void;
	search: string;
	view: SchemaPropertyGridView;
}

const SchemaPropertyGridToolbar = ({
	children,
	expansionDisabled,
	onCollapse,
	onExpand,
	onSearchChange,
	search,
	view,
}: SchemaPropertyGridToolbarProps) => {
	const [isSearchOpen, setIsSearchOpen] = useState(false);
	const searchRef = useRef<any>();
	const classNamePrefix = schemaPropertyGridClassNamePrefixes[view];

	// Filtering rerenders the grid and can move focus away from the search input.
	// Wait for Shoelace to finish updating before restoring it.
	useEffect(() => {
		if (!isSearchOpen || searchRef.current == null) {
			return;
		}

		let animationFrame: number | undefined;
		let canceled = false;
		const input = searchRef.current;
		input.updateComplete.then(() => {
			if (canceled) {
				return;
			}
			animationFrame = requestAnimationFrame(() => {
				if (searchRef.current === input && !input.matches(':focus-within')) {
					input.focus({ preventScroll: true });
				}
			});
		});

		return () => {
			canceled = true;
			if (animationFrame != null) {
				cancelAnimationFrame(animationFrame);
			}
		};
	}, [isSearchOpen, search]);

	const searchControl = (
		<SchemaPropertySearch
			classNamePrefix={classNamePrefix}
			isOpen={isSearchOpen}
			onChange={onSearchChange}
			onOpenChange={setIsSearchOpen}
			searchRef={searchRef}
			value={search}
		/>
	);

	return (
		<div className={`${classNamePrefix}__toolbar`}>
			<SchemaPropertyGridExpansionButtons
				classNamePrefix={classNamePrefix}
				disabled={expansionDisabled}
				onCollapse={onCollapse}
				onExpand={onExpand}
			/>
			{children == null ? (
				searchControl
			) : (
				<div className={`${classNamePrefix}__toolbar-controls`}>
					{searchControl}
					{children}
				</div>
			)}
		</div>
	);
};

export {
	SchemaPropertyGridSummary,
	SchemaPropertyGridToolbar,
	SchemaPropertyName,
	SchemaPropertyIdentifierBadge,
	SchemaPropertyIdentifierLabel,
	SchemaPropertyIdentifierValue,
	SchemaPropertyInfoTooltip,
	SchemaPropertyPrimarySourceLabel,
	schemaPropertyGridNestedRowsIndentation,
};
