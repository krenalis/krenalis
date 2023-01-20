import { useState, useEffect, useRef } from 'react';
import './Schema.css';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Header from '../../components/Header/Header';
import HeadedGrid from '../../components/HeadedGrid/HeadedGrid';
import Toast from '../../components/Toast/Toast';
import call from '../../utils/call';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Schema = () => {
	let [properties, setProperties] = useState([]);
	let [isLoading, setIsLoading] = useState(false);
	let [status, setStatus] = useState(null);

	let toastRef = useRef();

	const onError = (err) => {
		setTimeout(() => {
			setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
			toastRef.current.toast();
			setIsLoading(false);
		}, 500);
	};

	useEffect(() => {
		const fetchSchema = async () => {
			let [schema, err] = await call('/admin/user-schema', 'GET');
			if (err != null) {
				onError(err);
				return;
			}
			setProperties(schema.properties);
			setTimeout(() => setIsLoading(false), 500);
		};
		setIsLoading(true);
		fetchSchema();
	}, []);

	const onReloadSchema = async () => {
		let err;
		setIsLoading(true);
		[, err] = await call('/api/workspace/reload-schemas', 'POST');
		if (err != null) {
			setProperties([]);
			onError(err);
			return;
		}
		let schema;
		[schema, err] = await call('/admin/user-schema', 'GET');
		if (err != null) {
			setProperties([]);
			onError(err);
			return;
		}
		setProperties(schema.properties);
		setTimeout(() => {
			setIsLoading(false);
			setStatus({ variant: 'success', icon: 'check2-circle', text: 'The schema has been reloaded successfully' });
			toastRef.current.toast();
		}, 500);
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
					if (name == 'Array' && 'itemType' in pr.type) {
						name = 'Array (of ' + pr.type.itemType.name +' elements)'
					}
					if ('enum' in pr.type) {
						name += " (enum with values: " + pr.type.enum.join(', ') + ")"
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
				if (name == 'Array' && 'itemType' in p.type) {
					name = 'Array (of ' + p.type.itemType.name +' elements)'
				}
				if ('enum' in p.type) {
					name += " (enum with values: " + p.type.enum.join(', ') + ")"
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
				<Toast reactRef={toastRef} status={status} />
				<HeadedGrid columns={columns} rows={rows} title='Golden Record schema' isLoading={isLoading}>
					<SlButton className='reloadSchema' variant='default' onClick={onReloadSchema}>
						<SlIcon name='arrow-clockwise' slot='prefix'></SlIcon>
						Reload Schema
					</SlButton>
				</HeadedGrid>
			</div>
		</div>
	);
};

export default Schema;
