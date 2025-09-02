import React, { useContext, useRef, useEffect, useMemo } from 'react';
import Section from '../../base/Section/Section';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { CONFIRM_ANIMATION_DURATION, ERROR_ANIMATION_DURATION } from './Action.constants';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { flattenSchema, propertyTypesAreEqual } from '../../../lib/core/action';
import { Popover } from '../../base/Popover/Popover';
import { getTableKeyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import { Combobox } from '../../base/Combobox/Combobox';
import { TableSchemaResponse } from '../../../lib/api/types/responses';

const ActionTable = () => {
	const { handleError, api } = useContext(AppContext);
	const {
		connection,
		action,
		setAction,
		actionType,
		setActionType,
		transformationSectionRef,
		setIsTableChanged,
		isTransformationDisabled,
		isTransformationHidden,
		setIssues,
	} = useContext(ActionContext);

	const tableConfirmationButtonRef = useRef<any>();
	const tableKeySectionRef = useRef<any>();
	const tableKeyRef = useRef<any>();
	const tableRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	useEffect(() => {
		tableRef.current = {
			lastConfirmation: action.tableName!,
			lastUpdate: action.tableName!,
		};
	}, []);

	const tableKeyComboboxItems = useMemo(() => {
		return getTableKeyComboboxItems(actionType.outputSchema);
	}, [actionType]);

	const onUpdateTableName = (e) => {
		const value = e.target.value;
		tableRef.current.lastUpdate = value;
		if (
			tableRef.current.lastUpdate !== tableRef.current.lastConfirmation &&
			tableRef.current.lastConfirmation !== ''
		) {
			setIsTableChanged(true);
		} else {
			setIsTableChanged(false);
		}
		const a = { ...action };
		a.tableName = value;
		setAction(a);
	};

	const onTableKeyPropertyUpdate = (_: string, value: string) => {
		const a = { ...action };
		a.tableKey = value;
		setAction(a);
	};

	const onTableKeyPropertySelect = (_: string, value: string) => {
		const a = { ...action };
		a.tableKey = value;
		setAction(a);
	};

	const onConfirmTable = async () => {
		setIssues([]);
		tableConfirmationButtonRef.current!.load();
		let res: TableSchemaResponse;
		try {
			res = await api.workspaces.connections.tableSchema(connection.id, action.tableName);
		} catch (err) {
			tableConfirmationButtonRef.current!.stop();
			handleError(err);
			return;
		}
		if (res.schema == null) {
			tableConfirmationButtonRef.current!.error("This table doesn't have any compatible column");
			setTimeout(() => {
				setIssues(res.issues);
				const actionTyp = { ...actionType };
				actionTyp.outputSchema = null;
				setActionType(actionTyp);
			}, ERROR_ANIMATION_DURATION);
		} else {
			tableConfirmationButtonRef.current!.confirm();
			setTimeout(() => {
				setIssues(res.issues);
				tableRef.current.lastConfirmation = action.tableName;
				setIsTableChanged(false);
				const actionTyp = { ...actionType };
				actionTyp.outputSchema = res.schema;
				setActionType(actionTyp);
				const a = { ...action };
				const mapping = flattenSchema(res.schema, true);
				if (a.transformation.mapping != null) {
					// Keep the old mapping (if the column stil exists
					// in the new out schema and the type is the same).
					for (const path in mapping) {
						const existedInOldSchema = a.transformation.mapping[path] != null;
						if (!existedInOldSchema) {
							continue;
						}
						const newType = mapping[path].full.type;
						const oldType = a.transformation.mapping[path].full.type;
						if (!propertyTypesAreEqual(newType, oldType)) {
							continue;
						}
						mapping[path].value = a.transformation.mapping[path].value;
						mapping[path].error = a.transformation.mapping[path].error;
					}
				}
				a.transformation.mapping = mapping;
				setAction(a);
				setTimeout(() => {
					let scrollSection = transformationSectionRef.current;
					if (tableKeyRef.current != null) {
						scrollSection = tableKeySectionRef.current;
					}
					const top = scrollSection.getBoundingClientRect().top;
					scrollSection.closest('.fullscreen').scrollBy({
						top: top - 130,
						left: 0,
						behavior: 'smooth',
					});
				}, 100);
			}, CONFIRM_ANIMATION_DURATION);
		}
	};

	return (
		<>
			<Section
				title='Table name'
				description='An existing table on the database where users will be exported'
				padded={true}
				annotated={true}
			>
				<div className='action__table'>
					<SlInput value={action.tableName} onSlInput={onUpdateTableName} />
					<FeedbackButton
						ref={tableConfirmationButtonRef}
						variant='success'
						size='small'
						onClick={onConfirmTable}
						animationDuration={CONFIRM_ANIMATION_DURATION}
						disabled={action.tableName === ''}
					>
						Confirm
					</FeedbackButton>
					<Popover
						isOpen={isTransformationDisabled}
						content='Confirm when you have finished editing the table name.'
					/>
				</div>
			</Section>
			{actionType.target === 'User' && !isTransformationHidden && (
				<Section
					title='Table key'
					description='The property of the table that is used as key in export queries'
					padded={true}
					annotated={true}
					ref={tableKeySectionRef}
					className={`action__table-key-section${isTransformationDisabled ? ' action__table-key-section--disabled' : ''}`}
				>
					<div className='action__table-key-property' ref={tableKeyRef}>
						<Combobox
							value={action.tableKey}
							onInput={onTableKeyPropertyUpdate}
							name='table-key'
							items={tableKeyComboboxItems}
							onSelect={onTableKeyPropertySelect}
							disabled={isTransformationDisabled}
							isExpression={false}
						/>
					</div>
				</Section>
			)}
		</>
	);
};

export default ActionTable;
