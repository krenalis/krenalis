import { useState, useEffect, useContext } from 'react';
import './Schema.css';
import StyledGrid from '../StyledGrid/StyledGrid';
import Toolbar from '../Toolbar/Toolbar';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import statuses from '../../constants/statuses';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { UnprocessableError } from '../../api/errors';

const Schema = () => {
	let [properties, setProperties] = useState([]);
	let [isLoading, setIsLoading] = useState(true);

	let { API, showError, showStatus } = useContext(AppContext);

	let { setCurrentTitle } = useContext(NavigationContext);

	setCurrentTitle('Golden Record schema');

	useEffect(() => {
		const fetchSchema = async () => {
			let [schema, err] = await API.workspace.userSchema();
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
		[, err] = await API.workspace.reloadSchemas();
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
		[schema, err] = await API.workspace.userSchema();
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
			let nestedRows = [{ cells: [p.name, p.type.name] }];
			for (let pr of p.type.properties) {
				if (pr.type.name === 'Object') {
					let nr = getNestedRows(pr);
					nestedRows.push(nr);
					continue;
				} else {
					let name = pr.type.name;
					if (name === 'Array' && 'itemType' in pr.type) {
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
		let rows = [];
		for (let p of properties) {
			if (p.type.name === 'Object') {
				let nestedRows = getNestedRows(p);
				rows.push(nestedRows);
				continue;
			} else {
				let name = p.type.name;
				if (name === 'Array' && 'itemType' in p.type) {
					name = 'Array (of ' + p.type.itemType.name + ' elements)';
				}
				if ('enum' in p.type) {
					name += ' (enum with values: ' + p.type.enum.join(', ') + ')';
				}
				let row = { cells: [p.name, name] };
				rows.push(row);
			}
		}
		return rows;
	};

	let columns = [{ name: 'name' }, { name: 'type' }];
	let rows = getPropertiesRows(properties);

	return (
		<div className='Schema'>
			<Toolbar>
				<SlButton className='reloadSchemas' variant='default' onClick={onReloadSchemas}>
					<SlIcon name='arrow-clockwise' slot='prefix'></SlIcon>
					Reload Schemas
				</SlButton>
			</Toolbar>
			<div className='routeContent schema'>
				<StyledGrid columns={columns} rows={rows} isLoading={isLoading}></StyledGrid>
			</div>
		</div>
	);
};

export default Schema;
