import { useState, useEffect, useContext } from 'react';
import './ConnectionStream.css';
import FlexContainer from '../FlexContainer/FlexContainer';
import { AppContext } from '../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import statuses from '../../constants/statuses';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionStream = ({ connection: c, onConnectionChange }) => {
	let [streams, setStreams] = useState([]);
	let [showStreams, setShowStreams] = useState(false);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchStreams = async () => {
			let [connections, err] = await API.connections.find();
			if (err) {
				showError(err);
				return;
			}
			let streams = [];
			for (let cn of connections) {
				if (cn.Type === 'Stream' && cn.Role === c.Role) {
					streams.push(cn);
				}
			}
			setStreams(streams);
		};
		fetchStreams();
	}, []);

	const onChangeStream = async (stream) => {
		let [, err] = await API.connections.setStorage(c.ID, stream);
		setShowStreams(false);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'StreamNotExist') {
					showStatus(statuses.streamNotExist);
				}
				return;
			}
			showError(err);
			return;
		}
		let cn = { ...c };
		cn.Stream = stream;
		onConnectionChange(cn);
	};

	const onRemoveStream = async () => {
		let [, err] = await API.connections.setStorage(c.ID, 0);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		let cn = { ...c };
		cn.Stream = 0;
		onConnectionChange(cn);
	};

	let currentStream = streams.find((s) => s.ID === c.Stream);
	let dialogStreams = streams.filter((s) => s.ID !== c.Stream);

	return (
		<>
			{currentStream && (
				<>
					<FlexContainer className='streamContainer' alignItems='center' gap={30}>
						<div className='stream'>{currentStream.Name}</div>
						<SlButton variant='danger' onClick={onRemoveStream}>
							<SlIcon slot='prefix' name='x' />
							Remove
						</SlButton>
					</FlexContainer>
				</>
			)}
			<SlButton variant='neutral' onClick={() => setShowStreams(true)}>
				<SlIcon slot='prefix' name={c.Stream === 0 ? 'plus' : 'pencil-fill'} />
				{c.Stream === 0 ? 'Add a stream' : 'Change the stream'}
			</SlButton>
			<SlDialog
				className='streamsDialog'
				open={showStreams}
				style={{ '--width': '600px' }}
				onSlAfterHide={() => setShowStreams(false)}
				label={`Select a stream`}
			>
				{dialogStreams.length === 0 ? (
					<div className='noStream'>No Stream available</div>
				) : (
					dialogStreams.map((s) => (
						<FlexContainer className='stream' alignItems='center' justifyContent='space-between' gap={20}>
							<div className='name'>{s.Name}</div>
							<SlButton
								variant='primary'
								onClick={async () => {
									await onChangeStream(s.ID);
								}}
								className='changeStreamButton'
							>
								<SlIcon name='arrow-right' />
							</SlButton>
						</FlexContainer>
					))
				)}
			</SlDialog>
		</>
	);
};

export default ConnectionStream;
