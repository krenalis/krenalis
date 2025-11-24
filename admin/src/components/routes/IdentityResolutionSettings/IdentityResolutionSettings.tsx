import React, { useEffect, useState } from 'react';
import './IdentityResolutionSettings.css';
import Section from '../../base/Section/Section';
import * as icons from '../../../constants/icons';
import { useContext } from 'react';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import { ObjectType } from '../../../lib/api/types/types';
import { Identifiers } from '../../../lib/api/types/identifiers';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboboxItems';
import IconWrapper from '../../base/IconWrapper/IconWrapper';
import { Link } from '../../base/Link/Link';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import { Combobox } from '../../base/Combobox/Combobox';

const IdentityResolutionSettings = () => {
	const [runOnBatchImport, setRunOnBatchImport] = useState<boolean>(false);
	const [identifiers, setIdentifiers] = useState<Identifiers>();
	const [suitableAsIdentifiers, setSuitableAsIdentifiers] = useState<ObjectType>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const onRunOnBatchImportChange = () => setRunOnBatchImport(!runOnBatchImport);

	const { api, handleError, showStatus, workspaces, setIsLoadingWorkspaces, selectedWorkspace } =
		useContext(AppContext);

	useEffect(() => {
		const fetchData = async () => {
			const workspace = workspaces.find((w) => w.id === selectedWorkspace);
			setRunOnBatchImport(workspace.resolveIdentitiesOnBatchImport);
			setIdentifiers(workspace.identifiers);
			let suitableAsIdentifiers: ObjectType;
			try {
				suitableAsIdentifiers = await api.workspaces.profilePropertiesSuitableAsIdentifiers();
			} catch (err) {
				handleError(err);
				return;
			}
			setSuitableAsIdentifiers(suitableAsIdentifiers);
			setTimeout(() => setIsLoading(false), 150);
		};
		fetchData();
	}, [selectedWorkspace]);

	const onAddIdentifier = () => {
		const ident = [...identifiers];
		ident.push('');
		setIdentifiers(ident);
	};

	const onUpdateIdentifier = (name: string, value: string) => {
		const pos = Number(name);
		const ident = [...identifiers];
		ident[pos - 1] = value;
		setIdentifiers(ident);
	};

	const moveIdentifierDown = (position: number) => {
		const elementIndex = position - 1;
		const element = identifiers[elementIndex];
		const nextElementIndex = elementIndex + 1;
		const nextElement = identifiers[nextElementIndex];
		const ident = [
			...identifiers.slice(0, elementIndex),
			nextElement,
			element,
			...identifiers.slice(nextElementIndex + 1),
		];
		setIdentifiers(ident);
	};

	const moveIdentifierUp = (position: number) => {
		const elementIndex = position - 1;
		const element = identifiers[elementIndex];
		const previousElementIndex = elementIndex - 1;
		const previousElement = identifiers[previousElementIndex];
		const ident = [
			...identifiers.slice(0, previousElementIndex),
			element,
			previousElement,
			...identifiers.slice(elementIndex + 1),
		];
		setIdentifiers(ident);
	};

	const removeIdentifier = (position: number) => {
		const ident = [...identifiers];
		ident.splice(position - 1, 1);
		setIdentifiers(ident);
	};

	const onSave = async () => {
		try {
			validateIdentifiers(identifiers);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsSaving(true);
		try {
			await api.workspaces.updateIdentityResolution(runOnBatchImport, identifiers);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				handleError(err);
			}, 500);
			return;
		}
		setIsLoadingWorkspaces(true);
		setTimeout(() => {
			setIsSaving(false);
			showStatus({ variant: 'success', icon: icons.OK, text: 'Identity Resolution settings saved successfully' });
		}, 500);
	};

	const items = getSchemaComboboxItems(suitableAsIdentifiers);
	const identifiersComboboxItems = [];
	for (const it of items) {
		const isAlreadyUsed = identifiers.includes(it.term);
		if (isAlreadyUsed) {
			continue;
		}
		identifiersComboboxItems.push(it);
	}

	return (
		<div className='identifiers'>
			{isLoading ? (
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			) : suitableAsIdentifiers == null ? (
				<div className='identifiers__no-schema'>
					<IconWrapper name='person-exclamation' size={40} />
					<div className='identifiers__no-schema-description'>
						The current profile schema doesn't include any property that can be used as an identifier
					</div>
					<Link path='schema'>
						<SlButton variant='primary' className='identifiers__no-schema-button'>
							See schema
						</SlButton>
					</Link>
				</div>
			) : (
				<div>
					<Section
						title='Automatic execution'
						description='Define when the Identity Resolution should be automatically started'
						padded={true}
						annotated={true}
					>
						<SlCheckbox
							className='identifiers__automatic-execution'
							checked={runOnBatchImport}
							onSlChange={onRunOnBatchImportChange}
						>
							Automatically run the Identity Resolution when importing users from apps, files and
							databases
						</SlCheckbox>
					</Section>
					<Section
						title='Identifiers'
						description='Define the identifiers used to resolve the identity of the users'
						padded={true}
						annotated={true}
					>
						{identifiers.map((identifier, i) => {
							const position = i + 1;
							return (
								<div key={position} className='identifiers__identifier'>
									<div className='identifiers__identifier-position'>{position}</div>
									<Combobox
										className='identifiers__identifier-input'
										name={String(position)}
										value={identifier}
										onInput={onUpdateIdentifier}
										onSelect={onUpdateIdentifier}
										isExpression={false}
										controlled={true}
										items={identifiersComboboxItems}
										size='small'
									/>
									<SlDropdown>
										<SlButton size='small' className='identifiers__identifier-menu' slot='trigger'>
											<SlIcon slot='prefix' name='three-dots'></SlIcon>
										</SlButton>
										<SlMenu>
											<SlMenuItem
												className='identifiers__mapping-up'
												onClick={() => moveIdentifierUp(position)}
												disabled={position === 1}
											>
												<SlIcon slot='prefix' name='arrow-up-circle' />
												Move up
											</SlMenuItem>
											<SlMenuItem
												className='identifiers__mapping-down'
												onClick={() => moveIdentifierDown(position)}
												disabled={position === identifiers.length}
											>
												<SlIcon slot='prefix' name='arrow-down-circle' />
												Move down
											</SlMenuItem>
											<SlMenuItem
												className='identifiers__mapping-remove'
												onClick={() => removeIdentifier(position)}
											>
												<SlIcon slot='prefix' name='trash3' />
												Remove
											</SlMenuItem>
										</SlMenu>
									</SlDropdown>
								</div>
							);
						})}
						<SlButton
							className='identifiers__add'
							size='small'
							variant='neutral'
							onClick={onAddIdentifier}
							circle
						>
							<SlIcon name='plus' />
						</SlButton>
					</Section>
					<SlButton
						className='identifiers__save-button'
						onClick={onSave}
						variant='primary'
						loading={isSaving}
					>
						Save
					</SlButton>
				</div>
			)}
		</div>
	);
};

const validateIdentifiers = (identifiers: Identifiers) => {
	for (const identifier of identifiers) {
		if (identifier === '') {
			throw new Error('identifier cannot be empty');
		}
	}
};

export default IdentityResolutionSettings;
