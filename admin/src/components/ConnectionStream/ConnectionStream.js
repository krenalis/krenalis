import { useState, useEffect } from 'react';
import './ConnectionStream.css';
import FlexContainer from '../FlexContainer/FlexContainer';
import call from '../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionStream = ({ connection: c, onConnectionChange, onError }) => {
	let [streams, setStreams] = useState([]);
	let [showStreams, setShowStreams] = useState(false);

	useEffect(() => {
		const fetchStreams = async () => {
			let [connections, err] = await call('/admin/connections/find', 'GET');
			if (err) {
				onError(err);
				return;
			}
			let eventStreams = [];
			for (let cn of connections) {
				if (cn.Type === 'EventStream' && cn.Role === c.Role) {
					eventStreams.push(cn);
				}
			}
			setStreams(eventStreams);
		};
		fetchStreams();
	}, []);

	const onChangeStream = async (stream) => {
		let [, err] = await call(`/api/connections/${c.ID}/stream/${stream}`, 'PUT');
		if (err !== null) {
			onError(err);
			setShowStreams(false);
			return;
		}
		let cn = { ...c };
		cn.Stream = stream;
		setShowStreams(false);
		onConnectionChange(cn);
	};

	const onRemoveStream = async () => {
		let [, err] = await call(`/api/connections/${c.ID}/stream/0`, 'PUT');
		if (err !== null) {
			onError(err);
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
