import React from 'react';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import { formatNumber } from '../../../utils/formatNumber';

const profilePageSizes = [25, 50, 100];

interface ProfilesPaginationProps {
	children?: React.ReactNode;
	first: number;
	hasNext: boolean;
	isLoading: boolean;
	limit: number;
	onFirstChange: (first: number) => void;
	onLimitChange: (limit: number) => void;
	profileCount: number;
	total: number;
}

const ProfilesPagination = ({
	children,
	first,
	hasNext,
	isLoading,
	limit,
	onFirstChange,
	onLimitChange,
	profileCount,
	total,
}: ProfilesPaginationProps) => {
	const firstProfile = first + 1;
	const lastProfile = first + profileCount;
	const hasPreviousPage = first > 0;
	const previousPageDisabled = isLoading || !hasPreviousPage;
	const nextPageDisabled = isLoading || !hasNext;

	return (
		<nav className='profiles-list__pagination' aria-label='Profiles pagination'>
			<div className='profiles-list__pagination-results'>
				<span className='profiles-list__pagination-range'>
					{profileCount === 0
						? 'No profiles on this page'
						: `${formatNumber(firstProfile)}–${formatNumber(lastProfile)} of ${formatNumber(total)}`}
				</span>
				{children}
			</div>
			<div className='profiles-list__pagination-controls'>
				<SlSelect
					className='profiles-list__pagination-page-size'
					label='Profiles per page'
					size='small'
					value={String(limit)}
					disabled={isLoading}
					onSlChange={(event: any) => onLimitChange(Number(event.target.value))}
				>
					{profilePageSizes.map((pageSize) => (
						<SlOption key={pageSize} value={String(pageSize)}>
							{pageSize}
						</SlOption>
					))}
				</SlSelect>
				<SlButtonGroup className='profiles-list__pagination-navigation' label='Profile pages'>
					<SlButton
						className='profiles-list__pagination-previous'
						size='small'
						aria-label='Previous page'
						disabled={previousPageDisabled}
						onClick={() => {
							if (!previousPageDisabled) {
								onFirstChange(Math.max(0, first - limit));
							}
						}}
					>
						<SlIcon name='chevron-left' aria-hidden='true' />
						<span className='profiles-list__pagination-action-label'>Previous page</span>
					</SlButton>
					<SlButton
						className='profiles-list__pagination-next'
						size='small'
						aria-label='Next page'
						disabled={nextPageDisabled}
						onClick={() => {
							if (!nextPageDisabled) {
								onFirstChange(first + limit);
							}
						}}
					>
						<SlIcon name='chevron-right' aria-hidden='true' />
						<span className='profiles-list__pagination-action-label'>Next page</span>
					</SlButton>
				</SlButtonGroup>
			</div>
		</nav>
	);
};

export { ProfilesPagination };
