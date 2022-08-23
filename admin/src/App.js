import './components/table';
import './App.css';

let columns = [
	{
		code: 'id',
		title: 'ID',
		width: 100,
	},
	{
		code: 'browser',
		title: 'Browser',
		width: 400,
	},
	{
		code: 'eventType',
		title: 'Event type',
		width: 250,
	},
	{
		code: 'language',
		title: 'Language',
		width: 200,
	},
	{
		code: 'referrer',
		title: 'Referrer',
		width: 200,
	},
	{
		code: 'session',
		title: 'Session',
		width: 250,
	},
	{
		code: 'target',
		title: 'Target',
		width: 400,
	},
	{
		code: 'text',
		title: 'Text',
		width: 300,
	},
	{
		code: 'timestamp',
		title: 'Timestamp',
		width: 500,
	},
	{
		code: 'title',
		title: 'Title',
		width: 300,
	},
	{
		code: 'url',
		title: 'URL',
		width: 400,
	},
];

function App() {
	return (
		<div className="App">
			<div className="page">
				<o2b-table
					title="Ultimi Eventi"
					description="Gli ultimi eventi registrati sulla tua web app"
					query="SELECT * FROM `events` LIMIT 5"
					columns={JSON.stringify(columns)}
				></o2b-table>
			</div>
		</div>
	);
}

export default App;
