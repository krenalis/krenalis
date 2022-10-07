import call from '../../../utils/call.js'

const url = 'https://localhost:9090/admin/api/visualization'

export default async function getChardData(query) {
    let [res, error] = await call(url, query);
    if (error != null) {
        return [null, error];
    };
    let chartData = [];
    for (let entry of res.Data) {
        chartData.push({ 'name': String(entry[0]), 'p': entry[1] });
    }
    let response = {
        'chartData': chartData,
        'columns': res.Columns,
        'sqlQuery': res.Query,
    }
    return [response, null];
}