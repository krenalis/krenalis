import React, { ReactNode } from 'react';
import SlIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface OverlayRowProps {
	children: ReactNode;
}

const OverlayRow = ({ children }: OverlayRowProps) => {
	return (
		<div className='overlay-row'>
			<button className='overlay-row__handle'>
				<SlIcon name='grip-vertical' />
			</button>
			<div className='overlay-row__name'>{children}</div>
		</div>
	);
};

export { OverlayRow };
