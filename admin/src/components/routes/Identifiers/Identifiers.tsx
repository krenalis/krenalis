import React, { useEffect, useState, useRef } from 'react';
import './Identifiers.css';
import IdentifiersMapping from '../../shared/IdentifiersMapping/IdentifiersMapping';
import Section from '../../shared/Section/Section';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { useContext } from 'react';
import AppContext from '../../../context/AppContext';
import { ComboBoxInput, ComboBoxList } from '../../shared/ComboBox/ComboBox';
import {
	validateIdentifiersMapping,
	transformAnonymousIdentifiers,
	untransformAnonymousIdentifiers,
	TransformedIdentifiers,
} from '../../../lib/helpers/transformedIdentifiers';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import { ObjectType } from '../../../types/external/types';
import { Identifiers } from '../../../types/external/identifiers';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';

const Identifiers = () => {
	const [anonymousIdentifiers, setAnonymousIdentifiers] = useState<TransformedIdentifiers>();
	const [identifiers, setIdentifiers] = useState<Identifiers>();
	const [eventSchema, setEventSchema] = useState<ObjectType>();
	const [userSchema, setUserSchema] = useState<ObjectType>();
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { api, handleError, showStatus, workspaces, setIsLoadingWorkspaces, selectedWorkspace, redirect } =
		useContext(AppContext);

	const identifiersListRef = useRef(null);

	useEffect(() => {
		const fetchData = async () => {
			const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
			setIdentifiers(workspace.Identifiers);
			const anonymousIdentifiers = transformAnonymousIdentifiers(workspace.AnonymousIdentifiers);
			setAnonymousIdentifiers(anonymousIdentifiers);
			let eventSchema: ObjectType;
			try {
				eventSchema = await api.eventsSchema();
			} catch (err) {
				handleError(err);
				return;
			}
			setEventSchema(eventSchema);
			let userSchema: ObjectType;
			try {
				userSchema = await api.workspaces.userSchema();
			} catch (err) {
				handleError(err);
				return;
			}
			setUserSchema(userSchema);
			setIsLoading(false);
		};
		fetchData();
	}, [selectedWorkspace]);

	const onConnectDataWarehouse = () => {
		redirect('settings/data-warehouse');
	};

	const onSelectIdentifier = (input, value) => {
		const pos = Number(input.name);
		const ident = [...identifiers];
		ident[pos - 1] = value;
		setIdentifiers(ident);
	};

	const onAddIdentifier = () => {
		const ident = [...identifiers];
		ident.push('');
		setIdentifiers(ident);
	};

	const onUpdateIdentifier = (e) => {
		const { name, value } = e.target;
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
		setIsSaving(true);
		const errorMessage = validateIdentifiersMapping(anonymousIdentifiers!);
		if (errorMessage) {
			setTimeout(() => {
				setIsSaving(false);
				handleError(errorMessage);
			}, 500);
			return;
		}
		const untransformedAnonymousIdentifiers = untransformAnonymousIdentifiers(anonymousIdentifiers!);
		try {
			await api.workspaces.setIdentifiers(identifiers, untransformedAnonymousIdentifiers);
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
			showStatus({ variant: variants.SUCCESS, icon: icons.OK, text: 'Identifiers saved successfully' });
		}, 500);
	};

	const items = getSchemaComboboxItems(userSchema);
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
			) : userSchema == null ? (
				<div className='identifiers__no-schema'>
					<div className='identifiers__no-schema-title'>Connect a data warehouse</div>
					<div className='identifiers__no-schema-description'>
						The anonymous identifiers are chosen from among the data warehouse columns. For this reason,
						before you can set the anonymous identifiers you must connect a data warehouse.
					</div>
					<SlButton
						variant='primary'
						className='identifiers__connect-warehouse-button'
						onClick={onConnectDataWarehouse}
					>
						<SlIcon name='database' slot='prefix' />
						Connect a data warehouse...
					</SlButton>
				</div>
			) : (
				<div>
					<Section
						title='Identifiers'
						description='Define the identifiers used to resolve the identity of the users'
					>
						{identifiers.map((identifier, i) => {
							const position = i + 1;
							return (
								<div key={position} className='identifiers__identifier'>
									<div className='identifiers__identifier-position'>{position}</div>
									<ComboBoxInput
										className='identifiers__identifier-input'
										comboBoxListRef={identifiersListRef}
										name={String(position)}
										value={identifier}
										onInput={onUpdateIdentifier}
										size='small'
									/>
									<SlDropdown>
										<SlButton size='small' className='identifiers-identifier__menu' slot='trigger'>
											<SlIcon slot='prefix' name='three-dots'></SlIcon>
										</SlButton>
										<SlMenu>
											<SlMenuItem
												onClick={() => moveIdentifierUp(position)}
												disabled={position === 1}
											>
												<SlIcon slot='prefix' name='arrow-up-circle' />
												Move up
											</SlMenuItem>
											<SlMenuItem
												onClick={() => moveIdentifierDown(position)}
												disabled={position === identifiers.length}
											>
												<SlIcon slot='prefix' name='arrow-down-circle' />
												Move down
											</SlMenuItem>
											<SlMenuItem
												className='identifiers-mapping__remove'
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
						<ComboBoxList
							ref={identifiersListRef}
							items={identifiersComboboxItems}
							onSelect={onSelectIdentifier}
						/>
					</Section>
					<Section
						title='Anonymous Identifiers'
						description='Define the identifiers used to resolve the identity of anonymous users'
					>
						<IdentifiersMapping
							mapping={anonymousIdentifiers!}
							setMapping={setAnonymousIdentifiers}
							inputSchema={eventSchema!}
							outputSchema={userSchema!}
						/>
						<SlButton
							className='identifiers__save-button'
							onClick={onSave}
							variant='primary'
							loading={isSaving}
						>
							Save
						</SlButton>
					</Section>
				</div>
			)}
		</div>
	);
};

export default Identifiers;
