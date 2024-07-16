import React, { useState, useEffect, useMemo, useRef, useContext } from 'react';
import './PropertyDialog.css';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import {
	ArrayType,
	DecimalType,
	FloatBitSize,
	FloatType,
	IntBitSize,
	IntType,
	MapType,
	TypeName,
	UintType,
} from '../../../lib/api/types/types';
import { PropertyToEdit } from './useSchemaEdit';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTextarea from '@shoelace-style/shoelace/dist/react/textarea/index.js';
import AppContext from '../../../context/AppContext';
import TransformedConnection from '../../../lib/core/connection';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { PrimarySources } from '../../../lib/api/types/workspace';

const TYPE_NAMES: TypeName[] = [
	'Boolean',
	'Int',
	'Uint',
	'Float',
	'Decimal',
	'DateTime',
	'Date',
	'Time',
	'Year',
	'UUID',
	'JSON',
	'Inet',
	'Text',
	'Array',
	'Object',
	'Map',
];

const INT_BITSIZES: string[] = ['8', '16', '24', '32', '64'];
const FLOAT_BITSIZES: string[] = ['32', '64'];
const MAX_DECIMAL_PRECISION: number = 76;
const MAX_DECIMAL_SCALE: number = 37;

interface PropertyDialogProps {
	propertyToEdit: PropertyToEdit | null;
	setPropertyToEdit: React.Dispatch<React.SetStateAction<PropertyToEdit | null>>;
	primarySources: PrimarySources;
	onAddProperty: (property: PropertyToEdit, primarySource: number | null) => void;
	onEditProperty: (property: PropertyToEdit, primarySource: number | null) => void;
}

