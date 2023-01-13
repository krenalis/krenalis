import './NavigationTabs.css';
import { NavLink } from 'react-router-dom';

const NavigationTabs = ({ tabs, onAccent }) => {
	return (
		<div className={`NavigationTabs${onAccent ? ' onAccent' : ''}`}>
			{tabs.map((t) => {
				return (
					<div className={`tab${t.Selected ? ' selected' : ''}`}>
						{t.Name}
						<NavLink to={t.Link}></NavLink>
					</div>
				);
			})}
		</div>
	);
};

export default NavigationTabs;
