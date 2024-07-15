import React, { useContext, useRef, useEffect, useMemo } from 'react';
import Section from '../../base/Section/Section';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { ObjectType } from '../../../lib/api/types/types';
import { flattenSchema } from '../../../lib/core/action';
import { Popover } from '../../base/Popover/Popover';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
import { getTableKeyComboboxItems } from '../../helpers/getSchemaComboBoxItems';

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
	} = useContext(ActionContext);

	const tableConfirmationButtonRef = useRef<any>();
	const tableKeySectionRef = useRef<any>();
	const tableKeyListRef = useRef<any>();
	const tableRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	useEffect(() => {
		tableRef.current = {
			lastConfirmation: action.Table!,
			lastUpdate: action.Table!,
		};
	}, []);

	const tableKeyComboboxItems = useMemo(() => {
		return getTableKeyComboboxItems(actionType.OutputSchema);
	}, [actionType]);

	const onUpdateTable = (e) => {
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
		a.Table = value;
		setAction(a);
	};

	const onTableKeyPropertyUpdate = (e) => {
		const value = e.target.value;
		const a = { ...action };
		a.TableKeyProperty = value;
		setAction(a);
	};

	const onTableKeyPropertySelect = (_, value) => {
		const a = { ...action };
		a.TableKeyProperty = value;
		setAction(a);
	};

	const onConfirmTable = async () => {
		tableConfirmationButtonRef.current!.load();
		let schema: ObjectType;
		try {
			schema = await api.workspaces.connections.tableSchema(connection.id, action.Table);
		} catch (err) {
			tableConfirmationButtonRef.current!.stop();
			handleError(err);
			return;
		}
		tableConfirmationButtonRef.current!.confirm();
		setTimeout(() => {
			tableRef.current.lastConfirmation = action.Table;
			setIsTableChanged(false);
			const actionTyp = { ...actionType };
			actionTyp.OutputSchema = schema;
			setActionType(actionTyp);
			const a = { ...action };
			a.Transformation.Mapping = flattenSchema(schema);
			setAction(a);
			setTimeout(() => {
				let scrollSection = transformationSectionRef.current;
				if (tableKeyListRef.current != null) {
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
	};

	return (
		<>
			<Section title='Table' description='The name of the table of the database'>
				<div className='action__table'>
					<SlInput value={action.Table} onSlInput={onUpdateTable} />
					<FeedbackButton
						ref={tableConfirmationButtonRef}
						variant='success'
						size='small'
						onClick={onConfirmTable}
						animationDuration={CONFIRM_ANIMATION_DURATION}
					>
						Confirm
					</FeedbackButton>
					<Popover
						isOpen={isTransformationDisabled}
						content='Confirm when you have finished editing the table name.'
					/>
				</div>
			</Section>
			{actionType.Target === 'Users' && !isTransformationHidden && (
				<Section
					title='Table key'
					description='The property of the table that is used as key in export queries'
					ref={tableKeySectionRef}
					className={`action__table-key-section${isTransformationDisabled ? ' action__table-key-section--disabled' : ''}`}
				>
					<div className='action__table-key-property'>
						<ComboBoxInput
							value={action.TableKeyProperty}
							comboBoxListRef={tableKeyListRef}
							onInput={onTableKeyPropertyUpdate}
							disabled={isTransformationDisabled}
						/>
						<ComboBoxList
							ref={tableKeyListRef}
							items={tableKeyComboboxItems}
							onSelect={onTableKeyPropertySelect}
						/>
					</div>
				</Section>
			)}
		</>
	);
};

export default ActionTable;
