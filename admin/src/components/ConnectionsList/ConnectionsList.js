import { useState, useEffect, useContext } from 'react';
import './ConnectionsList.css';
import IconWrapper from '../IconWrapper/IconWrapper';
import StyledGrid from '../StyledGrid/StyledGrid';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import StatusDot from '../StatusDot/StatusDot';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import { NavigationContext } from '../../context/NavigationContext';
import { AppContext } from '../../context/AppContext';
import { SlSpinner, SlTooltip, SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { useNavigate } from 'react-router-dom';

const GRID_COLUMNS = [
	{
		name: 'Name',
	},
	{
		name: 'Type',
	},
	{
		name: 'Connector',
	},
	{
		name: 'Health',
	},
	{
		name: 'Actions count',
	},
];

const ConnectionsList = () => {
	let [connectionsRows, setConnectionsRows] = useState([]);
	let [isLoading, setIsLoading] = useState(true);
	let [role, setRole] = useState('');

	let navigate = useNavigate();

	let { setCurrentTitle, setCurrentRoute, setPreviousRoute } = useContext(NavigationContext);
	let { API, showError, connectors } = useContext(AppContext);

	useEffect(() => {
		const fetchConnections = async () => {
			let [connections, err] = await API.connections.find();
			if (err) {
				showError(err);
				setTimeout(() => setIsLoading(false), 300);
				return;
			}
			let roleConnections = [];
			for (let c of connections) {
				if (c.Role === role) {
					roleConnections.push(c);
				}
			}
			if (roleConnections.length === 0) {
				setConnectionsRows([]);
				setTimeout(() => setIsLoading(false), 300);
				return;
			}

			let rows = [];
			for (let c of roleConnections) {
				let connector = connectors.find((con) => con.ID === c.Connector);
				let logo;
				if (connector.Icon === '') {
					logo = <UnknownLogo size={21} />;
				} else {
					logo = <LittleLogo icon={connector.Icon} />;
				}
				let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);
				rows.push({
					cells: [
						<div className='connectionNameCell'>
							{logo} {c.Name}
						</div>,
						c.Type,
						connector.Name,
						<SlTooltip content={statusText}>
							<StatusDot statusText={statusText} statusVariant={statusVariant} />
						</SlTooltip>,
						c.ActionsCount,
					],
					onClick: () => {
						navigate(`/admin/connections/${c.ID}/actions`);
					},
				});
			}
			setConnectionsRows(rows);
			setTimeout(() => setIsLoading(false), 300);
		};
		setCurrentTitle(`${role}s`);
		setIsLoading(true);
		fetchConnections();
	}, [role]);

	let path = window.location.pathname;
	let splitted = path.split('/');
	let urlRole = splitted[splitted.length - 1];
	setCurrentRoute(`connections/${urlRole}`);
	setPreviousRoute('connections');
	let r;
	if (urlRole === 'sources') {
		r = 'Source';
	} else {
		r = 'Destination';
	}
	if (r !== role) {
		setRole(r);
	}

	return (
		<div className='connectionsList'>
			<div className='routeContent'>
				{isLoading ? (
					<SlSpinner
						className='connectionsListSpinner'
						style={{
							fontSize: '3rem',
							'--track-width': '6px',
						}}
					></SlSpinner>
				) : connectionsRows.length === 0 ? (
					<div className='noConnection'>
						<IconWrapper name={role === 'Source' ? 'file-arrow-down' : 'file-arrow-up'} size={40} />
						<div className='noConnectionText'>You don't have any {role.toLowerCase()} installed</div>
						<SlButton
							variant='primary'
							onClick={() => {
								navigate(`/admin/connectors?role=${role}`);
							}}
						>
							Add a {role.toLowerCase()}...
						</SlButton>
					</div>
				) : (
					<>
						<SlButton
							variant='text'
							className='addNewConnection'
							onClick={() => {
								navigate(`/admin/connectors?role=${role}`);
							}}
						>
							<SlIcon slot='suffix' name='plus-circle' />
							Add a new {role.toLowerCase()}
						</SlButton>
						<StyledGrid columns={GRID_COLUMNS} rows={connectionsRows} />
					</>
				)}
			</div>
		</div>
	);
};

export default ConnectionsList;
