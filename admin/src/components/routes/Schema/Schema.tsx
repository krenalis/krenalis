import React, { useState, useEffect, useContext } from 'react';
import './Schema.css';
import Grid from '../../shared/Grid/Grid';
import Toolbar from '../../layout/Toolbar/Toolbar';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { UnprocessableError } from '../../../lib/api/errors';
import { ArrayType, TextType, Property, ObjectType } from '../../../types/external/types';
import { GridRow, NestedGridRows } from '../../../types/componentTypes/Grid.types';

const Schema = () => {
	const [properties, setProperties] = useState<Property[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, redirect, showError, showStatus, setTitle, selectedWorkspace, warehouse } = useContext(AppContext);

	setTitle('Schema');

	useEffect(() => {
		const fetchSchema = async () => {
			let schema;
			try {
				schema = await api.workspaces.userSchema();
			} catch (err) {
				showError(err);
				return;
			}
			setProperties(schema.properties);
			setTimeout(() => setIsLoading(false), 500);
		};
		if (warehouse == null) {
			redirect('settings');
			showError('Please connect to a data warehouse before proceeding');
			return;
		}
		fetchSchema();
	}, [selectedWorkspace]);

	const onReloadSchemas = async () => {
		setIsLoading(true);
		try {
			await api.workspaces.reloadSchemas();
		} catch (err) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'NotConnected':
						showStatus(statuses.warehouseNotConnected);
						break;
					case 'DataWarehouseFailed':
						showStatus(statuses.dataWarehouseFailed);
						break;
					default:
						break;
				}
				return;
			}
			setProperties([]);
			showError(err);
			return;
		}
		let schema;
		try {
			schema = await api.workspaces.userSchema();
		} catch (err) {
			setProperties([]);
			showError(err);
			return;
		}
		localStorage.removeItem('usersProperties');
		setProperties(schema.properties);
		setTimeout(() => {
			setIsLoading(false);
			showStatus(statuses.schemasReloaded);
		}, 300);
	};

	const getPropertiesRows = (properties: Property[]) => {
		const getNestedRows = (p: Property): NestedGridRows => {
			let nestedRows: GridRow[] = [{ cells: [p.name, p.type.name] }];
			const typ = p.type as ObjectType;
			if (typ.properties == null) {
				console.warn('Schema contains an object property without any properties');
				return [];
			}
			for (const pr of typ.properties) {
				if (pr.type.name === 'Object') {
					const nr = getNestedRows(pr);
					nestedRows.push(nr);
					continue;
				} else {
					let name: string = pr.type.name;
					if (name === 'Array') {
						const typ = p.type as ArrayType;
						name = 'Array(' + typ.itemType?.name + ')';
					}
					if ('values' in pr.type) {
						const typ = p.type as TextType;
						name += ' (' + typ.values?.map((e) => '"' + e + '"').join(', ') + ')';
					}
					nestedRows.push({ cells: [pr.name, name] });
				}
			}
			return nestedRows;
		};
		const rows: GridRow[] = [];
		for (const p of properties) {
			if (p.type.name === 'Object') {
				const nestedRows = getNestedRows(p);
				if (nestedRows.length === 0) continue;
				rows.push(nestedRows);
				continue;
			} else {
				let name: string = p.type.name;
				if (name === 'Array') {
					const typ = p.type as ArrayType;
					name = 'Array(' + typ.itemType?.name + ')';
				}
				if ('values' in p.type) {
					const typ = p.type as TextType;
					name += ' (' + typ.values?.map((e: string) => '"' + e + '"').join(', ') + ')';
				}
				const row = { cells: [p.name, name] };
				rows.push(row);
			}
		}
		return rows;
	};

	const columns = [{ name: 'name' }, { name: 'type' }];
	const rows = getPropertiesRows(properties);

	return (
		<div className='schema'>
			<Toolbar>
				<SlButton className='reloadSchemas' variant='default' onClick={onReloadSchemas}>
					<SlIcon name='arrow-clockwise' slot='prefix'></SlIcon>
					Reload Schemas
				</SlButton>
			</Toolbar>
			<div className='routeContent schema'>
				<Grid columns={columns} rows={rows} isLoading={isLoading}></Grid>
			</div>
		</div>
	);
};

export default Schema;
