export default async function call(url, value) {
    let body, res;
    if (value) body = JSON.stringify(value);
    try {
        res = body ? await fetch(url, { method: 'POST', body: body }) : await fetch(url);
    } catch (err) {
        return [null, `error while fetching ${url}: ${err.message}`];
    }

    if (res.status !== 200) {
        let error;
        switch (res.status) {
            case 500:
                error = 'Internal Server Error';
                break;
            case 400:
                error = 'Bad Request';
                break;
            default:
                error = "Unknown Server Error";
                break;
        }
        return [null, error];
    }

    let data;
    try {
        data = await res.json();
    } catch (err) {
        if (err.message === 'Unexpected end of JSON input') return [null, null];
        return [null, `error while parsing json response from ${url}: ${err.message}`];
    }

    return [data, null];
}
