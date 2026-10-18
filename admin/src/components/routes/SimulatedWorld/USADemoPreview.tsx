import React, { useContext, useEffect, useState } from 'react';
import AppContext from '../../../context/AppContext';
import type { USADemoPreview as USADemoPreviewRecord } from '../../../lib/api/types/simulatedSystem';
import './USADemoPreview.css';

interface USADemoPreviewProps {
	accountID: string;
}

const USADemoPreview = ({ accountID }: USADemoPreviewProps) => {
	const { api } = useContext(AppContext);
	const [records, setRecords] = useState<USADemoPreviewRecord[] | null>(null);
	const [error, setError] = useState('');

	useEffect(() => {
		const controller = new AbortController();
		setRecords(null);
		setError('');
		void api.workspaces.simulatedSystems.usaPreview(accountID, controller.signal).then(
			(value) => {
				if (!controller.signal.aborted) setRecords(value);
			},
			(err: unknown) => {
				if (!controller.signal.aborted) setError(err instanceof Error ? err.message : String(err));
			},
		);
		return () => controller.abort();
	}, [api, accountID]);

	return (
		<section className='usa-demo-preview' aria-label='United States internal demo'>
			<h2>United States</h2>
			<p>Internal demo · Synthetic people with AI-generated photographs.</p>
			{error !== '' && <p role='alert'>{error}</p>}
			{records == null && error === '' && <p>Loading people…</p>}
			{records != null && records.length === 0 && <p>No generated records yet.</p>}
			<div className='usa-demo-preview__people'>
				{records?.map((record) => (
					<article className='usa-demo-preview__person' key={record.id}>
						<img
							src={record.photo}
							alt={`AI-generated portrait of ${record.name}`}
							width={128}
							height={128}
						/>
						<h3>{record.name}</h3>
						<p>{record.location}</p>
						<p>
							{record.education} · {record.primaryActivity}
						</p>
						<p>
							{new Intl.NumberFormat('en-US', {
								style: 'currency',
								currency: 'USD',
								maximumFractionDigits: 0,
							}).format(record.annualIncomeUSD)}{' '}
							/ year
						</p>
					</article>
				))}
			</div>
		</section>
	);
};

export default USADemoPreview;
