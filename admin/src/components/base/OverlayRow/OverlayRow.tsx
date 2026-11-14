import React, { ReactNode } from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface OverlayRowProps {
	children: ReactNode;
}

const OverlayRow = ({ children }: OverlayRowProps) => {
	return (
		<div className='overlay-row'>
			<button className='overlay-row__handle'>
				<WaIcon name='grip-vertical' />
			</button>
			<div className='overlay-row__name'>{children}</div>
		</div>
	);
};

export { OverlayRow };
