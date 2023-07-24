import { useState, useEffect, useContext } from 'react';
import './Schema.css';
import Grid from '../../shared/Grid/Grid';
import Toolbar from '../../layout/Toolbar/Toolbar';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { UnprocessableError } from '../../../lib/api/errors';

const Schema = () => {
	const [properties, setProperties] = useState([]);
	const [isLoading, setIsLoading] = useState(true);

	const { api, showError, showStatus, setTitle } = useContext(AppContext);

	setTitle('Schema');

	useEffect(() => {
		const fetchSchema = async () => {
			const [schema, err] = await api.workspace.userSchema();
			if (err) {
				showError(err);
				return;
			}
			setProperties(schema.properties);
			setTimeout(() => setIsLoading(false), 500);
		};
		fetchSchema();
	}, []);

	const onReloadSchemas = async () => {
		let err;
		setIsLoading(true);
		[, err] = await api.workspace.reloadSchemas();
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'NotConnected':
						showStatus(statuses.warehouseNotConnected);
						break;
					case 'WarehouseFailed':
						showStatus(statuses.warehouseConnectionFailed);
						break;
					case 'InvalidSchemaTable':
						showStatus(statuses.invalidSchemaTable);
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
		[schema, err] = await api.workspace.userSchema();
		if (err) {
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

	const getPropertiesRows = (properties) => {
		const getNestedRows = (p) => {
			const nestedRows = [{ cells: [p.name, p.type.name] }];
			for (const pr of p.type.properties) {
				if (pr.type.name === 'Object') {
					const nr = getNestedRows(pr);
					nestedRows.push(nr);
					continue;
				} else {
					let name = pr.type.name;
					if (name === 'Array') {
						name = 'Array (of ' + pr.type.itemType.name + ' elements)';
					}
					if ('enum' in pr.type) {
						name += ' (enum with values: ' + pr.type.enum.join(', ') + ')';
					}
					nestedRows.push({ cells: [pr.name, name] });
				}
			}
			return nestedRows;
		};
		const rows = [];
		for (const p of properties) {
			if (p.type.name === 'Object') {
				const nestedRows = getNestedRows(p);
				rows.push(nestedRows);
				continue;
			} else {
				let name = p.type.name;
				if (name === 'Array') {
					name = 'Array (of ' + p.type.itemType.name + ' elements)';
				}
				if ('enum' in p.type) {
					name += ' (enum with values: ' + p.type.enum.join(', ') + ')';
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
