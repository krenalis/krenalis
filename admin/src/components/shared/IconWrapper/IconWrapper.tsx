import React from 'react';
import './IconWrapper.css';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface IconWrapperProps {
	name: string;
	size?: number;
	moat?: boolean;
}

const IconWrapper = ({ name, size, moat }: IconWrapperProps) => {
	return (
		<div
			className={`iconWrapper${moat ? ' moat' : ''}`}
			style={{ '--icon-size': size ? `${size}px` : '16px' } as React.CSSProperties}
		>
			<div className='innerWrapper'>
				<SlIcon name={name}></SlIcon>
			</div>
		</div>
	);
};

export default IconWrapper;
