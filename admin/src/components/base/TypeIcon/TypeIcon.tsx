import React, { ReactNode } from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface TypeIconProps {
	kind: string;
	slot?: string;
}

const TypeIcon = ({ kind, slot }: TypeIconProps) => {
	let icon: ReactNode = null;
	let s = slot !== undefined ? slot : null;
	if (kind === 'boolean') {
		icon = <SlIcon slot={s} name='type-bold' />;
	} else if (kind === 'int' || kind === 'uint' || kind === 'float' || kind === 'decimal' || kind === 'year') {
		icon = <SlIcon slot={s} name='123' />;
	} else if (kind === 'datetime' || kind === 'date') {
		icon = <SlIcon slot={s} name='calendar-date' />;
	} else if (kind === 'time') {
		icon = <SlIcon slot={s} name='clock' />;
	} else if (kind === 'uuid' || kind === 'ip' || kind === 'string') {
		icon = <SlIcon slot={s} name='fonts' />;
	} else if (kind === 'json') {
		icon = <SlIcon slot={s} name='filetype-json' />;
	} else if (kind === 'array') {
		icon = <SlIcon slot={s} name='input-cursor' />;
	} else if (kind === 'object') {
		icon = <SlIcon slot={s} name='braces' />;
	} else if (kind === 'map') {
		icon = <SlIcon slot={s} name='braces-asterisk' />;
	}
	return icon;
};

export { TypeIcon };
