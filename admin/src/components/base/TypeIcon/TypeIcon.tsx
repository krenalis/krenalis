import React, { ReactNode } from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface TypeIconProps {
	kind: string;
	slot?: string;
}

const TypeIcon = ({ kind, slot }: TypeIconProps) => {
	let icon: ReactNode = null;
	let s = slot !== undefined ? slot : null;
	if (kind === 'boolean') {
		icon = <WaIcon slot={s} name='type-bold' />;
	} else if (kind === 'int' || kind === 'float' || kind === 'decimal' || kind === 'year') {
		icon = <WaIcon slot={s} name='123' />;
	} else if (kind === 'datetime' || kind === 'date') {
		icon = <WaIcon slot={s} name='calendar-date' />;
	} else if (kind === 'time') {
		icon = <WaIcon slot={s} name='clock' />;
	} else if (kind === 'uuid' || kind === 'ip' || kind === 'string') {
		icon = <WaIcon slot={s} name='fonts' />;
	} else if (kind === 'json') {
		icon = <WaIcon slot={s} name='filetype-json' />;
	} else if (kind === 'array') {
		icon = <WaIcon slot={s} name='input-cursor' />;
	} else if (kind === 'object') {
		icon = <WaIcon slot={s} name='braces' />;
	} else if (kind === 'map') {
		icon = <WaIcon slot={s} name='braces-asterisk' />;
	}
	return icon;
};

export { TypeIcon };
