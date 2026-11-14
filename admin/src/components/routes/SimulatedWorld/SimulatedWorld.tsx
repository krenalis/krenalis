import React, { FormEvent, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import WaBadge from '@awesome.me/webawesome/dist/react/badge/index.js';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaDialog from '@awesome.me/webawesome/dist/react/dialog/index.js';
import type WaDialogElement from '@awesome.me/webawesome/dist/components/dialog/dialog.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaInput from '@awesome.me/webawesome/dist/react/input/index.js';
import type WaInputElement from '@awesome.me/webawesome/dist/components/input/input.js';
import WaOption from '@awesome.me/webawesome/dist/react/option/index.js';
import WaSelect from '@awesome.me/webawesome/dist/react/select/index.js';
import type WaSelectElement from '@awesome.me/webawesome/dist/components/select/select.js';
import WaSpinner from '@awesome.me/webawesome/dist/react/spinner/index.js';
import AppContext from '../../../context/AppContext';
import { NotFoundError } from '../../../lib/api/errors';
import { ConnectionRole } from '../../../lib/api/types/connection';
import { SimulatedSystem } from '../../../lib/api/types/simulatedSystem';
import Grid from '../../base/Grid/Grid';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { Link } from '../../base/Link/Link';
import Section from '../../base/Section/Section';
import NotFound from '../NotFound/NotFound';
import { parseDuplicateRecordPercent, parseInitialUserRecords } from './simulatedSystemValues';
import './SimulatedWorld.css';
import USADemoPreview from './USADemoPreview';

const POLL_INTERVAL_MS = 3000;
const METADATA_POLL_INTERVAL_MS = 15000;
const COUNT_PRESETS = [0, 10, 100, 1000, 10000, 100000, 1000000, 10000000];
const DUPLICATE_PRESETS = [
	{ value: 0, label: 'None (0%)' },
	{ value: 1, label: 'Very low (~1%)' },
	{ value: 2, label: 'Low (~2%)' },
	{ value: 5, label: 'Moderate (~5%)' },
	{ value: 10, label: 'High (~10%)' },
];

const errorText = (err: unknown): string =>
	err instanceof Error ? err.message : 'The request failed. Please try again.';

// Web Awesome does not propagate its host label to the native dialog.
const onDialogShow = (event: Event) => {
	if (event.target !== event.currentTarget) return;
	const dialog = event.currentTarget as WaDialogElement;
	dialog.dialog.setAttribute('aria-label', dialog.label);
};

interface AccountProgressProps {
	generated: number;
	total: number;
}

const AccountProgress = ({ generated, total }: AccountProgressProps) => {
	const count = Number.isFinite(generated) ? Math.min(total, Math.max(0, generated)) : 0;
	return (
		<span className='simulated-systems__progress'>
			<progress value={count} max={total} aria-label='Simulated users created' />
			<span>
				{count.toLocaleString()} / {total.toLocaleString()}
			</span>
		</span>
	);
};

interface ShimmerTextProps {
	children: React.ReactNode;
	isActive: boolean;
	className?: string;
}

const ShimmerText = ({ children, isActive, className = '' }: ShimmerTextProps) => (
	<span className={`${className}${isActive ? ' simulated-systems__shimmer' : ''}`}>{children}</span>
);

interface AccountConnectionProps {
	role: ConnectionRole;
	id: string;
	isRunning: boolean;
}

const AccountConnection = ({ role, id, isRunning }: AccountConnectionProps) => (
	<Link path={`connections/${id}/pipelines`} className='simulated-systems__connection-link'>
		<WaIcon name={role === 'Source' ? 'arrow-down-right-square' : 'arrow-up-right-square'} />
		<ShimmerText isActive={isRunning}>{role}</ShimmerText>
	</Link>
);

interface AccountStatusProps {
	account: SimulatedSystem;
}

const AccountStatus = ({ account }: AccountStatusProps) => (
	<div className='simulated-systems__status'>
		<strong>{account.status}</strong>
		{account.status === 'Preparing' && (
			<>
				<span>Preparation in progress</span>
				{account.userCount > 0 && (
					<AccountProgress generated={account.generatedRecordCount} total={account.userCount} />
				)}
			</>
		)}
		{account.status === 'Available' && <span>Preparation complete</span>}
		{account.status === 'Failed' && <span role='alert'>Preparation failed.</span>}
		{account.status === 'Deleting' && <span>Deleting simulated system and records</span>}
	</div>
);

type CreationStepStatus = 'pending' | 'active' | 'done' | 'skipped' | 'blocked' | 'failed';
type CreationStepName = 'account' | 'users' | 'source' | 'destination';

interface CreationStep {
	status: CreationStepStatus;
	message: string;
}

interface CreationProgress {
	id: string;
	userCount: number;
	generatedRecordCount: number;
	account: CreationStep;
	users: CreationStep;
	source: CreationStep;
	destination: CreationStep;
	isComplete: boolean;
}

const CREATION_STEPS = [
	{ name: 'account', label: 'Create simulated system' },
	{ name: 'users', label: 'Add simulated users' },
	{ name: 'source', label: 'Create source connection' },
	{ name: 'destination', label: 'Create destination connection' },
] as const;

const NOT_REQUESTED_STEPS = [
	{ name: 'users', label: 'Simulated users' },
	{ name: 'source', label: 'Source connection' },
	{ name: 'destination', label: 'Destination connection' },
] as const;

type DeletionStepName = 'source' | 'destination' | 'records' | 'account';

interface DeletionProgress {
	source: CreationStep;
	destination: CreationStep;
	records: CreationStep;
	account: CreationStep;
	isComplete: boolean;
}

const DELETION_STEPS = [
	{ name: 'source', label: 'Delete source connection' },
	{ name: 'destination', label: 'Delete destination connection' },
	{ name: 'records', label: 'Delete simulated system records' },
	{ name: 'account', label: 'Delete simulated system' },
] as const;

const ACCOUNT_COLUMNS: GridColumn[] = [
	{ name: 'Name' },
	{ name: 'Type' },
	{ name: 'Country' },
	{ name: 'Records', alignment: 'right' },
	{ name: 'Linked connections' },
];

const DETAIL_COLUMNS: GridColumn[] = [{ name: 'Field' }, { name: 'Value' }];

const SimulatedWorld = () => {
	const { workspaces, selectedWorkspace, setTitle } = useContext(AppContext);
	const workspace = workspaces.find((item) => item.id === selectedWorkspace);

	useLayoutEffect(() => {
		setTitle('Simulated world');
	}, [setTitle]);

	if (workspace?.environment !== 'development') return <NotFound />;
	return <SimulatedWorldPage key={selectedWorkspace} />;
};

const SimulatedWorldPage = () => {
	const { id } = useParams();
	const [listRefresh, setListRefresh] = useState(0);
	const isCreate = window.location.pathname.endsWith('/create');
	if (id != null) return <AccountDetail key={id} id={id} />;
	return (
		<>
			<AccountList refreshVersion={listRefresh} />
			{isCreate && <CreateAccount onDone={() => setListRefresh((value) => value + 1)} />}
		</>
	);
};

interface AccountListProps {
	refreshVersion: number;
}

const AccountList = ({ refreshVersion }: AccountListProps) => {
	const { api, connections, redirect } = useContext(AppContext);
	const [accounts, setAccounts] = useState<SimulatedSystem[] | null>(null);
	const [error, setError] = useState('');
	const [refresh, setRefresh] = useState(0);
	const [liveRunningConnectionIDs, setLiveRunningConnectionIDs] = useState<Set<string> | null>(null);
	const hasConnections = (accounts ?? []).some(
		(account) => account.sourceConnection != null || account.destinationConnection != null,
	);

	useEffect(() => {
		const controller = new AbortController();
		let timer: ReturnType<typeof setTimeout>;
		let hasLoadedData = false;
		const load = async () => {
			try {
				const result = await api.workspaces.simulatedSystems.find(controller.signal);
				if (controller.signal.aborted) return;
				setAccounts(result);
				hasLoadedData = true;
				setError('');
				timer = setTimeout(
					load,
					result.some((account) => account.status === 'Preparing' || account.status === 'Deleting')
						? POLL_INTERVAL_MS
						: METADATA_POLL_INTERVAL_MS,
				);
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
	}, [refresh, refreshVersion]);

	useEffect(() => {
		if (!hasConnections) return;
		let isActive = true;
		let timer: ReturnType<typeof setTimeout>;
		const load = async () => {
			try {
				const result = await api.workspaces.connections.find();
				if (isActive) {
					const runningIDs = new Set(
						result
							.filter((connection) => connection.pipelines?.some((pipeline) => pipeline.running) === true)
							.map((connection) => connection.id),
					);
					setLiveRunningConnectionIDs((current) =>
						current != null &&
						current.size === runningIDs.size &&
						[...runningIDs].every((id) => current.has(id))
							? current
							: runningIDs,
					);
				}
			} catch {
				// Keep the last known run state and retry on the next poll.
			}
			if (isActive) timer = setTimeout(load, POLL_INTERVAL_MS);
		};
		void load();
		return () => {
			isActive = false;
			clearTimeout(timer);
		};
	}, [hasConnections]);

	const runningConnectionIDs =
		liveRunningConnectionIDs ??
		new Set(
			connections
				.filter((connection) => connection.pipelines.some((pipeline) => pipeline.running))
				.map((connection) => connection.id),
		);
	const rows: GridRow[] = (accounts ?? []).map((account) => ({
		id: account.id,
		cells: [
			<span className='simulated-systems__name-cell' onClick={(event) => event.stopPropagation()}>
				<Link path={`simulated-world/${account.id}`} className='simulated-systems__account-link'>
					{account.name}
				</Link>
				{account.status !== 'Available' && (
					<WaBadge
						className={`simulated-systems__status-badge simulated-systems__status-badge--${account.status.toLowerCase()}`}
						pill
						variant='neutral'
					>
						{account.status}
					</WaBadge>
				)}
			</span>,
			'SaaS application',
			account.countries.US != null ? 'United States' : account.countries.IT != null ? 'Italy' : '—',
			account.userCount.toLocaleString('en-US'),
			account.sourceConnection != null || account.destinationConnection != null ? (
				<span className='simulated-systems__connection-links' onClick={(event) => event.stopPropagation()}>
					{account.sourceConnection != null && (
						<AccountConnection
							role='Source'
							id={account.sourceConnection}
							isRunning={runningConnectionIDs.has(account.sourceConnection)}
						/>
					)}
					{account.destinationConnection != null && (
						<AccountConnection
							role='Destination'
							id={account.destinationConnection}
							isRunning={runningConnectionIDs.has(account.destinationConnection)}
						/>
					)}
				</span>
			) : (
				'—'
			),
		],
		onClick: () => redirect(`simulated-world/${account.id}`),
	}));

	return (
		<main className='route-content simulated-systems'>
			<Link path='simulated-world/create' className='simulated-systems__create-link'>
				<WaButton variant='brand' appearance={accounts != null && accounts.length === 0 ? 'accent' : 'plain'}>
					<WaIcon name='plus-circle' slot='end' />
					Create simulated system
				</WaButton>
			</Link>
			<Section
				title='Simulated systems'
				description={
					<div className='simulated-systems__list-description'>
						<p>Test data ingestion and activation without connecting to real systems.</p>
						<p>
							Create simulated systems with synthetic customer data. Connect them to Krenalis as a source,
							destination, or both.
						</p>
					</div>
				}
				annotated={true}
			>
				<div className='simulated-systems__list-content'>
					{error !== '' && (
						<div className='simulated-systems__error' role='alert'>
							{error}{' '}
							<WaButton size='small' onClick={() => setRefresh((value) => value + 1)}>
								Retry
							</WaButton>
						</div>
					)}
					{accounts == null && error === '' && (
						<div className='simulated-systems__loading'>
							<WaSpinner /> Loading simulated systems…
						</div>
					)}
					{accounts != null && (
						<Grid
							columns={ACCOUNT_COLUMNS}
							rows={rows}
							gridColumnsWidths='minmax(150px, 1.2fr) minmax(110px, 1fr) minmax(130px, 1fr) minmax(100px, .7fr) minmax(190px, 1.3fr)'
							noRowsMessage='No simulated systems yet. Create one to get started.'
						/>
					)}
				</div>
			</Section>
		</main>
	);
};

interface CreateAccountProps {
	onDone: () => void;
}

const CreateAccount = ({ onDone }: CreateAccountProps) => {
	const { api, redirect, setIsLoadingConnections, publicMetadata } = useContext(AppContext);
	const isActive = useRef(true);
	const dialogRef = useRef<WaDialogElement>(null);
	const [isDialogOpen, setIsDialogOpen] = useState(false);
	const nameInput = useRef<HTMLInputElement>(null);
	const countInput = useRef<HTMLInputElement>(null);
	const duplicateInput = useRef<WaInputElement>(null);
	const isSubmitPending = useRef(false);
	const requestController = useRef<AbortController | null>(null);
	const [name, setName] = useState('');
	const [countMode, setCountMode] = useState('10000');
	const [customCount, setCustomCount] = useState('');
	const [duplicateMode, setDuplicateMode] = useState('2');
	const [customDuplicate, setCustomDuplicate] = useState('');
	const [country, setCountry] = useState('IT');
	const [createSource, setCreateSource] = useState(true);
	const [createDestination, setCreateDestination] = useState(true);
	const [nameError, setNameError] = useState('');
	const [countError, setCountError] = useState('');
	const [duplicateError, setDuplicateError] = useState('');
	const [error, setError] = useState('');
	const [progress, setProgress] = useState<CreationProgress | null>(null);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (dialog == null) return;
		let isMounted = true;
		void dialog.updateComplete.then(() => {
			if (isMounted) setIsDialogOpen(true);
		});
		return () => {
			isMounted = false;
		};
	}, []);
	useEffect(
		() => () => {
			isActive.current = false;
			requestController.current?.abort();
		},
		[],
	);
	useEffect(() => {
		if (countMode === 'custom') countInput.current?.focus();
	}, [countMode]);
	useEffect(() => {
		const input = duplicateInput.current;
		if (duplicateMode !== 'custom' || input == null) return;
		void input.updateComplete.then(() => {
			if (duplicateInput.current === input) input.focus();
		});
	}, [duplicateMode]);

	const rawCount = countMode === 'custom' ? customCount : countMode;
	const count = Number(rawCount.replaceAll(',', ''));
	const hasRecords = count > 0;
	const isCreationSuccessful =
		progress?.isComplete === true &&
		CREATION_STEPS.every(({ name }) => progress[name].status === 'done' || progress[name].status === 'skipped');
	const visibleCreationSteps = CREATION_STEPS.filter(
		({ name }) => !isCreationSuccessful || (name !== 'account' && progress?.[name].status !== 'skipped'),
	);
	const notRequestedSteps = NOT_REQUESTED_STEPS.filter(({ name }) => progress?.[name].status === 'skipped');
	const updateStep = (step: CreationStepName, status: CreationStepStatus, message = '') => {
		setProgress((current) => (current == null ? current : { ...current, [step]: { status, message } }));
	};

	const onSubmit = async (event: FormEvent) => {
		event.preventDefault();
		if (isSubmitPending.current) return;
		const trimmedName = name.trim();
		const nameLength = Array.from(trimmedName).length;
		if (nameLength === 0 || nameLength > 100) {
			setNameError('Simulated system name must be 1 to 100 characters.');
			return;
		}
		setNameError('');
		const rawDuplicate = duplicateMode === 'custom' ? customDuplicate : duplicateMode;
		let validCount: number;
		try {
			validCount = parseInitialUserRecords(rawCount);
		} catch (err) {
			setCountError(errorText(err));
			return;
		}
		setCountError('');
		let duplicate: number;
		try {
			duplicate = validCount === 0 ? 0 : parseDuplicateRecordPercent(rawDuplicate);
		} catch (err) {
			setDuplicateError(errorText(err));
			return;
		}
		setDuplicateError('');

		isSubmitPending.current = true;
		setError('');
		const shouldCreateSource = createSource;
		const shouldCreateDestination = createDestination;
		setProgress({
			id: '',
			userCount: validCount,
			generatedRecordCount: 0,
			account: { status: 'active', message: '' },
			users: { status: validCount === 0 ? 'skipped' : 'pending', message: '' },
			source: { status: shouldCreateSource ? 'pending' : 'skipped', message: '' },
			destination: { status: shouldCreateDestination ? 'pending' : 'skipped', message: '' },
			isComplete: false,
		});
		let id: string;
		try {
			id = await api.workspaces.simulatedSystems.create({
				name: trimmedName,
				userCount: validCount,
				duplicateRecordPercent: duplicate,
				countries: validCount === 0 ? {} : { [country]: 100 },
			});
		} catch (err) {
			if (isActive.current) {
				setProgress((current) =>
					current == null
						? current
						: {
								...current,
								account: { status: 'failed', message: errorText(err) },
								users: { status: validCount === 0 ? 'skipped' : 'blocked', message: '' },
								source: { status: shouldCreateSource ? 'blocked' : 'skipped', message: '' },
								destination: { status: shouldCreateDestination ? 'blocked' : 'skipped', message: '' },
								isComplete: true,
							},
				);
				isSubmitPending.current = false;
			}
			return;
		}
		if (!isActive.current) return;
		setProgress((current) =>
			current == null
				? current
				: {
						...current,
						id,
						account: { status: 'done', message: '' },
						users: { status: validCount === 0 ? 'skipped' : 'active', message: '' },
					},
		);

		const controller = new AbortController();
		requestController.current = controller;
		while (isActive.current) {
			let account: SimulatedSystem;
			try {
				account = await api.workspaces.simulatedSystems.get(id, controller.signal);
			} catch (err) {
				if (!isActive.current) return;
				setError(`Unable to refresh simulated system progress: ${errorText(err)} Retrying…`);
				await new Promise<void>((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
				continue;
			}
			if (!isActive.current) return;
			setError('');
			setProgress((current) =>
				current == null ? current : { ...current, generatedRecordCount: account.generatedRecordCount },
			);
			if (account.status === 'Available') {
				if (validCount > 0) updateStep('users', 'done');
				break;
			}
			if (account.status === 'Failed') {
				updateStep('users', 'failed', 'Simulated system preparation failed.');
				setProgress((current) =>
					current == null
						? current
						: {
								...current,
								source: { status: shouldCreateSource ? 'blocked' : 'skipped', message: '' },
								destination: { status: shouldCreateDestination ? 'blocked' : 'skipped', message: '' },
								isComplete: true,
							},
				);
				isSubmitPending.current = false;
				return;
			}
			await new Promise<void>((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
		}
		if (!isActive.current) return;
		let hasCreatedConnection = false;
		for (const role of ['Source', 'Destination'] as const) {
			const step = role === 'Source' ? 'source' : 'destination';
			if (role === 'Source' ? !shouldCreateSource : !shouldCreateDestination) continue;
			updateStep(step, 'active');
			try {
				await api.workspaces.createConnection(
					{
						name: trimmedName,
						role,
						connector: 'simulated-application',
						sendingMode: role === 'Destination' ? 'Server' : null,
						settings: { account: id },
						linkedConnections: null,
					},
					'',
				);
			} catch (err) {
				if (!isActive.current) return;
				updateStep(step, 'failed', errorText(err));
				continue;
			}
			if (!isActive.current) return;
			updateStep(step, 'done');
			hasCreatedConnection = true;
		}
		if (hasCreatedConnection) setIsLoadingConnections(true);
		setProgress((current) => (current == null ? current : { ...current, isComplete: true }));
		isSubmitPending.current = false;
	};

	return (
		<WaDialog
			ref={dialogRef}
			className='simulated-systems__dialog simulated-systems__dialog--create'
			label='Create simulated system'
			withoutHeader
			lightDismiss
			open={isDialogOpen}
			onWaShow={onDialogShow}
			onWaAfterShow={() => {
				nameInput.current?.focus({ preventScroll: true });
			}}
			onWaHide={(event) => {
				if (event.target !== event.currentTarget || !isDialogOpen) return;
				event.preventDefault();
				if (progress == null) redirect('simulated-world');
			}}
		>
			{progress == null && <h2 className='simulated-systems__dialog-title'>Create simulated system</h2>}
			{progress == null ? (
				<form className='simulated-systems__form' onSubmit={onSubmit} noValidate>
					<div className='simulated-systems__field'>
						<label htmlFor='simulated-system-name'>Name</label>
						<input
							id='simulated-system-name'
							ref={nameInput}
							value={name}
							maxLength={200}
							aria-invalid={nameError !== ''}
							aria-describedby={
								nameError !== ''
									? 'simulated-system-name-note simulated-system-name-error'
									: 'simulated-system-name-note'
							}
							onChange={(event) => {
								// HTML maxLength counts UTF-16 units; the backend counts Unicode characters.
								if (Array.from(event.target.value).length > 100) {
									event.target.value = name;
									return;
								}
								setName(event.target.value);
								setNameError('');
							}}
							required
						/>
						<small id='simulated-system-name-note' className='simulated-systems__name-note'>
							Also used to name connections created alongside this simulated system.
						</small>
						{nameError !== '' && (
							<span
								id='simulated-system-name-error'
								className='simulated-systems__field-error'
								role='alert'
							>
								{nameError}
							</span>
						)}
					</div>
					<div className='simulated-systems__field'>
						<label htmlFor='simulated-system-user-count'>User records to generate</label>
						{countMode === 'custom' ? (
							<input
								id='simulated-system-user-count'
								ref={countInput}
								inputMode='numeric'
								value={customCount}
								aria-invalid={countError !== ''}
								aria-describedby={countError !== '' ? 'simulated-system-user-count-error' : undefined}
								onChange={(event) => {
									setCustomCount(event.target.value);
									setCountError('');
								}}
							/>
						) : (
							<WaSelect
								id='simulated-system-user-count'
								aria-label='User records to generate'
								value={countMode}
								aria-invalid={countError !== ''}
								aria-describedby={countError !== '' ? 'simulated-system-user-count-error' : undefined}
								onChange={(event) => {
									const value = String((event.target as WaSelectElement).value);
									if (value === 'custom') setCustomCount(Number(countMode).toLocaleString('en-US'));
									setCountMode(value);
									setCountError('');
								}}
							>
								{COUNT_PRESETS.map((value) => (
									<WaOption key={value} value={String(value)}>
										{value === 0 ? '0 — Start empty' : value.toLocaleString('en-US')}
									</WaOption>
								))}
								<WaOption value='custom'>Custom…</WaOption>
							</WaSelect>
						)}
						{countError !== '' && (
							<span
								id='simulated-system-user-count-error'
								className='simulated-systems__field-error'
								role='alert'
							>
								{countError}
							</span>
						)}
					</div>
					{hasRecords && (
						<>
							<div className='simulated-systems__field'>
								<label htmlFor='simulated-system-duplicate-percent'>Duplicate records</label>
								<small>
									Controls how often multiple records represent the same person.
									<br />
									Use duplicates to test identity resolution and profile unification.
								</small>
								{duplicateMode === 'custom' ? (
									<WaInput
										id='simulated-system-duplicate-percent'
										aria-label='Duplicate records'
										ref={duplicateInput}
										inputmode='decimal'
										value={customDuplicate}
										aria-invalid={duplicateError !== ''}
										aria-describedby={
											duplicateError !== ''
												? 'simulated-system-duplicate-percent-error'
												: undefined
										}
										onInput={(event) => {
											setCustomDuplicate((event.target as WaInputElement).value);
											setDuplicateError('');
										}}
									>
										<span slot='end'>%</span>
									</WaInput>
								) : (
									<WaSelect
										id='simulated-system-duplicate-percent'
										aria-label='Duplicate records'
										value={duplicateMode}
										aria-invalid={duplicateError !== ''}
										aria-describedby={
											duplicateError !== ''
												? 'simulated-system-duplicate-percent-error'
												: undefined
										}
										onChange={(event) => {
											const value = String((event.target as WaSelectElement).value);
											if (value === 'custom') setCustomDuplicate(duplicateMode);
											setDuplicateMode(value);
											setDuplicateError('');
										}}
									>
										{DUPLICATE_PRESETS.map(({ value, label }) => (
											<WaOption key={value} value={String(value)}>
												{label}
											</WaOption>
										))}
										<WaOption value='custom'>Custom…</WaOption>
									</WaSelect>
								)}
								{duplicateError !== '' && (
									<span
										id='simulated-system-duplicate-percent-error'
										className='simulated-systems__field-error'
										role='alert'
									>
										{duplicateError}
									</span>
								)}
							</div>
							<div className='simulated-systems__field'>
								<label htmlFor='simulated-system-country'>Country distribution</label>
								<WaSelect
									id='simulated-system-country'
									aria-label='Country distribution'
									value={country}
									onChange={(event) => setCountry(String((event.target as WaSelectElement).value))}
								>
									<WaOption value='IT'>Italy</WaOption>
									{publicMetadata.usaDemoEnabled === true && (
										<WaOption value='US'>United States · Internal demo</WaOption>
									)}
								</WaSelect>
							</div>
						</>
					)}
					<fieldset className='simulated-systems__connections'>
						<legend>Also create connections</legend>
						<label>
							<input
								type='checkbox'
								checked={createSource}
								onChange={(event) => setCreateSource(event.target.checked)}
							/>
							Source connection
						</label>
						<label>
							<input
								type='checkbox'
								checked={createDestination}
								onChange={(event) => setCreateDestination(event.target.checked)}
							/>
							Destination connection
						</label>
					</fieldset>
					<div className='simulated-systems__actions'>
						<Link path='simulated-world'>
							<WaButton size='small'>Cancel</WaButton>
						</Link>
						<WaButton
							className='simulated-systems__create-button'
							size='small'
							type='submit'
							variant='brand'
						>
							Create
						</WaButton>
					</div>
				</form>
			) : (
				<div className='simulated-systems__dialog-progress'>
					<h2>
						{isCreationSuccessful
							? 'Simulated system created'
							: progress.isComplete
								? progress.id === ''
									? 'Simulated system creation failed'
									: CREATION_STEPS.some(({ name }) => progress[name].status === 'failed')
										? 'Simulated system setup incomplete'
										: 'Simulated system setup finished'
								: 'Preparing simulated system'}
					</h2>
					{isCreationSuccessful && (
						<p className='simulated-systems__creation-success'>
							<span aria-hidden='true'>✓</span>
							Simulated system created successfully
						</p>
					)}
					{visibleCreationSteps.length > 0 && (
						<ol className='simulated-systems__steps'>
							{visibleCreationSteps.map(({ name: stepName, label }) => {
								const step = progress[stepName];
								return (
									<li
										key={stepName}
										className={`simulated-systems__step simulated-systems__step--${step.status}`}
									>
										<span className='simulated-systems__step-marker' aria-hidden='true' />
										<div className='simulated-systems__step-content'>
											<div className='simulated-systems__step-line'>
												<ShimmerText
													isActive={step.status === 'active'}
													className='simulated-systems__step-label'
												>
													{label}
												</ShimmerText>
												{stepName === 'users' &&
													step.status === 'active' &&
													progress.userCount > 0 && (
														<AccountProgress
															generated={progress.generatedRecordCount}
															total={progress.userCount}
														/>
													)}
												{stepName === 'users' && step.status === 'done' && (
													<span>
														{progress.generatedRecordCount.toLocaleString()} records
													</span>
												)}
											</div>
											{step.status === 'skipped' && <small>Not requested</small>}
											{step.status === 'blocked' && (
												<small>Simulated system preparation did not complete</small>
											)}
											{step.status === 'failed' && <small role='alert'>{step.message}</small>}
										</div>
									</li>
								);
							})}
						</ol>
					)}
					{isCreationSuccessful && notRequestedSteps.length > 0 && (
						<section className='simulated-systems__not-requested'>
							<h3>Not requested</h3>
							<ul>
								{notRequestedSteps.map(({ name, label }) => (
									<li key={name}>{label}</li>
								))}
							</ul>
						</section>
					)}
					{error !== '' && (
						<div className='simulated-systems__error' role='alert'>
							{error}
						</div>
					)}
					{progress.isComplete && (
						<div className='simulated-systems__actions'>
							{progress.id !== '' ? (
								<Link path='simulated-world' onClick={onDone}>
									<WaButton className='simulated-systems__ok-button' variant='brand'>
										OK
									</WaButton>
								</Link>
							) : (
								<WaButton onClick={() => setProgress(null)}>Back to form</WaButton>
							)}
						</div>
					)}
				</div>
			)}
		</WaDialog>
	);
};

interface AccountDetailProps {
	id: string;
}

const AccountDetail = ({ id }: AccountDetailProps) => {
	const { api, connections, redirect, setIsLoadingConnections } = useContext(AppContext);
	const isActive = useRef(true);
	const isSavePending = useRef(false);
	const isDeletePending = useRef(false);
	const [account, setAccount] = useState<SimulatedSystem | null>(null);
	const [loadError, setLoadError] = useState('');
	const [actionError, setActionError] = useState('');
	const [refresh, setRefresh] = useState(0);
	const [isEditing, setIsEditing] = useState(false);
	const [name, setName] = useState('');
	const [isSaving, setIsSaving] = useState(false);
	const [isDeleting, setIsDeleting] = useState(false);
	const [isDeleteConfirmationOpen, setIsDeleteConfirmationOpen] = useState(false);
	const [deletionProgress, setDeletionProgress] = useState<DeletionProgress | null>(null);

	useEffect(
		() => () => {
			isActive.current = false;
		},
		[],
	);
	useEffect(() => {
		if (deletionProgress != null) return;
		const controller = new AbortController();
		let timer: ReturnType<typeof setTimeout>;
		let pollInterval = METADATA_POLL_INTERVAL_MS;
		let wasDeleting = false;
		const load = async () => {
			try {
				const result = await api.workspaces.simulatedSystems.get(id, controller.signal);
				if (controller.signal.aborted) return;
				setAccount(result);
				setLoadError('');
				wasDeleting = result.status === 'Deleting';
				pollInterval =
					result.status === 'Preparing' || result.status === 'Deleting'
						? POLL_INTERVAL_MS
						: METADATA_POLL_INTERVAL_MS;
				timer = setTimeout(load, pollInterval);
			} catch (err) {
				if (controller.signal.aborted) return;
				if (wasDeleting && err instanceof NotFoundError) {
					redirect('simulated-world');
					return;
				}
				setLoadError(errorText(err));
				timer = setTimeout(load, POLL_INTERVAL_MS);
			}
		};
		load();
		return () => {
			controller.abort();
			clearTimeout(timer);
		};
	}, [id, refresh, deletionProgress != null]);

	const onRename = async (event: FormEvent) => {
		event.preventDefault();
		if (isSavePending.current || account == null || account.status === 'Deleting') return;
		const trimmedName = name.trim();
		const nameLength = Array.from(trimmedName).length;
		if (nameLength === 0 || nameLength > 100) {
			setActionError('Simulated system name must be 1 to 100 characters.');
			return;
		}
		isSavePending.current = true;
		setIsSaving(true);
		setActionError('');
		try {
			await api.workspaces.simulatedSystems.rename(account.id, trimmedName);
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

	const runDeletion = async (initial: DeletionProgress, shouldRefresh = false) => {
		if (isDeletePending.current || account == null) return;
		isDeletePending.current = true;
		setIsDeleting(true);
		setActionError('');
		let current = initial;
		const updateStep = (name: DeletionStepName, status: CreationStepStatus, message = '') => {
			current = { ...current, [name]: { status, message } };
			if (isActive.current) setDeletionProgress(current);
		};
		const finish = () => {
			for (const name of ['source', 'destination'] as const) {
				if (current[name].status !== 'skipped') updateStep(name, 'done');
			}
			updateStep('records', 'done');
			updateStep('account', 'done');
			if (isActive.current) setDeletionProgress({ ...current, isComplete: true });
			isDeletePending.current = false;
			if (isActive.current) setIsDeleting(false);
		};
		let currentAccount = account;
		if (shouldRefresh) {
			try {
				currentAccount = await api.workspaces.simulatedSystems.get(account.id);
			} catch (err) {
				if (!isActive.current) return;
				if (err instanceof NotFoundError) {
					finish();
					return;
				}
				setActionError(`Unable to refresh simulated system: ${errorText(err)}`);
				isDeletePending.current = false;
				setIsDeleting(false);
				return;
			}
			if (!isActive.current) return;
		}
		for (const { name, connection } of [
			{ name: 'source', connection: currentAccount.sourceConnection },
			{ name: 'destination', connection: currentAccount.destinationConnection },
		] as const) {
			if (connection == null) {
				if (current[name].status !== 'skipped') updateStep(name, 'done');
				continue;
			}
			updateStep(name, 'active');
			try {
				await api.workspaces.connections.delete(connection);
			} catch (err) {
				if (!(err instanceof NotFoundError)) {
					updateStep(name, 'failed', errorText(err));
					isDeletePending.current = false;
					if (isActive.current) setIsDeleting(false);
					return;
				}
			}
			if (!isActive.current) return;
			updateStep(name, 'done');
			setIsLoadingConnections(true);
		}
		updateStep('records', 'active');
		try {
			await api.workspaces.simulatedSystems.delete(account.id);
		} catch (err) {
			if (!(err instanceof NotFoundError)) {
				updateStep('records', 'failed', errorText(err));
				isDeletePending.current = false;
				if (isActive.current) setIsDeleting(false);
				return;
			}
			finish();
			return;
		}
		while (isActive.current) {
			try {
				await api.workspaces.simulatedSystems.get(account.id);
				if (isActive.current) setActionError('');
			} catch (err) {
				if (!isActive.current) return;
				if (err instanceof NotFoundError) {
					finish();
					return;
				}
				setActionError(`Unable to refresh deletion progress: ${errorText(err)} Retrying…`);
			}
			await new Promise<void>((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
		}
	};

	const onDelete = () => {
		if (
			isDeletePending.current ||
			account == null ||
			account.status === 'Preparing' ||
			account.status === 'Deleting'
		)
			return;
		const initial: DeletionProgress = {
			source: { status: account.sourceConnection == null ? 'skipped' : 'pending', message: '' },
			destination: { status: account.destinationConnection == null ? 'skipped' : 'pending', message: '' },
			records: { status: 'pending', message: '' },
			account: { status: 'pending', message: '' },
			isComplete: false,
		};
		setIsDeleteConfirmationOpen(false);
		setDeletionProgress(initial);
		void runDeletion(initial);
	};

	const settingsRows: GridRow[] = [];
	const connectionRows: GridRow[] = [];
	if (account != null) {
		settingsRows.push({ cells: [<strong>Initial user records</strong>, account.userCount.toLocaleString()] });
		if (account.userCount > 0) {
			settingsRows.push(
				{ cells: [<strong>Duplicate records</strong>, `${account.duplicateRecordPercent}%`] },
				{ cells: [<strong>Countries</strong>, account.countries.US != null ? 'United States' : 'Italy'] },
			);
		}
		for (const role of ['source', 'destination'] as const) {
			const connection = role === 'source' ? account.sourceConnection : account.destinationConnection;
			const connectionName = connections.find((item) => item.id === connection)?.name;
			connectionRows.push({
				cells: [
					<strong>{role === 'source' ? 'Source connection' : 'Destination connection'}</strong>,
					connection == null ? (
						'Not created'
					) : (
						<Link path={`connections/${connection}/pipelines`}>{connectionName ?? 'View connection'}</Link>
					),
				],
			});
		}
	}

	return (
		<main className='route-content simulated-systems simulated-systems--detail'>
			<Link path='simulated-world' className='simulated-systems__back'>
				← Simulated world
			</Link>
			{loadError !== '' && (
				<div className='simulated-systems__error' role='alert'>
					Unable to refresh simulated system: {loadError}{' '}
					<WaButton size='small' onClick={() => setRefresh((value) => value + 1)}>
						Retry
					</WaButton>
				</div>
			)}
			{account == null && loadError === '' && (
				<div className='simulated-systems__loading'>
					<WaSpinner /> Loading simulated system…
				</div>
			)}
			{account != null && (
				<>
					<div className='simulated-systems__header'>
						<h1>{account.name}</h1>
						{isEditing ? (
							<form className='simulated-systems__rename' onSubmit={onRename}>
								<input
									value={name}
									maxLength={200}
									autoFocus
									onChange={(event) => {
										// HTML maxLength counts UTF-16 units; the backend counts Unicode characters.
										if (Array.from(event.target.value).length > 100) {
											event.target.value = name;
											return;
										}
										setName(event.target.value);
									}}
									aria-label='Simulated system name'
								/>
								<WaButton
									type='submit'
									variant='brand'
									loading={isSaving}
									disabled={isSaving || account.status === 'Deleting'}
								>
									Save
								</WaButton>
								<WaButton
									onClick={() => {
										setIsEditing(false);
										setActionError('');
									}}
								>
									Cancel
								</WaButton>
							</form>
						) : (
							<WaButton
								disabled={account.status === 'Deleting'}
								onClick={() => {
									setName(account.name);
									setIsEditing(true);
								}}
							>
								Rename simulated system
							</WaButton>
						)}
					</div>
					{actionError !== '' && deletionProgress == null && (
						<div className='simulated-systems__error' role='alert'>
							{actionError}
						</div>
					)}
					{account.status === 'Available' && account.userCount > 0 && account.countries.US != null && (
						<USADemoPreview accountID={account.id} />
					)}
					<Section
						title='Status'
						description='Current preparation state of this simulated system.'
						annotated={true}
						padded={true}
					>
						<AccountStatus account={account} />
					</Section>
					<Section
						title='Initial records'
						description='The settings used when preparing this simulated system.'
						annotated={true}
					>
						<Grid columns={DETAIL_COLUMNS} rows={settingsRows} />
					</Section>
					<Section
						title='Connections'
						description='The source and destination connections associated with this simulated system.'
						annotated={true}
					>
						<Grid columns={DETAIL_COLUMNS} rows={connectionRows} />
					</Section>
					<Section
						title='Delete simulated system'
						description='Permanently delete this simulated system, its connections, and its records.'
						annotated={true}
						padded={true}
					>
						{account.status === 'Preparing' && <p>Deletion is available when preparation finishes.</p>}
						{account.status === 'Deleting' && <p>Deletion is in progress.</p>}
						<WaButton
							variant='danger'
							disabled={account.status === 'Preparing' || account.status === 'Deleting'}
							onClick={() => setIsDeleteConfirmationOpen(true)}
						>
							Delete simulated system
						</WaButton>
					</Section>
					<WaDialog
						className='simulated-systems__dialog'
						label='Delete simulated system'
						withoutHeader
						lightDismiss
						open={isDeleteConfirmationOpen || deletionProgress != null}
						onWaShow={onDialogShow}
						onWaHide={(event) => {
							if (
								event.target !== event.currentTarget ||
								!(isDeleteConfirmationOpen || deletionProgress != null)
							)
								return;
							event.preventDefault();
							if (deletionProgress == null) setIsDeleteConfirmationOpen(false);
						}}
					>
						{deletionProgress == null ? (
							<>
								<h2 className='simulated-systems__dialog-title'>Delete simulated system?</h2>
								<p className='simulated-systems__dialog-description'>
									This permanently deletes “{account.name}”, its connections, and its records.
								</p>
								<div className='simulated-systems__actions'>
									<WaButton onClick={() => setIsDeleteConfirmationOpen(false)}>Cancel</WaButton>
									<WaButton variant='danger' onClick={onDelete}>
										Delete simulated system
									</WaButton>
								</div>
							</>
						) : (
							<div className='simulated-systems__dialog-progress'>
								<h2 className='simulated-systems__dialog-title'>
									{deletionProgress.isComplete
										? 'Simulated system deleted'
										: isDeleting
											? 'Deleting simulated system'
											: 'Simulated system deletion interrupted'}
								</h2>
								<ol className='simulated-systems__steps simulated-systems__steps--delete'>
									{DELETION_STEPS.map(({ name, label }) => {
										const step = deletionProgress[name];
										return (
											<li
												key={name}
												className={`simulated-systems__step simulated-systems__step--${step.status}`}
											>
												<span className='simulated-systems__step-marker' aria-hidden='true' />
												<div className='simulated-systems__step-content'>
													<ShimmerText
														isActive={step.status === 'active'}
														className='simulated-systems__step-label'
													>
														{label}
													</ShimmerText>
													{step.status === 'failed' && (
														<small role='alert'>{step.message}</small>
													)}
												</div>
											</li>
										);
									})}
								</ol>
								{actionError !== '' && (
									<div className='simulated-systems__error' role='alert'>
										{actionError}
									</div>
								)}
								{deletionProgress.isComplete ? (
									<div className='simulated-systems__actions'>
										<Link path='simulated-world'>
											<WaButton className='simulated-systems__ok-button' variant='brand'>
												OK
											</WaButton>
										</Link>
									</div>
								) : !isDeleting ? (
									<div className='simulated-systems__actions'>
										<WaButton
											onClick={() => {
												setDeletionProgress(null);
												setActionError('');
												setRefresh((value) => value + 1);
											}}
										>
											Close
										</WaButton>
										<WaButton
											variant='danger'
											onClick={() => void runDeletion(deletionProgress, true)}
										>
											Retry
										</WaButton>
									</div>
								) : null}
							</div>
						)}
					</WaDialog>
				</>
			)}
		</main>
	);
};

export { SimulatedWorld };
