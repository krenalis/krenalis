import call from './call.js'

const url = 'https://localhost:9090/api/visualization'

export default async function getChardData(query) {
    let [entries, error] = await call(url, query);
    if (error != null) {
        return [null, error];
    };
    let chartData = [];
    for (let e of entries) {
        chartData.push({ 'name': String(e[0]), 'p': e[1] });
    }
    return [chartData, null];
}