const PropertyDialog = ({
	propertyToEdit,
	setPropertyToEdit,
	primarySources,
	onAddProperty,
	onEditProperty,
}: PropertyDialogProps) => {
	const [property, setProperty] = useState<PropertyToEdit>();
	const [primarySource, setPrimarySource] = useState<number | null>(null);
	const [nameError, setNameError] = useState<string>('');
	const [typeError, setTypeError] = useState<string>('');

	const { connections } = useContext(AppContext);

	const nameInputRef = useRef<any>();
	const bitSizeSelectRef = useRef<any>();
	const precisionInputRef = useRef<any>();
	const elementTypeSelectRef = useRef<any>();
	const valueTypeSelectRef = useRef<any>();

	const isEditing = useMemo(() => {
		if (propertyToEdit == null) {
			return false;
		}
		return propertyToEdit.key != null;
	}, [propertyToEdit]);

	const sourceConnections = useMemo(() => {
		const sources: TransformedConnection[] = [];
		for (const c of connections) {
			if (c.role === 'Source') {
				sources.push(c);
			}
		}
		return sources;
	}, [connections]);

	useEffect(() => {
		if (propertyToEdit == null) {
			return;
		}
		setNameError('');
		setTypeError('');
		setProperty(propertyToEdit);
		setPrimarySource(primarySources[propertyToEdit.key]);
	}, [propertyToEdit]);

	const onInputName = (e) => {
		const name = e.target.value;
		if (name === '') {
			setNameError('Name cannot be empty');
		} else {
			setNameError('');
		}
		const p = { ...property };
		p.name = name;
		setProperty(p);
	};

	const onChangeType = (e) => {
		const p = { ...property };
		const typeName = e.target.value as TypeName;
		const typ: any = { name: typeName };
		if (typeName === 'Int' || typeName === 'Uint') {
			typ.bitSize = 32;
			setTimeout(() => bitSizeSelectRef.current?.focus(), 50);
		}
		if (typeName === 'Float') {
			typ.bitSize = 64;
			setTimeout(() => bitSizeSelectRef.current?.focus(), 50);
		}
		if (typeName === 'Decimal') {
			typ.scale = '';
			typ.precision = '';
			setTimeout(() => precisionInputRef.current?.focus(), 50);
		}
		if (typeName === 'Array') {
			typ.elementType = { name: '' };
			setTimeout(() => elementTypeSelectRef.current?.focus(), 50);
		}
		if (typeName === 'Map') {
			typ.valueType = { name: '' };
			setTimeout(() => valueTypeSelectRef.current?.focus(), 50);
		}
		p.nullable = false;
		p.type = typ;
		setProperty(p);
		if (typeError !== '') {
			setTypeError('');
		}
	};

	const onChangeBitSize = (e) => {
		const p = { ...property };
		if (p.type.name === 'Array') {
			const typ = p.type as ArrayType;
			const elementTyp = typ.elementType as IntType | UintType | FloatType;
			elementTyp.bitSize = Number(e.target.value) as IntBitSize | FloatBitSize;
			typ.elementType = elementTyp;
			p.type = typ;
		} else {
			const typ = p.type as IntType | UintType | FloatType;
			typ.bitSize = Number(e.target.value) as IntBitSize | FloatBitSize;
			p.type = typ;
		}
		setProperty(p);
	};

	const onInputPrecision = (e) => {
		const p = { ...property };
		if (p.type.name === 'Array') {
			const typ = p.type as ArrayType;
			const elementTyp = typ.elementType as DecimalType;
			elementTyp.precision = Number(e.target.value);
			typ.elementType = elementTyp;
			p.type = typ;
		} else {
			const typ = p.type as DecimalType;
			typ.precision = Number(e.target.value);
			p.type = typ;
		}
		setProperty(p);
		if (typeError !== '') {
			setTypeError('');
		}
	};

	const onInputScale = (e) => {
		const p = { ...property };
		if (p.type.name === 'Array') {
			const typ = p.type as ArrayType;
			const elementTyp = typ.elementType as DecimalType;
			elementTyp.scale = Number(e.target.value);
			typ.elementType = elementTyp;
			p.type = typ;
		} else {
			const typ = p.type as DecimalType;
			typ.scale = Number(e.target.value);
			p.type = typ;
		}
		setProperty(p);
		if (typeError !== '') {
			setTypeError('');
		}
	};

	const onChangeElementType = (e) => {
		const p = { ...property };
		const typeName = e.target.value as TypeName;
		const typ: any = { name: typeName };
		if (typeName === 'Int' || typeName === 'Uint') {
			typ.bitSize = 32;
			setTimeout(() => bitSizeSelectRef.current?.focus(), 50);
		}
		if (typeName === 'Float') {
			typ.bitSize = 64;
			setTimeout(() => bitSizeSelectRef.current?.focus(), 50);
		}
		if (typeName === 'Decimal') {
			typ.scale = '';
			typ.precision = '';
			setTimeout(() => precisionInputRef.current?.focus(), 50);
		}
		(p.type as ArrayType).elementType = typ;
		setProperty(p);
		if (typeError !== '') {
			setTypeError('');
		}
	};

	const onChangeValueType = (e) => {
		const p = { ...property };
		const typ = p.type as MapType;
		typ.valueType.name = e.target.value;
		p.type = typ;
		setProperty(p);
	};

	const onInputLabel = (e) => {
		const p = { ...property };
		p.label = e.target.value;
		setProperty(p);
	};

	const onChangeNote = (e) => {
		const p = { ...property };
		p.note = e.target.value;
		setProperty(p);
	};

	const onChangePrimarySource = (e) => {
		const v = e.target.value;
		if (v === 'none') {
			setPrimarySource(null);
		} else {
			setPrimarySource(Number(e.target.value));
		}
	};

	const onHide = (e) => {
		if (e.target.tagName === 'SL-DIALOG') {
			setPropertyToEdit(null);
		}
	};

	const onShow = (e) => {
		if (e.target.tagName === 'SL-DIALOG') {
			if (nameInputRef.current) {
				nameInputRef.current.focus();
			}
		}
	};

	const onSave = () => {
		if (property.name === '') {
			setNameError('Name cannot be empty');
			return;
		}
		if (property.type === null) {
			setTypeError('Type cannot be empty');
			return;
		}
		if (
			property.type.name === 'Decimal' ||
			(property.type.name === 'Array' && (property.type as ArrayType).elementType.name === 'Decimal')
		) {
			const typ: DecimalType = (
				property.type.name === 'Array' ? property.type.elementType : property.type
			) as DecimalType;
			const err = checkDecimalType(typ);
			if (err) {
				setTypeError(err);
				return;
			}
		}
		if (isEditing) {
			try {
				onEditProperty(property, primarySource);
			} catch (err) {
				setNameError(err.message);
				return;
			}
		} else {
			try {
				onAddProperty(property, primarySource);
			} catch (err) {
				setNameError(err.message);
				return;
			}
		}
		setPropertyToEdit(null);
	};

	let bitSizeSection = null;
	if (property != null && property.type != null) {
		const isArray = property.type.name === 'Array';
		const hasBitSize =
			hasBitSizeConstraint(property.type.name) ||
			(isArray && hasBitSizeConstraint((property.type as ArrayType).elementType.name));
		if (hasBitSize) {
			const typ: any = isArray ? (property.type as ArrayType).elementType : property.type;
			bitSizeSection = (
				<SlSelect
					className='property-dialog__bitsize'
					ref={bitSizeSelectRef}
					size='small'
					label='Bit size'
					value={String(typ.bitSize)}
					onSlChange={onChangeBitSize}
				>
					{typ.name === 'Int' || typ.name === 'Uint'
						? INT_BITSIZES.map((s) => (
								<SlOption key={s} value={s}>
									{s}
								</SlOption>
							))
						: FLOAT_BITSIZES.map((s) => (
								<SlOption key={s} value={s}>
									{s}
								</SlOption>
							))}
				</SlSelect>
			);
		}
	}

	let precisionSection = null;
	let scaleSection = null;
	if (property != null && property.type != null) {
		const isArray = property.type.name === 'Array';
		const hasDecimal =
			property.type.name === 'Decimal' ||
			(isArray && (property.type as ArrayType).elementType.name === 'Decimal');
		if (hasDecimal) {
			const typ: any = isArray ? (property.type as ArrayType).elementType : property.type;
			precisionSection = (
				<SlInput
					className='property-dialog__precision'
					ref={precisionInputRef}
					size='small'
					label='Precision'
					value={String(typ.precision)}
					type='number'
					max={MAX_DECIMAL_PRECISION}
					maxlength={2}
					onSlInput={onInputPrecision}
				/>
			);
			scaleSection = (
				<SlInput
					className='property-dialog__scale'
					size='small'
					label='Scale'
					value={String(typ.scale)}
					type='number'
					max={MAX_DECIMAL_SCALE}
					maxlength={2}
					onSlInput={onInputScale}
				/>
			);
		}
	}

	return (
		<SlDialog
			className='property-dialog'
			open={propertyToEdit != null}
			label={isEditing ? `Edit "${propertyToEdit?.name}"` : 'Add a new property'}
			onSlAfterHide={onHide}
			onSlAfterShow={onShow}
		>
			{property != null && (
				<>
					<div className='property-dialog__control'>
						<SlInput
							ref={nameInputRef}
							size='small'
							value={property.name}
							label='Name'
							placeholder='first_name'
							onSlInput={onInputName}
						/>
						{nameError !== '' && (
							<div className='property-dialog__control-error'>
								<SlIcon name='exclamation-circle' />
								{nameError}
							</div>
						)}
					</div>
					{!isEditing || property.isEditable ? (
						<div className='property-dialog__control'>
							<div className='property-dialog__control-type'>
								<SlSelect
									className='property-dialog__type-select'
									size='small'
									label='Type'
									value={property.type?.name}
									onSlChange={onChangeType}
									hoist={true}
								>
									{TYPE_NAMES.map((t) => (
										<SlOption key={t} value={t}>
											{t}
										</SlOption>
									))}
								</SlSelect>
								{property.type?.name === 'Array' && (
									<SlSelect
										className='property-dialog__elementtype'
										ref={elementTypeSelectRef}
										size='small'
										label='Element type'
										value={property.type?.elementType?.name}
										onSlChange={onChangeElementType}
										hoist={true}
									>
										{TYPE_NAMES.map((t) => {
											if (t !== 'Array' && t !== 'Map' && t !== 'Object') {
												return (
													<SlOption key={t} value={t}>
														{t}
													</SlOption>
												);
											}
										})}
									</SlSelect>
								)}
								{bitSizeSection}
								{precisionSection}
								{scaleSection}
								{property.type?.name === 'Map' && (
									<SlSelect
										className='property-dialog__valuetype'
										ref={valueTypeSelectRef}
										size='small'
										label='Value type'
										value={property.type?.valueType?.name}
										onSlChange={onChangeValueType}
										hoist={true}
									>
										{TYPE_NAMES.map((t) => {
											if (t !== 'Array' && t !== 'Map' && t !== 'Object') {
												return (
													<SlOption key={t} value={t}>
														{t}
													</SlOption>
												);
											}
										})}
									</SlSelect>
								)}
							</div>
							{typeError !== '' && (
								<div className='property-dialog__control-error'>
									<SlIcon name='exclamation-circle' />
									{typeError}
								</div>
							)}
						</div>
					) : (
						<div className='property-dialog__type'>
							<div className='property-dialog__type-label'>Type</div>
							<div className='property-dialog__type-value'>{property.type?.name}</div>
						</div>
					)}
					<SlInput
						className='property-dialog__control'
						size='small'
						value={property.label}
						label='Label'
						placeholder='First name'
						onSlInput={onInputLabel}
					/>
					<SlTextarea
						className='property-dialog__control'
						size='small'
						value={property.note}
						label='Note'
						onSlChange={onChangeNote}
					/>
					{property.type?.name !== 'Object' &&
						property.type?.name !== 'Array' &&
						(sourceConnections.length === 0 ? (
							<div className='property-dialog__no-primary-source'>
								<div className='property-dialog__no-primary-source-label'>Primary Source</div>
								Currently there is no source connection
							</div>
						) : (
							<SlSelect
								className='property-dialog__primary-source'
								size='small'
								value={primarySource == null ? 'none' : String(primarySource)}
								label='Primary Source'
								onSlChange={onChangePrimarySource}
							>
								<div slot='prefix'>
									{primarySource &&
										getConnectorLogo(
											sourceConnections.find((c) => c.id === primarySource).connector.icon,
										)}
								</div>
								<SlOption value='none'>None</SlOption>
								{sourceConnections.map((c) => (
									<SlOption key={c.id} value={String(c.id)}>
										<div slot='prefix'>{getConnectorLogo(c.connector.icon)}</div>
										{c.name}
									</SlOption>
								))}
							</SlSelect>
						))}
					<div className='property-dialog__buttons'>
						<SlButton size='small' variant='neutral' onClick={() => setPropertyToEdit(null)}>
							Cancel
						</SlButton>
						<SlButton
							className='property-dialog__save'
							size='small'
							variant='primary'
							onClick={onSave}
							disabled={nameError !== '' || typeError !== ''}
						>
							{isEditing ? 'Save' : 'Add'}
						</SlButton>
					</div>
				</>
			)}
		</SlDialog>
	);
};

const hasBitSizeConstraint = (name: string) => {
	return name === 'Int' || name === 'Uint' || name === 'Float';
};

const checkDecimalType = (type: DecimalType) => {
	if (type.precision < 1 || type.precision > MAX_DECIMAL_PRECISION) {
		return `Precision must be in range [1, ${MAX_DECIMAL_PRECISION}]`;
	}
	if (type.scale < 0 || type.scale > MAX_DECIMAL_SCALE) {
		return `Scale must be in range [0, ${MAX_DECIMAL_SCALE}]`;
	}
};

export { PropertyDialog };
