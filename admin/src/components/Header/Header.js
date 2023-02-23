import './Header.css';
import Flex from '../Flex/Flex';
import { SlSelect, SlMenuItem, SlAvatar } from '@shoelace-style/shoelace/dist/react/index.js';

const Header = () => {
	return (
		<>
			<Flex className='Header' justifyContent='space-between' alignItems='center'>
				<SlSelect name='workspaceSelector' value='1'>
					<SlMenuItem value='1' selected>
						Mock workspace 1
					</SlMenuItem>
					<SlMenuItem value='2'>Mock workspace 2</SlMenuItem>
				</SlSelect>
				<div className='account'>
					<sl-icon name='bell-fill'></sl-icon>
					<SlAvatar image='data:image/jpeg;base64,/9j/' />
				</div>
			</Flex>
		</>
	);
};

export default Header;
