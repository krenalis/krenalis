import React, { useState, useEffect, ReactNode } from 'react';
import './ConnectorKeyvalue.css';
import ConnectorField from '../ConnectorField';
import { KeyContext } from '../../../../context/KeyContext';
import { ValueContext } from '../../../../context/ValueContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import ConnectorFieldInterface from '../../../../types/external/ui';

interface ConnectorKeyValueProps {
	name: string;
	label: string;
	keyComponent: ConnectorFieldInterface;
	keyLabel: string;
	valueComponent: ConnectorFieldInterface;
	valueLabel: string;
	error: string;
	val: any;
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
	const initRows = (val: any) => {
		let keyValues: any[] = [];
		if (val !== '') keyValues = Object.entries(val);
		const rws: any[] = [];
		if (keyValues.length > 0) {
			let counter = 1;
			for (const [key, value] of keyValues) {
				rws.push({ id: counter, key: key, value: value });
				counter = counter + 1;
			}
		} else {
			rws.push({ id: 1, key: '', value: '' });
		}
		return rws;
	};

	const [rows, setRows] = useState(initRows(val));

	useEffect(() => {
		setRows(initRows(val));
	}, [val]);

	const formatRows = (rws) => {
		const formatted = {};
		for (const r of rws) formatted[r.key] = r.value;
		return formatted;
	};

	const onAddRowClick = () => {
		const rws = [...rows, { id: rows[rows.length - 1].id + 1, key: '', value: '' }];
		setRows(rws);
	};

	const onRemoveRowClick = (id) => {
		const rws = [...rows];
		const filtered = rws.filter((r) => r.id !== id);
		setRows(filtered);
		onChange(name, formatRows(filtered));
	};

	const onKeyChange = async (n, key, e) => {
		const id = Number(e.currentTarget.closest('.row').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) return { ...r, key: key };
			return r;
		});
		setRows(updated);
		onChange(name, formatRows(updated));
	};

	const onValueChange = (n, value, e) => {
		const id = Number(e.currentTarget.closest('.row').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) return { ...r, value: value };
			return r;
		});
		setRows(updated);
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
			</div>
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
