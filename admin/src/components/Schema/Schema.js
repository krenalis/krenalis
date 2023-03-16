import { useState, useEffect, useContext } from 'react';
import './Schema.css';
import PrimaryBackground from '../PrimaryBackground/PrimaryBackground';
import Header from '../Header/Header';
import HeadedGrid from '../HeadedGrid/HeadedGrid';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { UnprocessableError } from '../../api/errors';

const Schema = () => {
	let [properties, setProperties] = useState([]);
	let [isLoading, setIsLoading] = useState(false);

	let { API, showError, showStatus } = useContext(AppContext);

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
		setIsLoading(true);
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
		setProperties(schema.properties);
		setTimeout(() => {
			setIsLoading(false);
			showStatus(statuses.schemasReloaded);
		}, 300);
	};

	const getPropertiesRows = (properties) => {
		const getNestedRows = (p) => {
			let nestedRows = [[p.name, p.type.name]];
			for (let pr of p.type.properties) {
				if (pr.type.name === 'Object') {
					let nr = getNestedRows(pr);
					nestedRows.push(nr);
				} else {
					let name = pr.type.name;
					if (name === 'Array' && 'itemType' in pr.type) {
						name = 'Array (of ' + pr.type.itemType.name + ' elements)';
					}
					if ('enum' in pr.type) {
						name += ' (enum with values: ' + pr.type.enum.join(', ') + ')';
					}
					nestedRows.push([pr.name, name]);
				}
			}
			return nestedRows;
		};
		let rows = [];
		for (let p of properties) {
			if (p.type.name === 'Object') {
				let nestedRows = getNestedRows(p);
				rows.push(nestedRows);
			} else {
				let name = p.type.name;
				if (name === 'Array' && 'itemType' in p.type) {
					name = 'Array (of ' + p.type.itemType.name + ' elements)';
				}
				if ('enum' in p.type) {
					name += ' (enum with values: ' + p.type.enum.join(', ') + ')';
				}
				let row = [p.name, name];
				rows.push(row);
			}
		}
		return rows;
	};

	let columns = [{ Name: 'name' }, { Name: 'type' }];
	let rows = getPropertiesRows(properties);

	return (
		<div className='Schema'>
			<PrimaryBackground height={250} overlap={100}>
				<Header />
			</PrimaryBackground>
			<div className='routeContent'>
				<HeadedGrid columns={columns} rows={rows} title='Golden Record schema' isLoading={isLoading}>
					<SlButton className='reloadSchemas' variant='default' onClick={onReloadSchemas}>
						<SlIcon name='arrow-clockwise' slot='prefix'></SlIcon>
						Reload Schemas
					</SlButton>
				</HeadedGrid>
			</div>
		</div>
	);
};

export default Schema;
