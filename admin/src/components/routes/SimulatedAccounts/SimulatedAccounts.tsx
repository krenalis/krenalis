import React, { FormEvent, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import AppContext from '../../../context/AppContext';
import { SimulatedAccount } from '../../../lib/api/types/simulatedAccount';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { Link } from '../../base/Link/Link';
import NotFound from '../NotFound/NotFound';
import { parseDuplicateRecordPercent, parseInitialUserRecords } from './simulatedAccountValues';
import './SimulatedAccounts.css';

const POLL_INTERVAL_MS = 3000;
const COUNT_PRESETS = [0, 10, 100, 1000, 10000, 100000, 1000000];
const DUPLICATE_PRESETS = [
	{ value: 0, label: '0% — None' },
	{ value: 1, label: '~1% — Minimal' },
	{ value: 2, label: '~2% — Low' },
	{ value: 5, label: '~5% — Moderate' },
	{ value: 10, label: '~10% — High' },
];

const errorText = (err: unknown): string =>
	err instanceof Error ? err.message : 'The request failed. Please try again.';

interface AccountProgressProps {
	account: SimulatedAccount;
}

const AccountProgress = ({ account }: AccountProgressProps) => {
	const percent =
		account.userCount > 0
			? Math.min(100, Math.floor((account.generatedRecordCount * 100) / account.userCount))
			: 100;
	return (
		<div className='simulated-accounts__progress' aria-live='polite'>
			<div>
				{account.generatedRecordCount.toLocaleString()} of {account.userCount.toLocaleString()} initial user
				records · {percent}%
			</div>
			<progress value={account.generatedRecordCount} max={account.userCount > 0 ? account.userCount : 1} />
			{account.generatedRecordCount >= account.userCount && <div>Completing preparation…</div>}
		</div>
	);
};

interface AccountStatusProps {
	account: SimulatedAccount;
}

const AccountStatus = ({ account }: AccountStatusProps) => (
	<div className='simulated-accounts__status'>
		<strong>{account.status}</strong>
		{account.status === 'Preparing' && <AccountProgress account={account} />}
		{account.status === 'Ready' && <span>Initial preparation complete</span>}
		{account.status === 'Failed' && (
			<span role='alert'>{account.generationError !== '' ? account.generationError : 'Preparation failed.'}</span>
		)}
	</div>
);

const SimulatedAccounts = () => {
	const { workspaces, selectedWorkspace, setTitle } = useContext(AppContext);
	const workspace = workspaces.find((item) => item.id === selectedWorkspace);

	useLayoutEffect(() => {
		setTitle('Simulated accounts');
	}, [setTitle]);

	if (workspace?.environment !== 'development') return <NotFound />;
	return <SimulatedAccountsPage key={selectedWorkspace} />;
};

const SimulatedAccountsPage = () => {
	const { id } = useParams();
	const isCreate = window.location.pathname.endsWith('/create');
	if (isCreate) return <CreateAccount />;
	if (id != null) return <AccountDetail key={id} id={id} />;
	return <AccountList />;
};

const AccountList = () => {
	const { api } = useContext(AppContext);
	const [accounts, setAccounts] = useState<SimulatedAccount[] | null>(null);
	const [error, setError] = useState('');
	const [refresh, setRefresh] = useState(0);

	useEffect(() => {
		const controller = new AbortController();
		let timer: ReturnType<typeof setTimeout>;
		let hasLoadedData = false;
		const load = async () => {
			try {
				const result = await api.workspaces.simulatedAccounts.find(controller.signal);
				if (controller.signal.aborted) return;
				setAccounts(result);
				hasLoadedData = true;
				setError('');
				if (result.some((account) => account.status === 'Preparing'))
					timer = setTimeout(load, POLL_INTERVAL_MS);
			} catch (err) {
				if (controller.signal.aborted) return;
				setError(errorText(err));
				if (hasLoadedData) timer = setTimeout(load, POLL_INTERVAL_MS);
			}
		};
		load();
		return () => {
			controller.abort();
			clearTimeout(timer);
		};
	}, [refresh]);

	return (
		<main className='route-content simulated-accounts'>
			<div className='simulated-accounts__header'>
				<div>
					<h1>Simulated accounts</h1>
					<p>Reusable test data for this Development workspace.</p>
				</div>
				<Link path='simulated-accounts/create'>
					<SlButton variant='primary'>
						<SlIcon name='plus' slot='prefix' />
						Create simulated account
					</SlButton>
				</Link>
			</div>
			{error !== '' && (
				<div className='simulated-accounts__error' role='alert'>
					{error}{' '}
					<SlButton size='small' onClick={() => setRefresh((value) => value + 1)}>
						Retry
					</SlButton>
				</div>
			)}
			{accounts == null && error === '' && (
				<div className='simulated-accounts__loading'>
					<SlSpinner /> Loading accounts…
				</div>
			)}
			{accounts != null && accounts.length === 0 && (
				<div className='simulated-accounts__empty'>
					<h2>No simulated accounts yet</h2>
					<p>Create an account with initial user records, or start empty.</p>
					<Link path='simulated-accounts/create'>
						<SlButton variant='primary'>Create simulated account</SlButton>
					</Link>
				</div>
			)}
			{accounts != null && accounts.length > 0 && (
				<div className='simulated-accounts__list'>
					{accounts.map((account) => (
						<Link
							key={account.id}
							path={`simulated-accounts/${account.id}`}
							className='simulated-accounts__row'
						>
							<div className='simulated-accounts__row-name'>
								<strong>{account.name}</strong>
							</div>
							<AccountStatus account={account} />
							<SlIcon name='chevron-right' />
						</Link>
					))}
				</div>
			)}
		</main>
	);
};

const CreateAccount = () => {
	const { api, redirect } = useContext(AppContext);
	const isActive = useRef(true);
	const countInput = useRef<HTMLInputElement>(null);
	const duplicateInput = useRef<HTMLInputElement>(null);
	const isSubmitPending = useRef(false);
	const [name, setName] = useState('');
	const [countMode, setCountMode] = useState('10000');
	const [customCount, setCustomCount] = useState('');
	const [duplicateMode, setDuplicateMode] = useState('2');
	const [customDuplicate, setCustomDuplicate] = useState('');
	const [error, setError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	useEffect(
		() => () => {
			isActive.current = false;
		},
		[],
	);
	useEffect(() => {
		if (countMode === 'custom') countInput.current?.focus();
	}, [countMode]);
	useEffect(() => {
		if (duplicateMode === 'custom') duplicateInput.current?.focus();
	}, [duplicateMode]);

	const rawCount = countMode === 'custom' ? customCount : countMode;
	const count = Number(rawCount.replaceAll(',', ''));
	const hasRecords = count > 0;

	const onSubmit = async (event: FormEvent) => {
		event.preventDefault();
		if (isSubmitPending.current) return;
		const trimmedName = name.trim();
		if (trimmedName.length === 0 || trimmedName.length > 100) {
			setError('Account name must be 1 to 100 characters.');
			return;
		}
		const rawDuplicate = duplicateMode === 'custom' ? customDuplicate : duplicateMode;
		let validCount: number, duplicate: number;
		try {
			validCount = parseInitialUserRecords(rawCount);
			duplicate = validCount === 0 ? 0 : parseDuplicateRecordPercent(rawDuplicate);
		} catch (err) {
			setError(errorText(err));
			return;
		}

		isSubmitPending.current = true;
		setIsSubmitting(true);
		setError('');
		try {
			const id = await api.workspaces.simulatedAccounts.create({
				name: trimmedName,
				userCount: validCount,
				duplicateRecordPercent: duplicate,
				countries: validCount === 0 ? {} : { IT: 100 },
			});
			if (isActive.current) redirect(`simulated-accounts/${id}`);
		} catch (err) {
			if (isActive.current) {
				setError(errorText(err));
				setIsSubmitting(false);
				isSubmitPending.current = false;
			}
		}
	};

	return (
		<main className='route-content simulated-accounts'>
			<Link path='simulated-accounts' className='simulated-accounts__back'>
				← Simulated accounts
			</Link>
			<div className='simulated-accounts__header'>
				<div>
					<h1>Create account</h1>
					<p>Prepare a simulated account for this Development workspace.</p>
				</div>
			</div>
			<form className='simulated-accounts__form' onSubmit={onSubmit} noValidate>
				<label>
					Account name
					<input value={name} maxLength={100} onChange={(event) => setName(event.target.value)} required />
				</label>
				<label>
					Initial user records
					{countMode === 'custom' ? (
						<input
							ref={countInput}
							inputMode='numeric'
							value={customCount}
							onChange={(event) => setCustomCount(event.target.value)}
							aria-label='Custom initial user records'
						/>
					) : (
						<select value={countMode} onChange={(event) => setCountMode(event.target.value)}>
							{COUNT_PRESETS.map((value) => (
								<option key={value} value={value}>
									{value === 0 ? '0 — Start empty' : value.toLocaleString('en-US')}
								</option>
							))}
							<option value='custom'>Custom…</option>
						</select>
					)}
				</label>
				{hasRecords && (
					<>
						<label>
							Duplicate records
							<small>Percentage of records representing the same person with similar data.</small>
							{duplicateMode === 'custom' ? (
								<span className='simulated-accounts__suffix'>
									<input
										ref={duplicateInput}
										inputMode='decimal'
										value={customDuplicate}
										onChange={(event) => setCustomDuplicate(event.target.value)}
										aria-label='Custom duplicate records'
									/>
									<span>%</span>
								</span>
							) : (
								<select
									value={duplicateMode}
									onChange={(event) => setDuplicateMode(event.target.value)}
								>
									{DUPLICATE_PRESETS.map(({ value, label }) => (
										<option key={value} value={value}>
											{label}
										</option>
									))}
									<option value='custom'>Custom…</option>
								</select>
							)}
						</label>
						<div className='simulated-accounts__field'>
							<span>Countries</span>
							<div>Italy · 100%</div>
						</div>
					</>
				)}
				{error !== '' && (
					<div className='simulated-accounts__error' role='alert'>
						{error}
					</div>
				)}
				<div className='simulated-accounts__actions'>
					<Link path='simulated-accounts'>
						<SlButton>Cancel</SlButton>
					</Link>
					<SlButton type='submit' variant='primary' loading={isSubmitting} disabled={isSubmitting}>
						Create simulated account
					</SlButton>
				</div>
			</form>
		</main>
	);
};

interface AccountDetailProps {
	id: string;
}

const AccountDetail = ({ id }: AccountDetailProps) => {
	const { api, redirect } = useContext(AppContext);
	const isActive = useRef(true);
	const isSavePending = useRef(false);
	const isDeletePending = useRef(false);
	const [account, setAccount] = useState<SimulatedAccount | null>(null);
	const [loadError, setLoadError] = useState('');
	const [actionError, setActionError] = useState('');
	const [refresh, setRefresh] = useState(0);
	const [isEditing, setIsEditing] = useState(false);
	const [name, setName] = useState('');
	const [isSaving, setIsSaving] = useState(false);
	const [isDeleting, setIsDeleting] = useState(false);
	const [isDeleteConfirmationOpen, setIsDeleteConfirmationOpen] = useState(false);

	useEffect(
		() => () => {
			isActive.current = false;
		},
		[],
	);
	useEffect(() => {
		const controller = new AbortController();
		let timer: ReturnType<typeof setTimeout>;
		let isPreparing = false;
		const load = async () => {
			try {
				const result = await api.workspaces.simulatedAccounts.get(id, controller.signal);
				if (controller.signal.aborted) return;
				setAccount(result);
				setLoadError('');
				isPreparing = result.status === 'Preparing';
				if (isPreparing) timer = setTimeout(load, POLL_INTERVAL_MS);
			} catch (err) {
				if (controller.signal.aborted) return;
				setLoadError(errorText(err));
				if (isPreparing) timer = setTimeout(load, POLL_INTERVAL_MS);
			}
		};
		load();
		return () => {
			controller.abort();
			clearTimeout(timer);
		};
	}, [id, refresh]);

	const onRename = async (event: FormEvent) => {
		event.preventDefault();
		if (isSavePending.current || account == null) return;
		const trimmedName = name.trim();
		if (trimmedName.length === 0 || trimmedName.length > 100) {
			setActionError('Account name must be 1 to 100 characters.');
			return;
		}
		isSavePending.current = true;
		setIsSaving(true);
		setActionError('');
		try {
			await api.workspaces.simulatedAccounts.rename(account.id, trimmedName);
			if (isActive.current) {
				setAccount((current) => (current == null ? current : { ...current, name: trimmedName }));
				setIsEditing(false);
				setRefresh((value) => value + 1);
			}
		} catch (err) {
			if (isActive.current) setActionError(errorText(err));
		}
		if (isActive.current) {
			isSavePending.current = false;
			setIsSaving(false);
		}
	};

	const onDelete = async () => {
		if (isDeletePending.current || account == null || account.status === 'Preparing') return;
		isDeletePending.current = true;
		setIsDeleting(true);
		setActionError('');
		try {
			await api.workspaces.simulatedAccounts.delete(account.id);
			if (isActive.current) redirect('simulated-accounts');
		} catch (err) {
			if (isActive.current) {
				setActionError(errorText(err));
				setIsDeleteConfirmationOpen(false);
				setIsDeleting(false);
				isDeletePending.current = false;
			}
		}
	};

	return (
		<main className='route-content simulated-accounts'>
			<Link path='simulated-accounts' className='simulated-accounts__back'>
				← Simulated accounts
			</Link>
			{loadError !== '' && (
				<div className='simulated-accounts__error' role='alert'>
					Unable to refresh account: {loadError}{' '}
					<SlButton size='small' onClick={() => setRefresh((value) => value + 1)}>
						Retry
					</SlButton>
				</div>
			)}
			{account == null && loadError === '' && (
				<div className='simulated-accounts__loading'>
					<SlSpinner /> Loading account…
				</div>
			)}
			{account != null && (
				<>
					<div className='simulated-accounts__header'>
						<div>
							<h1>{account.name}</h1>
							<p>Simulated account</p>
						</div>
					</div>
					<div className='simulated-accounts__card'>
						<div className='simulated-accounts__field'>
							<span>Status</span>
							<AccountStatus account={account} />
						</div>
						<div className='simulated-accounts__field'>
							<span>Initial user records</span>
							<div>{account.userCount.toLocaleString()}</div>
						</div>
						{account.userCount > 0 && (
							<>
								<div className='simulated-accounts__field'>
									<span>Duplicate records</span>
									<div>{account.duplicateRecordPercent}%</div>
								</div>
								<div className='simulated-accounts__field'>
									<span>Countries</span>
									<div>Italy</div>
								</div>
							</>
						)}
					</div>
					<div className='simulated-accounts__card'>
						<h2>Account name</h2>
						{isEditing ? (
							<form className='simulated-accounts__rename' onSubmit={onRename}>
								<input
									value={name}
									maxLength={100}
									autoFocus
									onChange={(event) => setName(event.target.value)}
									aria-label='Account name'
								/>
								<SlButton type='submit' variant='primary' loading={isSaving} disabled={isSaving}>
									Save
								</SlButton>
								<SlButton
									onClick={() => {
										setIsEditing(false);
										setActionError('');
									}}
								>
									Cancel
								</SlButton>
							</form>
						) : (
							<SlButton
								onClick={() => {
									setName(account.name);
									setIsEditing(true);
								}}
							>
								Rename account
							</SlButton>
						)}
						{actionError !== '' && (
							<div className='simulated-accounts__error' role='alert'>
								{actionError}
							</div>
						)}
					</div>
					<div className='simulated-accounts__card simulated-accounts__danger'>
						<h2>Delete account</h2>
						{account.status === 'Preparing' ? (
							<p>Deletion is available when preparation finishes.</p>
						) : (
							<p>Permanently delete this simulated account and its initial data.</p>
						)}
						<SlButton
							variant='danger'
							disabled={account.status === 'Preparing'}
							onClick={() => setIsDeleteConfirmationOpen(true)}
						>
							Delete account
						</SlButton>
					</div>
					<AlertDialog
						isOpen={isDeleteConfirmationOpen}
						onClose={() => setIsDeleteConfirmationOpen(false)}
						title='Delete simulated account?'
						variant='danger'
						actions={
							<>
								<SlButton onClick={() => setIsDeleteConfirmationOpen(false)}>Cancel</SlButton>
								<SlButton
									variant='danger'
									loading={isDeleting}
									disabled={isDeleting}
									onClick={onDelete}
								>
									Delete account
								</SlButton>
							</>
						}
					>
						This permanently deletes “{account.name}” and its data.
					</AlertDialog>
				</>
			)}
		</main>
	);
};

export { SimulatedAccounts };
