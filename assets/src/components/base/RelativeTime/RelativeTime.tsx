import React from 'react';
import './RelativeTime.css';
import SlRelativeTime from '@shoelace-style/shoelace/dist/react/relative-time/index.js';

interface RelativeTimeProps {
	date: Date | string;
}

const RelativeTime = ({ date }: RelativeTimeProps) => {
	let d: Date;

	if (typeof date === 'string') {
		d = new Date(date);
	} else {
		d = date;
	}

	return <SlRelativeTime date={d} lang='en-US' title={d.toLocaleString()} />;
};

export { RelativeTime };
