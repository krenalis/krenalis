import React, { useState, useEffect, useContext, useLayoutEffect } from 'react';
import './Schema.css';
import Grid from '../../shared/Grid/Grid';
import AppContext from '../../../context/AppContext';
import { ArrayType, TextType, Property, ObjectType, IntType, UintType, FloatType } from '../../../types/external/types';
import { GridRow, NestedGridRows } from '../../../types/componentTypes/Grid.types';

const Schema = () => {
	const [properties, setProperties] = useState<Property[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, redirect, showError, setTitle, selectedWorkspace, warehouse } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Schema');
	}, []);

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
					if (name === 'Int' || name === 'Uint' || name === 'Float') {
						const typ = p.type as IntType | UintType | FloatType;
						name += `(${typ.bitSize})`;
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
				if (name === 'Int' || name === 'Uint' || name === 'Float') {
					const typ = p.type as IntType | UintType | FloatType;
					name += `(${typ.bitSize})`;
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
			<div className='routeContent schema'>
				<Grid columns={columns} rows={rows} isLoading={isLoading}></Grid>
			</div>
		</div>
	);
};

export default Schema;
