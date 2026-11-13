import React from 'react';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaButtonGroup from '@awesome.me/webawesome/dist/react/button-group/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaOption from '@awesome.me/webawesome/dist/react/option/index.js';
import WaSelect from '@awesome.me/webawesome/dist/react/select/index.js';
import { formatNumber } from '../../../utils/formatNumber';

const PROFILE_PAGE_SIZES = [25, 50, 100];

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
				<WaSelect
					className='profiles-list__pagination-page-size'
					label='Profiles per page'
					size='s'
					value={String(limit)}
					disabled={isLoading}
					onChange={(event: any) => onLimitChange(Number(event.target.value))}
				>
					{PROFILE_PAGE_SIZES.map((pageSize) => (
						<WaOption key={pageSize} value={String(pageSize)}>
							{pageSize}
						</WaOption>
					))}
				</WaSelect>
				<WaButtonGroup className='profiles-list__pagination-navigation' label='Profile pages'>
					<WaButton
						appearance='outlined'
						className='profiles-list__pagination-previous'
						size='s'
						aria-label='Previous page'
						disabled={previousPageDisabled}
						onClick={() => {
							if (!previousPageDisabled) {
								onFirstChange(Math.max(0, first - limit));
							}
						}}
					>
						<WaIcon name='chevron-left' aria-hidden='true' />
						<span className='profiles-list__pagination-action-label'>Previous page</span>
					</WaButton>
					<WaButton
						appearance='outlined'
						className='profiles-list__pagination-next'
						size='s'
						aria-label='Next page'
						disabled={nextPageDisabled}
						onClick={() => {
							if (!nextPageDisabled) {
								onFirstChange(first + limit);
							}
						}}
					>
						<WaIcon name='chevron-right' aria-hidden='true' />
						<span className='profiles-list__pagination-action-label'>Next page</span>
					</WaButton>
				</WaButtonGroup>
			</div>
		</nav>
	);
};

export { ProfilesPagination };
