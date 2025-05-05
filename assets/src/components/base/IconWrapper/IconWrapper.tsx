import React from 'react';
import './IconWrapper.css';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface IconWrapperProps {
	name: string;
	size?: number;
	moat?: boolean;
	onClick?: (e: any) => void;
}

const IconWrapper = ({ name, size, moat, onClick }: IconWrapperProps) => {
	return (
		<div
			className={`icon-wrapper${moat ? ' icon-wrapper--moat' : ''}`}
			style={
				{ '--icon-size': size ? `${size}px` : '16px', cursor: onClick ? 'pointer' : '' } as React.CSSProperties
			}
			onClick={onClick}
		>
			<div className='icon-wrapper__inner-wrapper'>
				<SlIcon name={name}></SlIcon>
			</div>
		</div>
	);
};

export default IconWrapper;
