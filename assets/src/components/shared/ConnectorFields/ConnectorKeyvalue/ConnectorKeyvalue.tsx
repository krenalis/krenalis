import React, { useState, useEffect, ReactNode } from 'react';
import './ConnectorKeyvalue.css';
import ConnectorField from '../ConnectorField';
import { KeyContext } from '../../../../context/KeyContext';
import { ValueContext } from '../../../../context/ValueContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import ConnectorFieldInterface from '../../../../types/external/ui';

type KeyValueValue = '' | Record<string, any>;

interface KeyValueRow {
	id: number;
	key: string;
	value: any;
}

const initRows = (value: KeyValueValue): KeyValueRow[] => {
	const keys = Object.keys(value);
	if (keys.length > 0) {
		const rows: any[] = [];
		let counter = 1;
		for (const key of keys) {
			rows.push({ id: counter, key: key, value: value[key] });
			counter++;
		}
		return rows;
	} else {
		return [{ id: 1, key: '', value: '' }];
	}
};

interface ConnectorKeyValueProps {
	name: string;
	label: string;
	keyComponent: ConnectorFieldInterface;
	keyLabel: string;
	valueComponent: ConnectorFieldInterface;
	valueLabel: string;
	error: string;
	val: KeyValueValue;
	onChange: (...args: any) => void;
}

const ConnectorKeyValue = ({
	name,
	label,
	keyComponent,
	keyLabel,
	valueComponent,
	valueLabel,
	error,
	val,
	onChange,
}: ConnectorKeyValueProps) => {
	const [rows, setRows] = useState<KeyValueRow[]>(initRows(val));

	useEffect(() => {
		setRows(initRows(val));
	}, [val]);

	const formatRows = (rows: KeyValueRow[]): KeyValueValue => {
		const formatted = {};
		for (const row of rows) {
			formatted[row.key] = row.value;
		}
		return formatted;
	};

	const onAddRowClick = () => {
		const rws = [...rows, { id: rows[rows.length - 1].id + 1, key: '', value: '' }];
		setRows(rws);
	};

	const onRemoveRowClick = (id: number) => {
		const rws = [...rows];
		const filtered = rws.filter((r) => r.id !== id);
		onChange(name, formatRows(filtered));
	};

	const onKeyChange = (_, key, e) => {
		const id = Number(e.currentTarget.closest('.keyValueRow').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) {
				return { ...r, key: key };
			}
			return r;
		});
		onChange(name, formatRows(updated));
	};

	const onValueChange = (_, value, e) => {
		const id = Number(e.currentTarget.closest('.keyValueRow').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) {
				return { ...r, value: value };
			}
			return r;
		});
		onChange(name, formatRows(updated));
	};

	const keyValueRows: ReactNode[] = [];
	for (const r of rows) {
		keyValueRows.push(
			<div className='keyValueRow' data-id={r.id} key={r.id}>
				<KeyContext.Provider value={{ value: r.key, onChange: onKeyChange }}>
					<div className='key cell'>
						<ConnectorField field={keyComponent} />
					</div>
				</KeyContext.Provider>
				<ValueContext.Provider value={{ value: r.value, onChange: onValueChange }}>
					<div className='value cell'>
						<ConnectorField field={valueComponent} />
					</div>
				</ValueContext.Provider>
				{r.id !== 1 && <SlIcon className='removeRow' name='trash3' onClick={() => onRemoveRowClick(r.id)} />}
			</div>,
		);
	}

	return (
		<div className='connectorKeyValue'>
			<div className='label'>{label}</div>
			<div className='keyValueGrid'>
				<div className='keyValueRow labels'>
					<div className='keyLabel'>{keyLabel}</div>
					<div className='valueLabel'>{valueLabel}</div>
				</div>
				{keyValueRows}
			</div>
			<SlIcon className='addRow' onClick={onAddRowClick} name='plus-circle' />
			{error !== '' && <div className='error'>{error}</div>}
		</div>
	);
};

export default ConnectorKeyValue;
