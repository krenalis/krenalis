import React, { ReactNode, useEffect, useRef, useState } from 'react';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { GridNestedRowsIndentation } from '../../base/Grid/Grid.types';

const schemaPropertyGridNestedRowsIndentation: GridNestedRowsIndentation = { base: 34, step: 20 };

const schemaPropertyGridClassNamePrefixes = {
	edit: 'schema-edit',
	readOnly: 'schema-grid',
} as const;

type SchemaPropertyGridView = keyof typeof schemaPropertyGridClassNamePrefixes;
type SchemaPropertyGridClassNamePrefix = (typeof schemaPropertyGridClassNamePrefixes)[SchemaPropertyGridView];

interface SchemaPropertyGridSummaryProps {
	children?: ReactNode;
	propertyCount: number;
	view: SchemaPropertyGridView;
}

const SchemaPropertyGridSummary = ({ children, propertyCount, view }: SchemaPropertyGridSummaryProps) => (
	<div className={`${schemaPropertyGridClassNamePrefixes[view]}__summary`}>
		<span>
			<SlIcon name='table' />
			{propertyCount} {propertyCount === 1 ? 'property' : 'properties'}
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

export { SchemaPropertyGridSummary, SchemaPropertyGridToolbar, schemaPropertyGridNestedRowsIndentation };
